package commands

import (
	"fmt"
	"os"
	"syscall"
	"time"

	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Graceful shutdown",
		Long: `Stop the running khayal server.

Sends SIGTERM to the khayal process and waits for it to
finish any in-progress jobs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "kill without waiting")

	return cmd
}

func runStop(force bool) error {
	pid, err := cli.GetPID()
	if err != nil {
		cli.ErrorWithHint("khayal is not running", []string{
			"start khayal:     khayal start",
		})
		return fmt.Errorf("server not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		cli.ErrorWithHint("cannot find khayal process", []string{
			"remove stale PID: rm ~/.config/khayal/khayal.pid",
			"start khayal:     khayal start",
		})
		return err
	}

	fmt.Println(theme.Dim.Render("stopping khayal..."))

	if force {
		if err := process.Kill(); err != nil {
			cli.Fatal(cli.ExitServer, "failed to kill process: %v", err)
			return err
		}
		cli.PrintAction("worker", "killed")
		cli.PrintAction("server", "killed")
	} else {
		if err := process.Signal(syscall.SIGTERM); err != nil {
			cli.ErrorWithHint("cannot signal process", []string{
				"try --force:     khayal stop --force",
			})
			return err
		}

		fmt.Print(theme.Dim.Render("  waiting for current job to finish"))
		for i := 0; i < 30; i++ {
			time.Sleep(1 * time.Second)
			fmt.Print(theme.Dim.Render("."))
			if !cli.IsRunning() {
				fmt.Println()
				cli.PrintAction("worker", "finished gracefully")
				cli.PrintAction("server", "stopped")
				break
			}
		}
		fmt.Println()
	}

	if err := cli.RemovePID(); err != nil {
		cli.Warnf("failed to remove PID file: %v", err)
	}

	fmt.Println(theme.Muted.Render("khayal stopped."))
	return nil
}
