package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"

	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
)

var deleteYes bool

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <note-path-or-job-id>",
		Aliases: []string{"rm"},
		Short:   "Soft-delete a note (moved to .khayal-trash/, recoverable)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0])
		},
	}
	cmd.Flags().BoolVar(&deleteYes, "yes", false, "skip confirmation prompt")
	return cmd
}

func runDelete(target string) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		internal.Fatal(internal.ExitServer, "%s", err.Error())
		return err
	}
	client := klapi.NewClient(cfg.Host, cfg.Token)

	notePath := target

	// Job IDs are UUIDs; note paths contain slashes. Try job resolution first.
	if !strings.Contains(target, "/") && len(target) == 36 && strings.Count(target, "-") == 4 {
		job, err := client.GetJob(target)
		if err == nil && job.NotePath != "" {
			notePath = job.NotePath
		}
	}

	if !deleteYes {
		fmt.Println(theme.Muted.Render("delete") + " " + theme.Primary.Render(notePath) +
			theme.Dim.Render(" → moved to .khayal-trash/"))
		fmt.Print(theme.Muted.Render("confirm? [y/N] "))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println(theme.Dim.Render("cancelled"))
			return nil
		}
	}

	resp, err := client.DeleteNote(notePath)
	if err != nil {
		if strings.Contains(err.Error(), "note not found") {
			fmt.Println(theme.ErrorStyle.Render("✗") + " " + theme.Muted.Render("note not found:") + " " + theme.Primary.Render(notePath))
			return nil
		}
		internal.ServerUnreachable(cfg.Host)
		return err
	}

	fmt.Println(theme.SuccessStyle.Render("✓") + " " + theme.Muted.Render("deleted") + " " +
		theme.Primary.Render(notePath))
	fmt.Println(theme.Dim.Render("  trash: " + resp.TrashPath))
	return nil
}
