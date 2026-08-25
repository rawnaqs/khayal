package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedKey(t *testing.T) string {
	t.Helper()
	keyPath := filepath.Join(t.TempDir(), "backup.key")
	if err := GenerateKey(keyPath); err != nil {
		t.Fatal(err)
	}
	return keyPath
}

func seedSource(t *testing.T, dbBody, configBody string) (vaultDir, dbPath, configPath string) {
	t.Helper()
	vaultDir = t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "khayal"), 0755)
	os.WriteFile(filepath.Join(vaultDir, "khayal", "note-a.md"), []byte("---\ntitle: A\n---\nalpha body"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "khayal", "note-b.md"), []byte("---\ntitle: B\n---\nbeta body"), 0644)

	dbPath = filepath.Join(t.TempDir(), "khayal.db")
	os.WriteFile(dbPath, []byte(dbBody), 0644)

	configPath = filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(configPath, []byte(configBody), 0644)
	return
}

func TestGenerateKeyRefusesOverwrite(t *testing.T) {
	key := seedKey(t)
	if err := GenerateKey(key); err == nil {
		t.Error("expected refusal on existing key")
	}
	b, _ := os.ReadFile(key)
	if !strings.Contains(string(b), "AGE-SECRET-KEY-1") {
		t.Error("key file missing identity line")
	}
}

func TestBackupAndRestoreRoundTripPlain(t *testing.T) {
	vaultDir, dbPath, configPath := seedSource(t, "DB-BODY-V1", "config-v1")
	dest := t.TempDir()

	res, err := Backup(Config{
		VaultPath: vaultDir, DBPath: dbPath, ConfigPath: configPath,
		DestPath: dest, KeyPath: seedKey(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.VaultFiles != 2 || res.DBSize == 0 || res.ConfigSize == 0 {
		t.Fatalf("unexpected result: %+v", res)
	}

	// restore into fresh locations
	newVault := t.TempDir()
	newDB := filepath.Join(newVault, "..", "restored.db")
	newCfg := filepath.Join(t.TempDir(), "restored-config.yaml")

	got, err := Restore(Config{
		VaultPath: newVault, DBPath: newDB, ConfigPath: newCfg,
		DestPath: dest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.DBDate == "" {
		t.Error("expected DBDate set")
	}
	body, err := os.ReadFile(newDB)
	if err != nil || string(body) != "DB-BODY-V1" {
		t.Errorf("db round trip failed: %q err=%v", string(body), err)
	}
	cfgBody, _ := os.ReadFile(newCfg)
	if string(cfgBody) != "config-v1" {
		t.Errorf("config round trip failed: %q", string(cfgBody))
	}
	noteA, err := os.ReadFile(filepath.Join(newVault, "khayal", "note-a.md"))
	if err != nil || !strings.Contains(string(noteA), "alpha") {
		t.Errorf("vault merge missing note-a: %v", err)
	}
}

func TestBackupAndRestoreEncryptedCycle(t *testing.T) {
	key := seedKey(t)
	vaultDir, dbPath, configPath := seedSource(t, "SECRET-DB", "SECRET-CONFIG")
	dest := t.TempDir()

	if _, err := Backup(Config{
		VaultPath: vaultDir, DBPath: dbPath, ConfigPath: configPath,
		DestPath: dest, Encrypt: true, KeyPath: key,
	}); err != nil {
		t.Fatal(err)
	}

	dbAge, err := os.Open(filepath.Join(dest, dbFilePrefix+dateStamp()+".db.age"))
	if err != nil {
		t.Fatal(err)
	}
	defer dbAge.Close()
	buf := make([]byte, 64)
	n, _ := dbAge.Read(buf)
	if !strings.HasPrefix(string(buf[:n]), "-----BEGIN AGE ENCRYPTED FILE-----") {
		t.Error("expected armored age ciphertext")
	}

	// plaintext must not exist
	if _, err := os.Stat(filepath.Join(dest, dbFilePrefix+dateStamp()+".db")); !os.IsNotExist(err) {
		t.Error("plaintext db leaked alongside encrypted backup")
	}

	newDB := filepath.Join(t.TempDir(), "out.db")
	newCfg := filepath.Join(t.TempDir(), "out.yaml")
	if _, err := Restore(Config{
		VaultPath: t.TempDir(), DBPath: newDB, ConfigPath: newCfg,
		DestPath: dest, SkipVault: true, Overwrite: true, KeyPath: key,
	}); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(newDB)
	if string(body) != "SECRET-DB" {
		t.Errorf("decrypted db mismatch: %q", string(body))
	}
	cfgBody, _ := os.ReadFile(newCfg)
	if string(cfgBody) != "SECRET-CONFIG" {
		t.Errorf("decrypted config mismatch: %q", string(cfgBody))
	}
}

func TestRestoreMergeIsAdditive(t *testing.T) {
	vaultDir, dbPath, configPath := seedSource(t, "db", "cfg")
	dest := t.TempDir()

	if _, err := Backup(Config{
		VaultPath: vaultDir, DBPath: dbPath, ConfigPath: configPath,
		DestPath: dest, KeyPath: seedKey(t),
	}); err != nil {
		t.Fatal(err)
	}

	// live vault gains a newer note that the backup lacks
	liveVault := t.TempDir()
	if err := os.MkdirAll(filepath.Join(liveVault, "khayal"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(liveVault, "khayal", "note-a.md"), []byte("newer alpha"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(liveVault, "khayal", "new-note.md"), []byte("brand new"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := Restore(Config{
		VaultPath: liveVault, DBPath: filepath.Join(t.TempDir(), "x.db"),
		ConfigPath: filepath.Join(t.TempDir(), "y.yaml"),
		DestPath:   dest, SkipVault: false, Overwrite: true,
		KeyPath: seedKey(t),
	}); err != nil {
		t.Fatal(err)
	}

	// new-note survived the merge
	if _, err := os.Stat(filepath.Join(liveVault, "khayal", "new-note.md")); err != nil {
		t.Error("additive merge deleted a newer note")
	}
}

func TestFindLatestDate(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"khayal-2026-01-01.db", "khayal-2026-03-05.db.age",
		"khayal-2026-02-02.db", "config-2026-03-05.yaml", "random.txt",
	} {
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644)
	}
	if got := findLatestDate(dir); got != "2026-03-05" {
		t.Errorf("got %q", got)
	}
	if got := findLatestDate(t.TempDir()); got != "" {
		t.Errorf("empty dir: got %q", got)
	}
}

func TestResolveBackupPrefersEncrypted(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "khayal-2026-01-01.db"), []byte("plain"), 0644)
	os.WriteFile(filepath.Join(dir, "khayal-2026-01-01.db.age"), []byte("cipher"), 0644)

	got, err := resolveBackupFile(dir, dbFilePrefix, ".db", "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, ".age") {
		t.Errorf("expected encrypted variant, got %s", got)
	}
	if _, err := resolveBackupFile(dir, dbFilePrefix, ".db", "1999-01-01"); err == nil {
		t.Error("expected error for missing date")
	}
}
