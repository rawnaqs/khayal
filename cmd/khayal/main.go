package main

import (
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rawnaqs/khayal/internal/api"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/log"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
	"github.com/rawnaqs/khayal/internal/version"
	"github.com/rawnaqs/khayal/internal/worker"
)

func main() {
	cfgPath := os.Getenv("KHAYAL_CONFIG")
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath
	}

	cfg, absCfgPath, err := config.LoadFromPath(cfgPath)
	if err != nil {
		stdlog.Fatalf("Failed to load config from %s: %v", cfgPath, err)
	}

	fmt.Printf("Khayal v%s\n", version.Get())
	fmt.Println()

	fmt.Printf("Config:       %s\n", absCfgPath)
	fmt.Printf("Vault path:   %s\n", cfg.Vault.Path)
	fmt.Printf("DB path:      %s\n", cfg.DB.Path)
	fmt.Printf("Server:       %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("LLM provider: %s\n", cfg.LLM.Provider)
	fmt.Printf("Log level:    %s\n", cfg.Log.Level)
	if cfg.Log.WorkerLevel != "" {
		fmt.Printf("Worker level: %s\n", cfg.Log.WorkerLevel)
	}
	fmt.Println()

	if err := cfg.EnsureDirectories(absCfgPath); err != nil {
		stdlog.Fatalf("Failed to create directories: %v", err)
	}

	fmt.Println("All directories ready.")

	loggerSetup, err := log.SetupLogger(
		cfg.Log.File,
		absCfgPath,
		cfg.Log.RotationMaxSizeMB,
		cfg.Log.RotationMaxFiles,
		cfg.Log.Level,
		cfg.Log.WorkerLevel,
	)
	if err != nil {
		stdlog.Fatalf("Failed to initialize logger: %v", err)
	}
	defer loggerSetup.Close()
	defer loggerSetup.Sync()
	loggerSetup.MainLogger.Info("logging initialized",
		"log_file", cfg.Log.File,
		"level", cfg.Log.Level,
		"worker_level", cfg.Log.WorkerLevel,
	)

	dbPath := config.MakeAbsolute(cfg.DB.Path, absCfgPath)
	q, err := queue.NewQueue(dbPath)
	if err != nil {
		stdlog.Fatalf("Failed to initialize queue: %v", err)
	}
	defer q.Close()
	fmt.Println("Database ready.")

	v, err := vault.NewWriter(cfg, absCfgPath)
	if err != nil {
		stdlog.Fatalf("Failed to initialize vault: %v", err)
	}
	fmt.Println("Vault ready.")

	llmClient, err := llm.NewLLM(cfg.LLM)
	if err != nil {
		stdlog.Fatalf("Failed to initialize LLM: %v", err)
	}
	fmt.Println("LLM ready.")

	w := worker.NewWorker(cfg.Worker, q, v, llmClient, loggerSetup.WorkerLogger)
	w.Start()
	defer w.Stop()
	fmt.Println("Worker started.")

	srv := api.NewServer(cfg, q, v, llmClient, loggerSetup.MainLogger)

	go func() {
		if err := srv.Start(); err != nil {
			loggerSetup.MainLogger.Error("server error", "error", err)
		}
	}()

	fmt.Printf("Server listening on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("Press Ctrl+C to stop")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	if err := srv.Close(); err != nil {
		loggerSetup.MainLogger.Error("error closing server", "error", err)
	}
	loggerSetup.MainLogger.Info("goodbye")
	fmt.Println("Goodbye!")
}
