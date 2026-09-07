package commands

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

var voiceNote string

// recorder describes the best available CLI mic recorder on this machine.
// Go has no stdlib microphone access, so recording shells out to a
// platform tool when present; uploading an existing file always works.
type recorder struct {
	bin  string
	args func(out string) []string
	ext  string
	hint string
}

func detectRecorder() *recorder {
	if path, err := exec.LookPath("arecord"); err == nil {
		return &recorder{bin: path, ext: "wav",
			args: func(out string) []string {
				return []string{"-f", "S16_LE", "-r", "16000", "-c", "1", out}
			},
			hint: "press ctrl+c to stop"}
	}
	for _, bin := range []string{"rec", "sox"} {
		if path, err := exec.LookPath(bin); err == nil {
			return &recorder{bin: path, ext: "wav",
				args: func(out string) []string {
					if bin == "sox" {
						return []string{"-d", out}
					}
					return []string{out}
				},
				hint: "press ctrl+c to stop"}
		}
	}
	return nil
}

func newCaptureVoiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "voice [file]",
		Short: "Capture a voice note (record from mic, or upload an audio file)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			var audioPath string
			if len(args) == 1 {
				audioPath = args[0]
			} else {
				rec := detectRecorder()
				if rec == nil {
					fmt.Println(theme.ErrorStyle.Render("✗") + " " + theme.Muted.Render("no mic recorder found"))
					fmt.Println(theme.Muted.Render("  install one:   brew install sox   |   apt install sox (or alsa-utils)"))
					fmt.Println(theme.Muted.Render("  or upload:     kl voice recording.wav"))
					return fmt.Errorf("no recorder available")
				}

				tmp := filepath.Join(os.TempDir(), "khayal-voice."+rec.ext)
				fmt.Println(theme.ProcessingStyle.Render("⏳ recording · " + rec.hint))

				stop := make(chan os.Signal, 1)
				signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

				cmdProc := exec.Command(rec.bin, rec.args(tmp)...)
				cmdProc.Stdin = os.Stdin
				cmdProc.Stdout = os.Stdout
				cmdProc.Stderr = os.Stderr
				if err := cmdProc.Start(); err != nil {
					return fmt.Errorf("recorder failed to start: %w", err)
				}
				go func() {
					<-stop
					// graceful: let the recorder flush its file
					cmdProc.Process.Signal(os.Interrupt)
				}()
				_ = cmdProc.Wait()
				signal.Stop(stop)

				if _, err := os.Stat(tmp); err != nil {
					return fmt.Errorf("recording produced no file (stopped too early?)")
				}
				audioPath = tmp
				fmt.Println(theme.SuccessStyle.Render("✓") + " " + theme.Muted.Render("recorded · uploading"))
			}

			client := klapi.NewClient(cfg.Host, cfg.Token)
			result, err := client.CaptureAudio(audioPath, voiceNote)
			if err != nil {
				if recorded := len(args) == 0; recorded {
					os.Remove(audioPath)
				}
				// Distinguish "server down" from server-side failures
				// (auth, STT errors) — collapsing them into
				// ServerUnreachable hid the real reason.
				if cerr := client.CheckConnection(); cerr != nil {
					internal.ServerUnreachable(cfg.Host)
					return err
				}
				fmt.Println(theme.ErrorStyle.Render("✗") + " " + theme.Muted.Render(err.Error()))
				return nil
			}
			if len(args) == 0 {
				os.Remove(audioPath)
			}

			println(theme.SuccessStyle.Render("✓") + " " + theme.Muted.Render("queued · voice") +
				theme.Dim.Render(" · id: "+result.ID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&voiceNote, "note", "n", "", "add a note")

	return cmd
}
