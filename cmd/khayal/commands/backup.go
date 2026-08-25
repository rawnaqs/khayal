package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"

	"github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/backup"
	"github.com/rawnaqs/khayal/internal/config"
)

var (
	backupDest    string
	backupEncrypt bool
	backupInitKey bool
)

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup vault, database, and config",
		RunE:  runBackup,
	}
	cmd.Flags().StringVar(&backupDest, "dest", "", "Destination path for backup (required)")
	cmd.Flags().BoolVar(&backupEncrypt, "encrypt", false, "Encrypt database and config")
	cmd.Flags().BoolVar(&backupInitKey, "init-key", false, "Generate the backup encryption key, then exit")
	return cmd
}

func runBackup(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	keyPath := defaultKeyPath()

	if backupInitKey {
		if err := backup.GenerateKey(keyPath); err != nil {
			cli.Fatal(cli.ExitUser, "%v", err)
			return err
		}
		fmt.Println(theme.SuccessStyle.Render("✓") + " " + theme.Muted.Render("key generated:") + " " + theme.Primary.Render(keyPath))
		fmt.Println(theme.Muted.Render("  public key is in the file header — share it only with backup writers"))
		return nil
	}

	if backupDest == "" {
		cli.Fatal(cli.ExitUser, "--dest is required")
		return fmt.Errorf("--dest is required")
	}

	vaultPath := config.MakeAbsolute(cfg.Vault.Path, configPath)
	dbPath := config.MakeAbsolute(cfg.DB.Path, configPath)
	cfgPath := config.MakeAbsolute(configPath, "")

	if cli.IsRunning() {
		fmt.Println(theme.WarningStyle.Render("⚠ khayal is running — vault copy is safe, but the database snapshot may catch a mid-write state"))
		fmt.Println(theme.Muted.Render("  → for a consistent db: khayal stop && khayal backup ... && khayal start"))
	}

	dest := backupDest
	if !filepath.IsAbs(dest) {
		dest, _ = filepath.Abs(dest)
	}

	res, err := backup.Backup(backup.Config{
		VaultPath:  vaultPath,
		DBPath:     dbPath,
		ConfigPath: cfgPath,
		DestPath:   dest,
		Encrypt:    backupEncrypt,
		KeyPath:    keyPath,
	})
	if err != nil {
		cli.Fatal(cli.ExitServer, "%v", err)
		return err
	}

	fmt.Println()
	fmt.Println(theme.Bold.Render("backing up vault..."))
	fmt.Printf("  %s → %s\n", theme.Muted.Render(vaultPath), theme.Primary.Render(filepath.Join(dest, "vault")))
	fmt.Printf("  %s files · %s\n", theme.Dim.Render(fmt.Sprintf("%d", res.VaultFiles)), theme.Dim.Render(formatSize(res.VaultSize)))

	fmt.Println(theme.Bold.Render("backing up database..."))
	fmt.Printf("  khayal.db → %s · %s%s\n",
		theme.Muted.Render("khayal-"+dbStamp()+".db"),
		theme.Dim.Render(formatSize(res.DBSize)), encMarker(backupEncrypt))

	fmt.Println(theme.Bold.Render("backing up config..."))
	fmt.Printf("  config.yaml → %s%s\n",
		theme.Muted.Render("config-"+dbStamp()+".yaml"), encMarker(backupEncrypt))

	fmt.Println()
	fmt.Printf("%s %s · %s\n",
		theme.SuccessStyle.Render("backup complete"),
		theme.Dim.Render(formatSize(res.VaultSize+res.DBSize+res.ConfigSize)),
		theme.Dim.Render(fmt.Sprintf("%.1fs", res.Duration.Seconds())))
	fmt.Println(theme.Muted.Render("dest: ") + theme.Primary.Render(res.BackupPath))
	return nil
}

func encMarker(on bool) string {
	if on {
		return " · " + theme.SuccessStyle.Render("encrypted ✓")
	}
	return ""
}

func dbStamp() string { return time.Now().Format("2006-01-02") }

func defaultKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".khayal-backup.key"
	}
	return filepath.Join(home, ".config", "khayal", "backup.key")
}
