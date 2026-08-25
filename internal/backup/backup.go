// Package backup implements khayal vault/database/config backups with
// optional age encryption, and additive-merge restores.
package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// Config describes one backup or restore operation.
type Config struct {
	VaultPath   string // vault root (copied as <dest>/vault)
	DBPath      string // live khayal.db
	ConfigPath  string // live config.yaml
	DestPath    string // backup directory (backup) / source directory (restore)
	Encrypt     bool   // encrypt db + config with the key at KeyPath
	KeyPath     string // age identity file
	Date        string // restore: pick a specific YYYY-MM-DD backup instead of latest
	Overwrite   bool   // restore: overwrite existing db/config files
	SkipVault   bool   // restore: only restore db + config
}

// Result reports what an operation did.
type Result struct {
	BackupPath string
	VaultFiles int
	VaultSize  int64
	DBSize     int64
	ConfigSize int64
	Duration   time.Duration
	DBDate     string // restore: date of the db backup that was used
}

const (
	dbFilePrefix     = "khayal-"
	configFilePrefix = "config-"
	vaultDirName     = "vault"
)

func dateStamp() string { return time.Now().Format("2006-01-02") }

// Backup copies the vault tree and snapshots db + config into DestPath.
// When Encrypt is set, db and config are written as armored age files.
func Backup(cfg Config) (*Result, error) {
	if cfg.DestPath == "" {
		return nil, fmt.Errorf("destination path is required")
	}
	start := time.Now()

	vaultDest := filepath.Join(cfg.DestPath, vaultDirName)
	if err := os.MkdirAll(vaultDest, 0755); err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}

	files, size, err := copyTree(cfg.VaultPath, vaultDest)
	if err != nil {
		return nil, fmt.Errorf("vault backup failed: %w", err)
	}

	dbSize, err := snapshotFile(cfg.DBPath, filepath.Join(cfg.DestPath, dbFilePrefix+dateStamp()+".db"), cfg)
	if err != nil {
		return nil, fmt.Errorf("database backup failed: %w", err)
	}

	configSize, err := snapshotFile(cfg.ConfigPath, filepath.Join(cfg.DestPath, configFilePrefix+dateStamp()+".yaml"), cfg)
	if err != nil {
		return nil, fmt.Errorf("config backup failed: %w", err)
	}

	return &Result{
		BackupPath: cfg.DestPath,
		VaultFiles: files,
		VaultSize:  size,
		DBSize:     dbSize,
		ConfigSize: configSize,
		Duration:   time.Since(start),
	}, nil
}

// Restore decrypts/copies the chosen db + config backups back into place
// and additively merges the vault copy — never deletes existing notes.
func Restore(cfg Config) (*Result, error) {
	if cfg.DestPath == "" {
		return nil, fmt.Errorf("--from is required")
	}
	start := time.Now()

	res := &Result{BackupPath: cfg.DestPath}

	if !cfg.SkipVault {
		vaultSrc := filepath.Join(cfg.DestPath, vaultDirName)
		files, size, err := mergeTree(vaultSrc, cfg.VaultPath, cfg.Overwrite)
		if err != nil {
			return nil, fmt.Errorf("vault restore failed: %w", err)
		}
		res.VaultFiles = files
		res.VaultSize = size
	}

	date := cfg.Date
	if date == "" {
		date = findLatestDate(cfg.DestPath)
		if date == "" {
			return nil, fmt.Errorf("no database backups found in %s", cfg.DestPath)
		}
	}
	res.DBDate = date

	dbSrc, err := resolveBackupFile(cfg.DestPath, dbFilePrefix, ".db", date)
	if err != nil {
		return nil, err
	}
	dbSize, err := restoreFile(dbSrc, cfg.DBPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("database restore failed: %w", err)
	}
	res.DBSize = dbSize

	configSrc, err := resolveBackupFile(cfg.DestPath, configFilePrefix, ".yaml", date)
	if err != nil {
		// Config restore is best-effort: a missing config backup should not
		// abort a successful database restore.
		configSrc = ""
	}
	if configSrc != "" {
		if _, err := os.Stat(cfg.ConfigPath); err == nil && !cfg.Overwrite {
			fmt.Fprintf(os.Stderr, "warning: %s exists — config not restored (use --overwrite)\n", cfg.ConfigPath)
		} else {
			configSize, err := restoreFile(configSrc, cfg.ConfigPath, cfg)
			if err != nil {
				return nil, fmt.Errorf("config restore failed: %w", err)
			}
			res.ConfigSize = configSize
		}
	}

	res.Duration = time.Since(start)
	return res, nil
}

// GenerateKey creates a new age X25519 identity file at keyPath.
func GenerateKey(keyPath string) error {
	if _, err := os.Stat(keyPath); err == nil {
		return fmt.Errorf("key already exists at %s (refusing to overwrite)", keyPath)
	}
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("generate identity: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return fmt.Errorf("create key directory: %w", err)
	}
	content := fmt.Sprintf("# created: %s\n# public key: %s\n%s\n",
		time.Now().Format(time.RFC3339), identity.Recipient().String(), identity.String())
	if err := os.WriteFile(keyPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("write key file: %w", err)
	}
	return nil
}

func loadIdentity(keyPath string) (*age.X25519Identity, error) {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key file %s: %w", keyPath, err)
	}
	// Key files carry # comments; find the identity line.
	var lastErr error
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		identity, err := age.ParseX25519Identity(line)
		if err != nil {
			lastErr = err
			continue
		}
		return identity, nil
	}
	return nil, fmt.Errorf("parse key file %s: %w", keyPath, lastErr)
}

// snapshotFile copies src to dst, encrypted when configured.
func snapshotFile(src, dst string, cfg Config) (int64, error) {
	info, err := os.Stat(src)
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", src, err)
	}
	if cfg.Encrypt {
		dst += ".age"
	}
	if info.IsDir() {
		return 0, fmt.Errorf("%s is a directory", src)
	}
	if cfg.Encrypt {
		return encryptFile(src, dst, cfg.KeyPath)
	}
	return copyFile(src, dst)
}

// restoreFile decrypts/copies a backup file back to dst.
func restoreFile(src, dst string, cfg Config) (int64, error) {
	if strings.HasSuffix(src, ".age") {
		if _, err := os.Stat(dst); err == nil && !cfg.Overwrite {
			return 0, fmt.Errorf("%s exists (use --overwrite)", dst)
		}
		return decryptFileSize(src, dst, cfg.KeyPath)
	}
	if _, err := os.Stat(dst); err == nil && !cfg.Overwrite {
		return 0, fmt.Errorf("%s exists (use --overwrite)", dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return 0, err
	}
	out, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	n, err := io.Copy(out, in)
	if err != nil {
		return 0, err
	}
	return n, out.Sync()
}

func encryptFile(src, dst, keyPath string) (int64, error) {
	identity, err := loadIdentity(keyPath)
	if err != nil {
		return 0, err
	}
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	armoredWriter := armor.NewWriter(out)
	encryptedWriter, err := age.Encrypt(armoredWriter, identity.Recipient())
	if err != nil {
		return 0, fmt.Errorf("init encryption: %w", err)
	}
	if _, err := io.Copy(encryptedWriter, in); err != nil {
		return 0, err
	}
	if err := encryptedWriter.Close(); err != nil {
		return 0, err
	}
	if err := armoredWriter.Close(); err != nil {
		return 0, err
	}
	return fileSize(dst)
}

func decryptFileSize(src, dst, keyPath string) (int64, error) {
	identity, err := loadIdentity(keyPath)
	if err != nil {
		return 0, err
	}
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	var ciphertext io.Reader = in
	if looksArmored(src) {
		ciphertext = armor.NewReader(ciphertext)
	}

	decrypted, err := age.Decrypt(ciphertext, identity)
	if err != nil {
		return 0, fmt.Errorf("decrypt: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return 0, err
	}
	out, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	n, err := io.Copy(out, decrypted)
	if err != nil {
		return 0, err
	}
	return n, out.Sync()
}

// looksArmored sniffs for the PEM-style armor header regardless of extension.
func looksArmored(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 64)
	n, _ := f.Read(buf)
	return strings.HasPrefix(string(buf[:n]), "-----BEGIN AGE ENCRYPTED FILE-----")
}

// copyTree recursively copies src under dst, skipping symlinks. Returns
// file count and total bytes.
func copyTree(src, dst string) (int, int64, error) {
	info, err := os.Stat(src)
	if err != nil {
		return 0, 0, err
	}
	if !info.IsDir() {
		return 0, 0, fmt.Errorf("%s is not a directory", src)
	}

	count, size := 0, int64(0)
	err = filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		n, err := copyFile(path, target)
		if err != nil {
			return err
		}
		count++
		size += n
		return nil
	})
	return count, size, err
}

// mergeTree copies everything from src into dst without deleting anything
// already present — the additive-merge restore contract. Existing files are
// skipped unless overwrite is set.
func mergeTree(src, dst string, overwrite bool) (int, int64, error) {
	info, err := os.Stat(src)
	if err != nil {
		return 0, 0, nil // empty/no vault dir: nothing to merge
	}
	if !info.IsDir() {
		return 0, 0, fmt.Errorf("%s is not a directory", src)
	}

	count, size := 0, int64(0)
	err = filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if _, statErr := os.Stat(target); statErr == nil && !overwrite {
			return nil
		}
		n, err := copyFile(path, target)
		if err != nil {
			return err
		}
		count++
		size += n
		return nil
	})
	return count, size, err
}

// findLatestDate returns the newest YYYY-MM-DD that has a khayal-<date>.db
// backup (plain or .age) in dir.
func findLatestDate(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	best := ""
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, dbFilePrefix) {
			continue
		}
		rest := strings.TrimPrefix(name, dbFilePrefix)
		rest = strings.TrimSuffix(rest, ".age")
		if !strings.HasSuffix(rest, ".db") {
			continue
		}
		date := strings.TrimSuffix(rest, ".db")
		if len(date) != 10 || date > "9" || date[4] != '-' {
			continue
		}
		if date > best {
			best = date
		}
	}
	return best
}

// resolveBackupFile locates the db/config backup for a date, preferring the
// encrypted variant when both exist.
func resolveBackupFile(dir, prefix, ext, date string) (string, error) {
	for _, name := range []string{prefix + date + ext + ".age", prefix + date + ext} {
		cand := filepath.Join(dir, name)
		if _, err := os.Stat(cand); err == nil {
			return cand, nil
		}
	}
	return "", fmt.Errorf("no %s%s backup for %s found in %s", prefix, ext, date, dir)
}

func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
