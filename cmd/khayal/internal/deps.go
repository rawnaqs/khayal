package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/theme"
)

type Dependency struct {
	Name    string
	Check   func() (bool, string)
	Install string
}

type DepStatus struct {
	Name    string
	OK      bool
	Details string
	Install string
}

func CheckDependencies(cfg *config.Config) []DepStatus {
	deps := getDependencies(cfg)
	results := make([]DepStatus, 0, len(deps))

	for _, dep := range deps {
		ok, details := dep.Check()
		results = append(results, DepStatus{
			Name:    dep.Name,
			OK:      ok,
			Details: details,
			Install: dep.Install,
		})
	}

	return results
}

func PrintDependencies(results []DepStatus) {
	for _, r := range results {
		if r.OK {
			fmt.Printf("  %s %s\n", theme.SuccessStyle.Render("✓"), theme.Primary.Render(r.Name))
			if r.Details != "" {
				fmt.Printf("      %s\n", theme.Muted.Render(r.Details))
			}
		} else {
			fmt.Printf("  %s %s\n", theme.ErrorStyle.Render("✗"), theme.Primary.Render(r.Name))
			if r.Details != "" {
				fmt.Printf("      %s %s\n", theme.Muted.Render("→"), r.Details)
			}
		}
	}
}

func HasAllDependencies(results []DepStatus) bool {
	for _, r := range results {
		if !r.OK {
			return false
		}
	}
	return true
}

func getDependencies(cfg *config.Config) []Dependency {
	return []Dependency{
		{
			Name: "ollama",
			Check: func() (bool, string) {
				url := cfg.LLM.OllamaHost + "/api/tags"
				client := &http.Client{Timeout: 2 * time.Second}
				resp, err := client.Get(url)
				if err != nil {
					return false, "not found"
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return false, fmt.Sprintf("returned status %d", resp.StatusCode)
				}

				return true, cfg.LLM.OllamaHost
			},
			Install: "brew install ollama",
		},
	}
}

func CheckOllamaModels(cfg *config.Config) ([]string, error) {
	url := cfg.LLM.OllamaHost + "/api/tags"
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	type TagsResponse struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	var tagsResp TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	models := make([]string, 0, len(tagsResp.Models))
	for _, m := range tagsResp.Models {
		models = append(models, m.Name)
	}

	return models, nil
}

func RequireOllamaModel(cfg *config.Config, modelName string) error {
	models, err := CheckOllamaModels(cfg)
	if err != nil {
		return err
	}

	for _, m := range models {
		if strings.Contains(m, modelName) {
			return nil
		}
	}

	return fmt.Errorf("model %s not found", modelName)
}
