package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/theme"
)

// ConfigPath returns the config file path, respecting KHAYAL_CONFIG env var.
func ConfigPath() string {
	cfgPath := os.Getenv("KHAYAL_CONFIG")
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath
	}
	return config.ExpandTilde(cfgPath)
}

// ConfigDir returns the directory containing the config file.
func ConfigDir() string {
	abs, _ := filepath.Abs(ConfigPath())
	return filepath.Dir(abs)
}

func LoadConfig() (*config.Config, string, error) {
	cfg, absCfgPath, err := config.LoadFromPath(ConfigPath())
	if err != nil {
		return nil, "", err
	}

	return cfg, absCfgPath, nil
}

func EnsureDirectories(cfg *config.Config, configPath string) error {
	absCfgPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	return cfg.EnsureDirectories(absCfgPath)
}

func ViewConfig(cfg *config.Config, configPath string) {
	keyStyle := theme.Muted.Width(16)

	fmt.Println()
	fmt.Println(theme.Bold.Render("vault"))
	fmt.Printf("  %s %s\n", keyStyle.Render("path"), theme.Primary.Render(config.MakeAbsolute(cfg.Vault.Path, configPath)))
	fmt.Printf("  %s %s\n", keyStyle.Render("inbox_dir"), theme.Primary.Render(cfg.Vault.InboxDir))
	fmt.Printf("  %s %s\n", keyStyle.Render("media_dir"), theme.Primary.Render(cfg.Vault.Media.DefaultDir))
	fmt.Println()

	fmt.Println(theme.Bold.Render("server"))
	fmt.Printf("  %s %s\n", keyStyle.Render("host"), theme.Primary.Render(cfg.Server.Host))
	fmt.Printf("  %s %s\n", keyStyle.Render("port"), theme.Primary.Render(fmt.Sprintf("%d", cfg.Server.Port)))
	if cfg.Server.Token != "" {
		fmt.Printf("  %s %s\n", keyStyle.Render("token"), theme.Dim.Render("..."))
	}
	fmt.Println()

	fmt.Println(theme.Bold.Render("llm"))
	fmt.Printf("  %s %s\n", keyStyle.Render("provider"), theme.Primary.Render(cfg.LLM.Provider))
	fmt.Printf("  %s %s\n", keyStyle.Render("ollama_host"), theme.Primary.Render(cfg.LLM.OllamaHost))
	fmt.Printf("  %s %s\n", keyStyle.Render("embed_model"), theme.Primary.Render(cfg.LLM.EmbedModel))
	fmt.Printf("  %s %s\n", keyStyle.Render("text_model"), theme.Primary.Render(cfg.LLM.TextModel))
	fmt.Printf("  %s %s\n", keyStyle.Render("vision_model"), theme.Primary.Render(cfg.LLM.VisionModel))
	fmt.Println()

	fmt.Println(theme.Bold.Render("worker"))
	fmt.Printf("  %s %s\n", keyStyle.Render("max_workers"), theme.Primary.Render(fmt.Sprintf("%d", cfg.Worker.MaxWorkers)))
	fmt.Printf("  %s %s\n", keyStyle.Render("max_retries"), theme.Primary.Render(fmt.Sprintf("%d", cfg.Worker.MaxRetries)))
	fmt.Println()

	fmt.Println(theme.Bold.Render("search"))
	fmt.Printf("  %s %s\n", keyStyle.Render("max_results"), theme.Primary.Render(fmt.Sprintf("%d", cfg.Search.MaxResults)))
	fmt.Println()

	fmt.Println(theme.Bold.Render("log"))
	fmt.Printf("  %s %s\n", keyStyle.Render("level"), theme.Primary.Render(cfg.Log.Level))
	fmt.Printf("  %s %s\n", keyStyle.Render("file"), theme.Primary.Render(config.MakeAbsolute(cfg.Log.File, configPath)))
	fmt.Println()

	fmt.Println(theme.Bold.Render("db"))
	fmt.Printf("  %s %s\n", keyStyle.Render("path"), theme.Primary.Render(config.MakeAbsolute(cfg.DB.Path, configPath)))
	fmt.Println()
}
