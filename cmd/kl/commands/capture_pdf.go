package commands

import (
	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

var pdfNote string

func newCapturePDFCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pdf <path>",
		Short: "Capture a PDF (text is extracted automatically)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			client := klapi.NewClient(cfg.Host, cfg.Token)
			result, err := client.CapturePDF(args[0], pdfNote)
			if err != nil {
				internal.ServerUnreachable(cfg.Host)
				return err
			}

			if result.Status == "done" {
				println(theme.SuccessStyle.Render("✓") + " " + theme.Primary.Render("saved"))
			} else {
				println(theme.ProcessingStyle.Render("⏳") + " " + theme.Muted.Render("queued · pdf") + " " + theme.Dim.Render("· id: "+result.ID))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&pdfNote, "note", "n", "", "add a note")

	return cmd
}
