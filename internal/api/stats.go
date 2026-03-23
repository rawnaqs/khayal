package api

import (
	"context"
	"encoding/json"
	"net/http"
)

func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Try cache first
	cached, err := s.queue.LoadStatsCache(ctx)
	if err != nil {
		s.logger.Error("cache read failed", "error", err)
	}

	if cached != "" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cached))
		return
	}

	// Cache miss — recompute
	stats, err := s.queue.RecomputeStats(ctx)
	if err != nil {
		s.logger.Error("stats recompute failed", "error", err)
		WriteError(w, "failed to compute stats", "STATS_ERROR", http.StatusInternalServerError)
		return
	}

	data, _ := json.Marshal(stats)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
