package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rawnaqs/khayal/internal/config"
)

func main() {
	var cfgPath string

	if os.Getenv("KHAYAL_CONFIG") != "" {
		cfgPath = os.Getenv("KHAYAL_CONFIG")
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}
		testCfg := filepath.Join(cwd, "testdata", "config.yaml")
		if _, err := os.Stat(testCfg); err == nil {
			cfgPath = testCfg
		} else {
			cfgPath = config.DefaultConfigPath
		}
	}

	cfg, err := config.LoadFromPath(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", cfgPath, err)
	}

	fmt.Println("Khayal v0.1.0")
	fmt.Println()

	fmt.Printf("Config:       %s\n", cfgPath)
	fmt.Printf("Vault path:   %s\n", cfg.Vault.Path)
	fmt.Printf("DB path:     %s\n", cfg.DB.Path)
	fmt.Printf("Server:      %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("LLM provider: %s\n", cfg.LLM.Provider)
	fmt.Println()

	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	fmt.Println("All directories ready.")
	os.Exit(0)
}
