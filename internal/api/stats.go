package api

import (
	"context"
	"net/http"
	"time"

	"github.com/rawnaqs/khayal/internal/queue"
)

type StatsResponse struct {
	Total         int                 `json:"total"`
	ThisWeek      int                 `json:"this_week"`
	ThisMonth     int                 `json:"this_month"`
	ByType        map[string]int      `json:"by_type"`
	TopTags       []queue.TagCount    `json:"top_tags"`
	TopPeople     []queue.PersonCount `json:"top_people"`
	CaptureStreak int                 `json:"capture_streak"`
	LongestStreak int                 `json:"longest_streak"`
}

func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	start := time.Now()

	total, err := s.queue.CountNotes(ctx)
	if err != nil {
		s.logger.Error("stats query failed",
			"code", "COUNT_ERROR",
			"operation", "count_notes",
			"error", err,
		)
		WriteError(w, "failed to count notes", "COUNT_ERROR", http.StatusInternalServerError)
		return
	}

	thisWeek, err := s.queue.CountNotesSince(ctx, time.Now().AddDate(0, 0, -7))
	if err != nil {
		s.logger.Error("stats query failed",
			"code", "COUNT_ERROR",
			"operation", "count_weekly",
			"error", err,
		)
		WriteError(w, "failed to count weekly notes", "COUNT_ERROR", http.StatusInternalServerError)
		return
	}

	thisMonth, err := s.queue.CountNotesSince(ctx, time.Now().AddDate(0, -1, 0))
	if err != nil {
		s.logger.Error("stats query failed",
			"code", "COUNT_ERROR",
			"operation", "count_monthly",
			"error", err,
		)
		WriteError(w, "failed to count monthly notes", "COUNT_ERROR", http.StatusInternalServerError)
		return
	}

	byType, err := s.queue.CountByType(ctx)
	if err != nil {
		s.logger.Error("stats query failed",
			"code", "COUNT_ERROR",
			"operation", "count_by_type",
			"error", err,
		)
		WriteError(w, "failed to count by type", "COUNT_ERROR", http.StatusInternalServerError)
		return
	}

	topTags, err := s.queue.GetTopTags(ctx, 5)
	if err != nil {
		s.logger.Error("stats query failed",
			"code", "COUNT_ERROR",
			"operation", "get_top_tags",
			"error", err,
		)
		WriteError(w, "failed to get top tags", "COUNT_ERROR", http.StatusInternalServerError)
		return
	}

	topPeople, err := s.queue.GetTopPeople(ctx, 3)
	if err != nil {
		s.logger.Error("stats query failed",
			"code", "COUNT_ERROR",
			"operation", "get_top_people",
			"error", err,
		)
		WriteError(w, "failed to get top people", "COUNT_ERROR", http.StatusInternalServerError)
		return
	}

	captureStreak, longestStreak, err := s.queue.GetStreaks(ctx)
	if err != nil {
		s.logger.Error("stats query failed",
			"code", "COUNT_ERROR",
			"operation", "get_streaks",
			"error", err,
		)
		WriteError(w, "failed to calculate streaks", "COUNT_ERROR", http.StatusInternalServerError)
		return
	}

	took := time.Since(start).Milliseconds()
	if took > 100 {
		s.logger.Debug("stats query slow",
			"took_ms", took,
			"total", total,
		)
	}

	WriteJSON(w, http.StatusOK, StatsResponse{
		Total:         total,
		ThisWeek:      thisWeek,
		ThisMonth:     thisMonth,
		ByType:        byType,
		TopTags:       topTags,
		TopPeople:     topPeople,
		CaptureStreak: captureStreak,
		LongestStreak: longestStreak,
	})
}
