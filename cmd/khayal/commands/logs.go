package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var follow bool

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Tail ~/.config/khayal/logs/khayal.log",
		Long: `Tail the khayal log file.

By default shows the last 100 lines. Use -f to follow
new log entries in real-time.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(follow)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")

	return cmd
}

func runLogs(follow bool) error {
	logPath := config.ExpandTilde("~/.config/khayal/logs/khayal.log")

	absPath, err := filepath.Abs(logPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	file, err := os.Open(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("no log file found:", absPath)
			return nil
		}
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	if !follow {
		scanner := bufio.NewScanner(file)
		count := 0
		for scanner.Scan() && count < 100 {
			fmt.Println(scanner.Text())
			count++
		}
		return nil
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		file.Close()
		os.Exit(0)
	}()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		fmt.Print(line)
	}

	return nil
}
