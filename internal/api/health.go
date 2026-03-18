package api

import (
	"context"
	"net/http"

	"github.com/rawnaqs/khayal/internal/version"
)

type HealthResponse struct {
	Status       string       `json:"status"`
	Version      string       `json:"version"`
	Dependencies Dependencies `json:"dependencies"`
	Queue        QueueStats   `json:"queue"`
}

type Dependencies struct {
	DB    Dependency `json:"db"`
	Vault Dependency `json:"vault"`
}

type Dependency struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
	Host   string `json:"host,omitempty"`
}

type QueueStats struct {
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Done       int `json:"done"`
	Failed     int `json:"failed"`
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	vaultStatus := "ok"
	if !s.vault.Exists() {
		vaultStatus = "error"
	}

	pending, _ := s.queue.CountByStatus(ctx, "pending")
	processing, _ := s.queue.CountByStatus(ctx, "processing")
	done, _ := s.queue.CountByStatus(ctx, "done")
	failed, _ := s.queue.CountByStatus(ctx, "failed")

	WriteJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: version.Get(),
		Dependencies: Dependencies{
			DB: Dependency{
				Status: "ok",
				Path:   s.config.DB.Path,
			},
			Vault: Dependency{
				Status: vaultStatus,
				Path:   s.config.Vault.Path,
			},
		},
		Queue: QueueStats{
			Pending:    pending,
			Processing: processing,
			Done:       done,
			Failed:     failed,
		},
	})
}
