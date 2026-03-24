package commands

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		force     bool
		vaultPath string
		token     string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "First-run setup — generates config.yaml + token",
		Long: `Initialize khayal for first use.

Creates the config directory, writes config.yaml with your vault path
and a secure auth token, and creates the log directory.

The token is shown ONCE and cannot be recovered from the config.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(force, vaultPath, token)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing config")
	cmd.Flags().StringVar(&vaultPath, "vault", "", "vault path (your notes directory)")
	cmd.Flags().StringVar(&token, "token", "", "auth token (auto-generated if empty)")

	return cmd
}

func runInit(force bool, vaultPath, token string) error {
	configPath := cli.ConfigPath()
	configDir := cli.ConfigDir()
	logDir := filepath.Join(configDir, "logs")

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.MkdirAll(logDir, 0700); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	if _, err := os.Stat(configPath); err == nil && !force {
		fmt.Printf("config already exists: %s\n", configPath)
		fmt.Println("use --force to overwrite")
		return nil
	}

	// Get vault path: flag → interactive prompt
	if vaultPath == "" {
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Vault path").
					Description("Where your notes will be stored").
					Prompt(" vault: ").
					Value(&vaultPath),
			),
		).Run(); err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}

		if vaultPath == "" {
			return fmt.Errorf("vault path is required")
		}
	}

	// Get token: flag → auto-generate
	if token == "" {
		token = generateToken()
	}
	tokenDisplay := token
	if len(tokenDisplay) > 16 {
		tokenDisplay = tokenDisplay[:16] + "..."
	}

	// Build config with token embedded
	defaultConfig := fmt.Sprintf(`# khayal configuration
vault:
  path: %s
  inbox_dir: khayal

server:
  host: %s
  port: %d
  token: %s

llm:
  provider: ollama
  ollama_host: %s
  embed_model: nomic-embed-text
  text_model: qwen2.5:3b
  vision_model: moondream

worker:
  max_workers: 1
  max_retries: 3

db:
  path: khayal.db

log:
  level: info
  worker_level: info
  file: logs/khayal.log
`,
		vaultPath,
		config.DefaultServerHost,
		config.DefaultServerPort,
		token,
		config.DefaultOllamaHost,
	)

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println()
	fmt.Printf("  ✓ config: %s\n", configPath)
	fmt.Printf("  ✓ token:  %s (shown once)\n", tokenDisplay)
	fmt.Println()
	fmt.Println("  get started:")
	fmt.Printf("    khayal start\n")
	fmt.Printf("    kl init --token %s\n", token)

	return nil
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
