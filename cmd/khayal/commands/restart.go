package commands

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Stop and start khayal",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestart()
		},
	}
}

func runRestart() error {
	fmt.Println("stopping khayal...")

	pid, err := cli.GetPID()
	if err == nil {
		process, err := os.FindProcess(pid)
		if err == nil {
			process.Signal(syscall.SIGTERM)

			for i := 0; i < 10; i++ {
				if !cli.IsRunning() {
					break
				}
			}
		}
	}

	cli.RemovePID()
	fmt.Println("khayal stopped.")

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable: %w", err)
	}

	fmt.Println("\nstarting khayal...")
	execmd := exec.Command(self, "start")
	execmd.Stdout = os.Stdout
	execmd.Stderr = os.Stderr
	execmd.Stdin = os.Stdin
	execmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := execmd.Start(); err != nil {
		return fmt.Errorf("failed to start khayal: %w", err)
	}

	if err := execmd.Wait(); err != nil {
		return fmt.Errorf("khayal exited: %w", err)
	}

	return nil
}
