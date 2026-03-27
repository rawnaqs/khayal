package api

import (
	"context"
	"net/http"

	"github.com/rawnaqs/khayal/internal/updater"
	"github.com/rawnaqs/khayal/internal/version"
)

type HealthResponse struct {
	Status       string              `json:"status"`
	Version      string              `json:"version"`
	Update       *updater.UpdateInfo `json:"update,omitempty"`
	Dependencies Dependencies        `json:"dependencies"`
}

type Dependencies struct {
	DB    Dependency `json:"db"`
	Vault Dependency `json:"vault"`
	LLM   Dependency `json:"llm"`
}

type Dependency struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
	Host   string `json:"host,omitempty"`
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	_ = context.Background()

	vaultStatus := "ok"
	if !s.vault.Exists() {
		vaultStatus = "error"
	}

	llmStatus := "ok"
	llmHost := s.config.LLM.OllamaHost
	if s.llm != nil {
		if err := s.llm.Ping(); err != nil {
			llmStatus = "error"
		}
	} else {
		llmStatus = "not configured"
	}

	if vaultStatus != "ok" {
		s.logger.Warn("health check degraded",
			"component", "vault",
			"status", vaultStatus,
		)
	}
	if llmStatus != "ok" {
		s.logger.Warn("health check degraded",
			"component", "llm",
			"status", llmStatus,
		)
	}

	updateInfo := updater.CheckForUpdate()

	WriteJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: version.Get(),
		Update:  updateInfo,
		Dependencies: Dependencies{
			DB: Dependency{
				Status: "ok",
				Path:   s.config.DB.Path,
			},
			Vault: Dependency{
				Status: vaultStatus,
				Path:   s.config.Vault.Path,
			},
			LLM: Dependency{
				Status: llmStatus,
				Host:   llmHost,
			},
		},
	})
}
