package commands

import (
	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

var imageNote string

func newCaptureImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image <path>",
		Short: "Capture an image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			client := klapi.NewClient(cfg.Host, cfg.Token)
			result, err := client.CaptureImage(args[0], imageNote)
			if err != nil {
				internal.ServerUnreachable(cfg.Host)
				return err
			}

			if result.Status == "done" {
				println(theme.SuccessStyle.Render("✓") + " " + theme.Primary.Render("saved"))
			} else {
				println(theme.ProcessingStyle.Render("⏳") + " " + theme.Muted.Render("queued · image") + " " + theme.Dim.Render("· id: "+result.ID))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&imageNote, "note", "n", "", "add a note")

	return cmd
}
