package commands

import (
	"fmt"
	"path/filepath"

	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"

	"github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/backup"
	"github.com/rawnaqs/khayal/internal/config"
)

var (
	restoreFrom      string
	restoreDate      string
	restoreOverwrite bool
)

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore from backup",
		RunE:  runRestore,
	}
	cmd.Flags().StringVar(&restoreFrom, "from", "", "Backup source path (required)")
	cmd.Flags().StringVar(&restoreDate, "date", "", "Specific backup date (YYYY-MM-DD), default: latest")
	cmd.Flags().BoolVar(&restoreOverwrite, "overwrite", false, "Overwrite existing database and config files")
	return cmd
}

func runRestore(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	if restoreFrom == "" {
		cli.Fatal(cli.ExitUser, "--from is required")
		return fmt.Errorf("--from is required")
	}

	if cli.IsRunning() {
		fmt.Println(theme.ErrorStyle.Render("✗ khayal must be stopped before restore"))
		fmt.Println(theme.Muted.Render("  → khayal stop"))
		fmt.Println(theme.Muted.Render(fmt.Sprintf("  → then: khayal restore --from %s", restoreFrom)))
		return fmt.Errorf("khayal is running")
	}

	vaultPath := config.MakeAbsolute(cfg.Vault.Path, configPath)
	dbPath := config.MakeAbsolute(cfg.DB.Path, configPath)
	cfgPath := config.MakeAbsolute(configPath, "")

	res, err := backup.Restore(backup.Config{
		VaultPath:  vaultPath,
		DBPath:     dbPath,
		ConfigPath: cfgPath,
		DestPath:   restoreFrom,
		Date:       restoreDate,
		Overwrite:  restoreOverwrite,
		KeyPath:    defaultKeyPath(),
	})
	if err != nil {
		cli.Fatal(cli.ExitServer, "%v", err)
		return err
	}

	fmt.Println()
	fmt.Println(theme.Bold.Render("restoring vault... (additive merge)"))
	fmt.Printf("  %s → %s\n", theme.Muted.Render(filepath.Join(res.BackupPath, "vault")+"/"), theme.Primary.Render(vaultPath))
	fmt.Printf("  %s files · %s\n", theme.Dim.Render(fmt.Sprintf("%d", res.VaultFiles)), theme.Dim.Render(formatSize(res.VaultSize)))

	fmt.Println(theme.Bold.Render("restoring database..."))
	fmt.Printf("  %s → khayal.db · %s\n",
		theme.Muted.Render("khayal-"+res.DBDate+".db"), theme.Dim.Render(formatSize(res.DBSize)))

	if res.ConfigSize > 0 {
		fmt.Println(theme.Bold.Render("restoring config..."))
	}

	fmt.Println()
	fmt.Printf("%s · %s\n",
		theme.SuccessStyle.Render("restore complete"),
		theme.Dim.Render(fmt.Sprintf("%.1fs", res.Duration.Seconds())))
	fmt.Println(theme.Muted.Render("  → start again: ") + theme.Primary.Render("khayal start"))
	return nil
}
