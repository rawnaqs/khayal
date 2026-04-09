# Phase 5: Backup and Restore

> Backup and restore vault, database, and config. Updated: 2026-04-09

## Goals

- [ ] Add age encryption dependency (pinned version)
- [ ] Create backup package
- [ ] backup command
- [ ] restore command
- [ ] Test backup/restore cycle
- [ ] Unit tests

## Dependency

**Pin to version for stability:**

```bash
go get filippo.io/age@v1.2.0
go mod tidy
```

Note: Pin to v1.2.0. Check https://pkg.go.dev/filippo.io/age for current version.

## Specification

Per SPEC.md (lines 483-591):

### What Gets Backed Up

| Item | Method | Encryption |
|------|--------|------------|
| vault/ | rsync/copy | Plain (user's choice) |
| khayal.db | snapshot | Optional with `--encrypt` |
| config.yaml | copy | Optional with `--encrypt` |

### Encryption

Uses `filippo.io/age` (Go library, embedded).

### Output

```
khayal backup --dest /Volumes/BackupDrive/khayal

  backing up vault...
    ~/brain → /Volumes/BackupDrive/khayal/vault/
    2,847 files · 124 MB · 8s

  backing up database...
    khayal.db → khayal-2024-03-16.db
    24 MB · encrypted ✓

  backing up config...
    config.yaml → config-2024-03-16.yaml
    encrypted ✓

  backup complete · 148 MB · 14s
  dest: /Volumes/BackupDrive/khayal/
```

## Step 5.1: Add Dependency

```bash
go get filippo.io/age@latest
go mod tidy
```

## Step 5.2: Create Backup Package

**File:** `internal/backup/backup.go`

```go
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

type Config struct {
    VaultPath    string
    DBPath       string
    ConfigPath  string
    DestPath    string
    Encrypt    bool
    KeyPath    string
}

type Result struct {
    BackupPath   string
    VaultSize   int64
    DBSize      int64
    ConfigSize int64
    Duration   time.Duration
}

func Backup(cfg Config) (*Result, error) {
    start := time.Now()

    vaultDest := filepath.Join(cfg.DestPath, "vault")
    dbDest := filepath.Join(cfg.DestPath, fmt.Sprintf("khayal-%s.db", time.Now().Format("2006-01-02")))
    configDest := filepath.Join(cfg.DestPath, fmt.Sprintf("config-%s.yaml", time.Now().Format("2006-01-02")))

    // Create destination directory
    os.MkdirAll(cfg.DestPath, 0755)
    os.MkdirAll(vaultDest, 0755)

    // Backup vault (copy all files)
    vaultSize, err := copyVault(cfg.VaultPath, vaultDest)
    if err != nil {
        return nil, fmt.Errorf("vault backup failed: %w", err)
    }

    // Backup database
    var dbSize int64
    if cfg.Encrypt {
        dbSize, err = encryptFile(cfg.DBPath, dbDest, cfg.KeyPath)
    } else {
        dbSize, err = copyFile(cfg.DBPath, dbDest)
    }
    if err != nil {
        return nil, fmt.Errorf("database backup failed: %w", err)
    }

    // Backup config
    var configSize int64
    if cfg.Encrypt {
        configSize, err = encryptFile(cfg.ConfigPath, configDest, cfg.KeyPath)
    } else {
        configSize, err = copyFile(cfg.ConfigPath, configDest)
    }
    if err != nil {
        return nil, fmt.Errorf("config backup failed: %w", err)
    }

    return &Result{
        BackupPath: cfg.DestPath,
        VaultSize: vaultSize,
        DBSize:   dbSize,
        ConfigSize: configSize,
        Duration: time.Since(start),
    }, nil
}

func Restore(cfg Config) (*Result, error) {
    start := time.Now()

    // Find latest backup files
    vaultSrc := filepath.Join(cfg.DestPath, "vault")
    dbSrc := findLatestFile(filepath.Join(cfg.DestPath, "khayal-*.db"))
    configSrc := findLatestFile(filepath.Join(cfg.DestPath, "config-*.yaml"))

    // Check if encrypted
    dbEncrypted := strings.HasSuffix(dbSrc, ".age")
    if dbEncrypted {
        if err := decryptFile(dbSrc, cfg.DBPath, cfg.KeyPath); err != nil {
            return nil, fmt.Errorf("database restore failed: %w", err)
        }
    } else {
        if err := copyFile(dbSrc, cfg.DBPath); err != nil {
            return nil, fmt.Errorf("database restore failed: %w", err)
        }
    }

    configEncrypted := strings.HasSuffix(configSrc, ".age")
    if configEncrypted {
        if err := decryptFile(configSrc, cfg.ConfigPath, cfg.KeyPath); err != nil {
            return nil, fmt.Errorf("config restore failed: %w", err)
        }
    } else {
        if err := copyFile(configSrc, cfg.ConfigPath); err != nil {
            return nil, fmt.Errorf("config restore failed: %w", err)
        }
    }

    // Restore vault (additive merge)
    vaultSize, err := mergeVault(vaultSrc, cfg.VaultPath)
    if err != nil {
        return nil, fmt.Errorf("vault restore failed: %w", err)
    }

    return &Result{
        BackupPath: cfg.DestPath,
        VaultSize: vaultSize,
        Duration:  time.Since(start),
    }, nil
}

func copyVault(src, dst string) (int64, error) {
    var total int64

    entries, err := os.ReadDir(src)
    if err != nil {
        return 0, err
    }

    for _, e := range entries {
        if e.IsDir() && e.Name() == ".khayal-trash" {
            continue
        }

        srcPath := filepath.Join(src, e.Name())
        dstPath := filepath.Join(dst, e.Name())

        if e.IsDir() {
            os.MkdirAll(dstPath, 0755)
            subTotal, _ := copyVault(srcPath, dstPath)
            total += subTotal
        } else {
            size, err := copyFile(srcPath, dstPath)
            if err != nil {
                return total, err
            }
            total += size
        }
    }

    return total, nil
}

func mergeVault(src, dst string) (int64, error) {
    var total int64

    entries, err := os.ReadDir(src)
    if err != nil {
        return 0, err
    }

    for _, e := range entries {
        srcPath := filepath.Join(src, e.Name())
        dstPath := filepath.Join(dst, e.Name())

        // Skip if exists in destination (additive merge)
        if _, err := os.Stat(dstPath); err == nil {
            continue
        }

        if e.IsDir() {
            os.MkdirAll(dstPath, 0755)
            subTotal, _ := mergeVault(srcPath, dstPath)
            total += subTotal
        } else {
            size, err := copyFile(srcPath, dstPath)
            if err != nil {
                return total, err
            }
            total += size
        }
    }

    return total, nil
}

func copyFile(src, dst string) (int64, error) {
    srcFile, err := os.Open(src)
    if err != nil {
        return 0, err
    }
    defer srcFile.Close()

    dstFile, err := os.Create(dst)
    if err != nil {
        return 0, err
    }
    defer dstFile.Close()

    written, err := io.Copy(dstFile, srcFile)
    if err != nil {
        return 0, err
    }

    return written, dstFile.Chmod(0644)
}

func encryptFile(src, dst, keyPath string) (int64, error) {
    // Load key
    identity, err := age.ParseX25519IdentityFile(keyPath)
    if err != nil {
        return 0, err
    }

    // Read input
    input, err := os.ReadFile(src)
    if err != nil {
        return 0, err
    }

    // Encrypt
    encrypted, err := identity.Encrypt(input)
    if err != nil {
        return 0, err
    }

    // Write with armor
    armored := armor.Serialize(encrypted)
    err = os.WriteFile(dst+".age", armored, 0644)
    if err != nil {
        return 0, err
    }

    stat, _ := os.Stat(dst + ".age")
    return stat.Size(), nil
}

func decryptFile(src, dst, keyPath string) error {
    // Load key
    identity, err := age.ParseX25519IdentityFile(keyPath)
    if err != nil {
        return err
    }

    // Read armored input
    input, err := os.ReadFile(src)
    if err != nil {
        return err
    }

    // Parse armor
    ciphertext, err := armor.Parse(bytes.NewReader(input))
    if err != nil {
        return err
    }

    // Decrypt
    decrypted, err := identity.Decrypt(ciphertext)
    if err != nil {
        return err
    }

    return os.WriteFile(dst, decrypted, 0644)
}

func findLatestFile(pattern string) string {
    pattern = strings.ReplaceAll(pattern, "*", "")

    entries, _ := os.ReadDir(filepath.Dir(pattern))
    var latest string
    var latestTime time.Time

    for _, e := range entries {
        if e.IsDir() {
            continue
        }
        matched, _ := filepath.Match(filepath.Base(pattern), e.Name())
        if !matched {
            continue
        }

        info, _ := e.Info()
        if info.ModTime().After(latestTime) {
            latestTime = info.ModTime()
            latest = filepath.Join(filepath.Dir(pattern), e.Name())
        }
    }

    return latest
}

func generateKey(keyPath string) error {
    identity, err := age.GenerateX25519Identity()
    if err != nil {
        return err
    }

    return os.WriteFile(keyPath, []byte(identity.String()), 0600)
}
```

## Step 5.3: Create backup Command

**File:** `cmd/khayal/commands/backup.go`

```go
package commands

import (
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/spf13/cobra"
    "github.com/rawnaqs/khayal/internal/backup"
    "github.com/rawnaqs/khayal/internal/config"
    "github.com/rawnaqs/khayal/pkg/theme"
)

var backupDest string
var backupEncrypt bool

var backupCmd = &cobra.Command{
    Use:   "backup",
    Short: "Backup vault, database, and config",
    RunE:  runBackup,
}

func init() {
    backupCmd.Flags().StringVar(&backupDest, "dest", "", "Destination path for backup")
    backupCmd.Flags().BoolVar(&backupEncrypt, "encrypt", false, "Encrypt database and config")
    rootCmd.AddCommand(backupCmd)
}

func runBackup(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    if backupDest == "" {
        return fmt.Errorf("--dest required")
    }

    // Check if khayal is stopped (or warn)
    if isRunning() {
        fmt.Println(theme.Muted.Render("! khayal should be stopped before backup"))
    }

    vaultPath := cfg.Vault.Path
    dbPath := cfg.DB.Path
    configPath := filepath.Join(filepath.Dir(dbPath), "config.yaml")

    fmt.Println(theme.Bold.Render("backing up vault..."))
    fmt.Printf("  %s → %s\n", theme.Muted.Render(vaultPath), theme.Primary.Render(backupDest+"/vault/"))

    bk := backup.BackupConfig{
        VaultPath:   vaultPath,
        DBPath:      dbPath,
        ConfigPath:  configPath,
        DestPath:   backupDest,
        Encrypt:   backupEncrypt,
        KeyPath:   filepath.Join(filepath.Dir(dbPath), "backup.key"),
    }

    result, err := backup.Backup(bk)
    if err != nil {
        return err
    }

    vaultSize := formatSize(result.VaultSize)
    dbSize := formatSize(result.DBSize)
    configSize := formatSize(result.ConfigSize)

    fmt.Println()
    fmt.Println(theme.Bold.Render("backing up database..."))
    fmt.Printf("  %s → %s\n", theme.Muted.Render("khayal.db"), theme.Primary.Render("khayal-"+time.Now().Format("2006-01-02")+".db"))
    if backupEncrypt {
        fmt.Printf("  %s\n", theme.SuccessStyle.Render("✓ encrypted"))
    }

    fmt.Println()
    fmt.Println(theme.Bold.Render("backing up config..."))
    fmt.Printf("  %s → %s\n", theme.Muted.Render("config.yaml"), theme.Primary.Render("config-"+time.Now().Format("2006-01-02")+".yaml"))
    if backupEncrypt {
        fmt.Printf("  %s\n", theme.SuccessStyle.Render("✓ encrypted"))
    }

    totalSize := result.VaultSize + result.DBSize + result.ConfigSize

    fmt.Println()
    fmt.Printf("%s · %s · %s\n",
        theme.SuccessStyle.Render("✓ backup complete"),
        theme.Primary.Render(formatSize(totalSize)),
        theme.Muted.Render(result.Duration.Round(time.Second).String()))
    fmt.Printf("  dest: %s\n", theme.Muted.Render(backupDest))

    return nil
}

func formatSize(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    if bytes < unit*unit {
        return fmt.Sprintf("%.1f KB", float64(bytes)/unit)
    }
    if bytes < unit*unit*unit {
        return fmt.Sprintf("%.1f MB", float64(bytes)/(unit*unit))
    }
    return fmt.Sprintf("%.1f GB", float64(bytes)/(unit*unit*unit))
}

func isRunning() bool {
    // Check if server is running
    pidPath := filepath.Join(os.Getenv("HOME"), ".config", "khayal", "khayal.pid")
    if _, err := os.Stat(pidPath); os.IsNotExist(err) {
        return false
    }
    data, _ := os.ReadFile(pidPath)
    // Try to read PID and check if process exists
    // Simplified: just check if pid file exists
    return true
}
```

## Step 5.4: Create restore Command

**File:** `cmd/khayal/commands/restore.go`

```go
package commands

import (
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/spf13/cobra"
    "github.com/rawnaqs/khayal/internal/backup"
    "github.com/rawnaqs/khayal/internal/config"
    "github.com/rawnaqs/khayal/pkg/theme"
)

var restoreFrom string
var restoreDate string
var restoreOverwrite bool

var restoreCmd = &cobra.Command{
    Use:   "restore",
    Short: "Restore from backup",
    RunE:  runRestore,
}

func init() {
    restoreCmd.Flags().StringVar(&restoreFrom, "from", "", "Backup source path")
    restoreCmd.Flags().StringVar(&restoreDate, "date", "", "Specific backup date (YYYY-MM-DD)")
    restoreCmd.Flags().BoolVar(&restoreOverwrite, "overwrite", false, "Overwrite existing files")
    rootCmd.AddCommand(restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    if restoreFrom == "" {
        return fmt.Errorf("--from required")
    }

    // MUST be stopped
    if isRunning() {
        fmt.Println(theme.ErrorStyle.Render("✗ khayal must be stopped before restore"))
        fmt.Println(theme.Muted.Render("  → run: khayal stop"))
        fmt.Println(theme.Muted.Render("  → then: khayal restore --from " + restoreFrom))
        return nil
    }

    // Show available backups
    fmt.Println(theme.Bold.Render("available backups"))

    // List backups in source
    entries, _ := os.ReadDir(restoreFrom)
    dates := []string{}

    for _, e := range entries {
        if e.IsDir() && len(e.Name()) == 10 {
            dates = append(dates, e.Name())
        }
    }

    if len(dates) == 0 {
        return fmt.Errorf("no backups found in %s", restoreFrom)
    }

    // Show latest
    latest := dates[len(dates)-1]
    fmt.Printf("  %s  vault: ? notes · db: ?  %s\n",
        theme.Primary.Render(latest),
        theme.Muted.Render("← latest"))

    fmt.Println()
    fmt.Printf("  %s\n", theme.Primary.Render("restoring latest backup..."))

    vaultPath := cfg.Vault.Path
    dbPath := cfg.DB.Path
    configPath := filepath.Join(filepath.Dir(dbPath), "config.yaml")

    fmt.Println()
    fmt.Println(theme.Bold.Render("restoring vault..."))
    fmt.Printf("  %s → %s\n", theme.Muted.Render(restoreFrom+"/vault/"), theme.Primary.Render(vaultPath))

    restoreCfg := backup.RestoreConfig{
        VaultPath: vaultPath,
        DBPath:    dbPath,
        ConfigPath: configPath,
        DestPath: restoreFrom,
        Overwrite: restoreOverwrite,
        KeyPath:  filepath.Join(filepath.Dir(dbPath), "backup.key"),
    }

    result, err := backup.Restore(restoreCfg)
    if err != nil {
        return err
    }

    fmt.Println()
    fmt.Println(theme.Bold.Render("restoring database..."))
    fmt.Printf("  %s → %s\n", theme.Muted.Render("khayal-"+latest+".db"), theme.Primary.Render(dbPath))
    if _, err := os.Stat(restoreFrom + "/khayal-" + latest + ".db.age"); err == nil {
        fmt.Printf("  %s\n", theme.SuccessStyle.Render("✓ decrypting"))
    }

    fmt.Println()
    fmt.Println(theme.Bold.Render("restoring config..."))
    fmt.Printf("  %s → %s\n", theme.Muted.Render("config-"+latest+".yaml"), theme.Primary.Render(configPath))

    fmt.Println()
    fmt.Printf("%s\n", theme.SuccessStyle.Render("✓ restore complete"))
    fmt.Printf("  %s\n", theme.Muted.Render("run: khayal start"))

    return nil
}
```

## Step 5.5: Create backup key Init

Add to backup command:

```go
var backupInitKey bool

func init() {
    backupCmd.Flags().BoolVar(&backupInitKey, "init-key", false, "Generate backup encryption key")
}
```

And in runBackup:

```go
if backupInitKey {
    keyPath := filepath.Join(os.Getenv("HOME"), ".config", "khayal", "backup.key")
    if err := backup.GenerateKey(keyPath); err != nil {
        return err
    }
    fmt.Printf("✓ key generated → %s\n", theme.Muted.Render(keyPath))
    return nil
}
```

## Checklist

- [ ] age dependency added (v1.2.0 pinned)
- [ ] backup package
- [ ] backup command --dest
- [ ] backup command --encrypt
- [ ] backup command --init-key
- [ ] restore command --from
- [ ] restore command --date
- [ ] restore command --overwrite
- [ ] Test backup
- [ ] Test restore
- [ ] Test encrypted backup/restore
- [ ] Unit tests

## Next Phase

[Phase 6: Polish](phase-6-polish.md)

## Notes

- --encrypt uses age library (embedded, no external binary)
- Backup key stored at ~/.config/khayal/backup.key
- Restore refuses if khayal is running
- Vault restore is additive merge (per SPEC.md)
- Backup format: khayal-YYYY-MM-DD.db, config-YYYY-MM-DD.yaml
- Tests cover: backup, restore, encryption

## Step 5.X: Unit Tests

**File:** `internal/backup/backup_test.go`

```go
package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test content")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	size, err := copyFile(src, dst)
	if err != nil {
		t.Errorf("copyFile() error = %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("size = %d, want %d", size, len(content))
	}

	// Check destination
	dstContent, _ := os.ReadFile(dst)
	if string(dstContent) != string(content) {
		t.Error("destination content mismatch")
	}
}

func TestFindLatestFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different timestamps
	file1 := filepath.Join(tmpDir, "khayal-2024-01-01.db")
	file2 := filepath.Join(tmpDir, "khayal-2024-01-15.db")
	file3 := filepath.Join(tmpDir, "khayal-2024-01-10.db")

	os.WriteFile(file1, []byte("1"), 0644)
	os.WriteFile(file2, []byte("2"), 0644)
	os.WriteFile(file3, []byte("3"), 0644)

	// Touch to order
	os.Chtimes(file1, time.Now(), time.Now().AddDate(0, 0, -14))
	os.Chtimes(file2, time.Now(), time.Now().AddDate(0, 0, -1))
	os.Chtimes(file3, time.Now(), time.Now().AddDate(0, 0, -7))

	latest := findLatestFile(filepath.Join(tmpDir, "khayal-*.db"))
	if !contains(latest, "khayal-2024-01-15.db") {
		t.Errorf("expected latest to be khayal-2024-01-15.db, got %s", latest)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatSize(tt.input)
		if result != tt.expected {
			t.Errorf("formatSize(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
```

**File:** `cmd/khayal/commands/backup_test.go`

```go
package commands

import (
	"testing"
)

func TestBackup_DestPath(t *testing.T) {
	// Test that --dest is required
	cmd := backupCmd

	if err := cmd.Run(nil); err == nil {
		t.Error("expected error for missing --dest")
	}
}

func TestRestore_FromPath(t *testing.T) {
	// Test that --from is required
	cmd := restoreCmd

	if err := cmd.Run(nil); err == nil {
		t.Error("expected error for missing --from")
	}
}
```