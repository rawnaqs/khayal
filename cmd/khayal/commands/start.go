package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/api"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/log"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
	"github.com/rawnaqs/khayal/internal/worker"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start server + worker, run dependency checker",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart()
		},
	}
}

func runStart() error {
	fmt.Println(theme.Primary.Render(fmt.Sprintf("khayal v%s", VersionCmd())))
	fmt.Println()

	if cli.IsRunning() {
		cli.ErrorWithHint("khayal is already running", []string{
			"check status:     khayal status",
			"stop khayal:     khayal stop",
		})
		return fmt.Errorf("server already running")
	}

	fmt.Println(theme.Muted.Render("loading config..."))
	cfg, configPath, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	fmt.Println(theme.Muted.Render("checking dependencies..."))
	deps := cli.CheckDependencies(cfg)
	cli.PrintDependencies(deps)
	fmt.Println()

	missingDeps := []string{}
	for _, d := range deps {
		if !d.OK {
			missingDeps = append(missingDeps, d.Name)
		}
	}
	if len(missingDeps) > 0 {
		cli.PrintSection("install missing dependencies:")
		for _, name := range missingDeps {
			for _, d := range deps {
				if d.Name == name && d.Install != "" {
					fmt.Println(theme.Dim.Render("  " + d.Install))
				}
			}
		}
		return fmt.Errorf("missing dependencies: %v", missingDeps)
	}

	if err := cli.EnsureDirectories(cfg, configPath); err != nil {
		cli.Fatal(cli.ExitVault, "failed to create directories: %v", err)
		return err
	}

	dbPath := config.MakeAbsolute(cfg.DB.Path, configPath)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		cli.PrintAction("db", "creating new database")
	} else {
		cli.PrintAction("db", dbPath)
	}

	logFile := config.MakeAbsolute(cfg.Log.File, configPath)
	loggerSetup, err := log.SetupLogger(
		logFile,
		configPath,
		cfg.Log.RotationMaxSizeMB,
		cfg.Log.RotationMaxFiles,
		cfg.Log.Level,
		cfg.Log.WorkerLevel,
	)
	if err != nil {
		cli.Fatal(cli.ExitServer, "failed to initialize logger: %v", err)
		return err
	}
	defer loggerSetup.Close()
	defer loggerSetup.Sync()
	cli.PrintAction("log", logFile)

	q, err := queue.NewQueue(dbPath)
	if err != nil {
		cli.Fatal(cli.ExitServer, "failed to initialize queue: %v", err)
		return err
	}
	defer q.Close()
	cli.PrintAction("queue", "ready")

	v, err := vault.NewWriter(cfg, configPath)
	if err != nil {
		cli.Fatal(cli.ExitVault, "failed to initialize vault: %v", err)
		return err
	}
	cli.PrintAction("vault", config.MakeAbsolute(cfg.Vault.Path, configPath))

	llmClient, err := llm.NewLLM(cfg.LLM)
	if err != nil {
		cli.Fatal(cli.ExitServer, "failed to initialize LLM: %v", err)
		return err
	}
	cli.PrintAction("llm", cfg.LLM.Provider)

	w := worker.NewWorker(cfg.Worker, q, v, llmClient, loggerSetup.WorkerLogger)
	w.Start()
	cli.PrintAction("worker", "started")

	srv := api.NewServer(cfg, q, v, llmClient, loggerSetup.MainLogger)

	go func() {
		if err := srv.Start(); err != nil {
			loggerSetup.MainLogger.Error("server error", "error", err)
		}
	}()
	cli.PrintAction("server", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))

	pid, err := cli.WritePID()
	if err != nil {
		cli.Warnf("failed to write PID file: %v", err)
	} else {
		cli.PrintAction("pid", fmt.Sprintf("%d", pid))
	}

	fmt.Println()
	fmt.Println(theme.Bold.Render("khayal is running."))
	fmt.Println(theme.Dim.Render("press ctrl+c to stop"))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	fmt.Println(theme.Muted.Render("shutting down..."))
	if err := srv.Close(); err != nil {
		loggerSetup.MainLogger.Error("error closing server", "error", err)
	}
	cli.RemovePID()
	loggerSetup.MainLogger.Info("goodbye")
	fmt.Println(theme.Muted.Render("khayal stopped."))

	return nil
}
