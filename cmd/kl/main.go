package main

import (
	"os"

	"github.com/rawnaqs/khayal/cmd/kl/commands"
)

func main() {
	rootCmd := commands.NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
