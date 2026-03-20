package main

import (
	"fmt"
	"os"

	"github.com/rawnaqs/khayal/cmd/khayal/commands"
	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/version"
)

func main() {
	rootCmd := commands.NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		if err.Error() == "server not running" {
			cli.ErrorWithHint("khayal is not running", []string{
				"start khayal:     khayal start",
			})
			os.Exit(cli.ExitServer)
		}
		if err.Error() == "server already running" {
			cli.ErrorWithHint("khayal is already running", []string{
				"check status:     khayal status",
				"stop khayal:     khayal stop",
			})
			os.Exit(cli.ExitUser)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	commands.InitVersion(version.Get())
}
