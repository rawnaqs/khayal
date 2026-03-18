package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rawnaqs/khayal/internal/api"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
	"github.com/rawnaqs/khayal/internal/version"
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

	fmt.Printf("Khayal v%s\n", version.Get())
	fmt.Println()

	fmt.Printf("Config:       %s\n", cfgPath)
	fmt.Printf("Vault path:   %s\n", cfg.Vault.Path)
	fmt.Printf("DB path:      %s\n", cfg.DB.Path)
	fmt.Printf("Server:       %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("LLM provider: %s\n", cfg.LLM.Provider)
	fmt.Println()

	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	fmt.Println("All directories ready.")

	q, err := queue.NewQueue(cfg.DB.Path)
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}
	defer q.Close()
	fmt.Println("Database ready.")

	v, err := vault.NewWriter(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize vault: %v", err)
	}
	fmt.Println("Vault ready.")

	srv := api.NewServer(cfg, q, v)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	fmt.Printf("Server listening on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("Press Ctrl+C to stop")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	if err := srv.Close(); err != nil {
		log.Printf("Error closing server: %v", err)
	}
	fmt.Println("Goodbye!")
}
