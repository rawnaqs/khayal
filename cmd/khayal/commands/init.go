package commands

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "First-run setup — generates config.yaml + token",
		Long: `Initialize khayal for first use.

Creates the config directory, generates a secure token,
writes config.yaml with secure permissions, and creates
the log directory.

The token is shown ONCE and cannot be recovered.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing config")

	return cmd
}

func runInit(force bool) error {
	configDir := config.ExpandTilde("~/.config/khayal")
	configPath := filepath.Join(configDir, "config.yaml")
	tokenPath := filepath.Join(configDir, "token")
	logDir := filepath.Join(configDir, "logs")

	fmt.Printf("creating config directory... %s\n", configDir)
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

	token := generateToken()
	fmt.Printf("generating token...       %s... (save this — shown once)\n", token[:16])

	defaultConfig := `# khayal configuration
vault:
  path: ~/Documents/brain
  inbox_dir: khayal

server:
  host: 127.0.0.1
  port: 1133
  token: ""

llm:
  provider: ollama
  ollama_host: http://localhost:11434
  embed_model: nomic-embed-text
  text_model: qwen2.5:3b
  vision_model: moondream

worker:
  max_workers: 1
  max_retries: 3

db:
  path: ~/.config/khayal/khayal.db

log:
  level: info
  worker_level: info
  file: ~/.config/khayal/logs/khayal.log
`

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Printf("writing config...         %s (600)\n", configPath)

	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to write token: %w", err)
	}
	fmt.Printf("writing token...          %s (600)\n", tokenPath)

	fmt.Println()
	fmt.Println("next steps:")
	fmt.Println("  1. edit ~/.config/khayal/config.yaml")
	fmt.Println("  2. set vault.path to your notes directory")
	fmt.Println("  3. set server.token to your desired token")
	fmt.Println("  4. start khayal:     khayal start")

	return nil
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
