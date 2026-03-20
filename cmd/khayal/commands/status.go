package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show server status",
		Long:  `Show a simple status overview of the khayal server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

type healthResponse struct {
	Status       string `json:"status"`
	Version      string `json:"version"`
	Dependencies struct {
		DB    struct{ Status string } `json:"db"`
		Vault struct{ Status string } `json:"vault"`
		LLM   struct {
			Status string `json:"status"`
			Host   string `json:"host"`
		} `json:"llm"`
	} `json:"dependencies"`
	Queue struct {
		Pending    int `json:"pending"`
		Processing int `json:"processing"`
		Done       int `json:"done"`
		Failed     int `json:"failed"`
	} `json:"queue"`
}

func runStatus() error {
	cfg, _, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	if !cli.IsRunning() {
		cli.ErrorWithHint("khayal is not running", []string{
			"start khayal:     khayal start",
		})
		return fmt.Errorf("server not running")
	}

	pid, _ := cli.GetPID()

	url := fmt.Sprintf("http://%s:%d/v1/health", cfg.Server.Host, cfg.Server.Port)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		cli.ServerUnreachable(url)
		return fmt.Errorf("cannot reach server")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cli.ErrorWithHint(fmt.Sprintf("server returned status %d", resp.StatusCode), []string{
			"check logs:       khayal logs",
		})
		return fmt.Errorf("server error")
	}

	var health healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	uptime := getUptime(pid)

	fmt.Println()
	fmt.Printf("  %s %s %s %s\n",
		theme.SuccessStyle.Render("✓"),
		theme.Primary.Render("khayal"),
		theme.Muted.Render(health.Version),
		theme.Primary.Render(fmt.Sprintf("· %s:%d", cfg.Server.Host, cfg.Server.Port)),
	)
	fmt.Println()

	fmt.Println(theme.Primary.Render("  queue"))
	fmt.Printf("    %-12s %d\n", "processing", health.Queue.Processing)
	fmt.Printf("    %-12s %d\n", "pending", health.Queue.Pending)
	fmt.Printf("    %-12s %d\n", "done", health.Queue.Done)
	fmt.Printf("    %-12s %d\n", "failed", health.Queue.Failed)
	fmt.Println()

	fmt.Println(theme.Primary.Render("  system"))
	fmt.Printf("    %-12s %d\n", "pid", pid)
	fmt.Printf("    %-12s %s\n", "uptime", theme.Muted.Render(uptime))
	fmt.Printf("    %-12s %s\n", "db", health.Dependencies.DB.Status)
	fmt.Printf("    %-12s %s\n", "vault", health.Dependencies.Vault.Status)
	fmt.Printf("    %-12s %s (%s)\n", "llm", health.Dependencies.LLM.Status, health.Dependencies.LLM.Host)
	fmt.Println()

	return nil
}

func getUptime(pid int) string {
	return "N/A"
}
