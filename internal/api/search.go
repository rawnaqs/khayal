package api

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/rawnaqs/khayal/internal/constants"
	"github.com/rawnaqs/khayal/internal/queue"
)

type SearchResponse struct {
	Query   string               `json:"query"`
	Mode    string               `json:"mode"`
	Results []queue.SearchResult `json:"results"`
	Total   int                  `json:"total"`
	TookMs  int64                `json:"took_ms"`
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	start := time.Now()

	query := r.URL.Query().Get("q")
	if query == "" {
		WriteError(w, "missing required parameter: q", "SEARCH_MISSING_QUERY", http.StatusBadRequest)
		return
	}

	limit := s.parseLimit(r.URL.Query().Get("limit"), 10, s.config.Search.MaxResults)

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "hybrid"
	}

	queryForLog := query
	if len(queryForLog) > 50 {
		queryForLog = queryForLog[:50] + "..."
	}

	var results []queue.SearchResult
	var err error

	switch mode {
	case "keyword":
		results, err = s.queue.SearchKeyword(ctx, query, limit)
	case "semantic":
		results, err = s.searchSemantic(ctx, query, limit)
	case "hybrid":
		results, err = s.searchHybrid(ctx, query, limit)
	default:
		WriteError(w, "invalid search mode", "SEARCH_INVALID_MODE", http.StatusBadRequest)
		return
	}

	if err != nil {
		s.logger.Error("search failed",
			"code", "SEARCH_FAILED",
			"query", queryForLog,
			"mode", mode,
			"error", err,
		)
		WriteError(w, "search failed", "SEARCH_FAILED", http.StatusInternalServerError)
		return
	}

	if results == nil {
		results = []queue.SearchResult{}
	}

	took := time.Since(start).Milliseconds()

	s.logger.Info("search",
		"query", queryForLog,
		"mode", mode,
		"results_count", len(results),
		"took_ms", took,
	)

	WriteJSON(w, http.StatusOK, SearchResponse{
		Query:   query,
		Mode:    mode,
		Results: results,
		Total:   len(results),
		TookMs:  took,
	})
}

func (s *Server) searchSemantic(ctx context.Context, query string, limit int) ([]queue.SearchResult, error) {
	embedding, err := s.llm.Embed(query)
	if err != nil {
		return nil, err
	}
	return s.queue.SearchSemantic(ctx, embedding, limit, s.config.Search.MinSemanticScore)
}

func (s *Server) searchHybrid(ctx context.Context, query string, limit int) ([]queue.SearchResult, error) {
	keywordResults, err := s.queue.SearchKeyword(ctx, query, limit*constants.SearchOverFetchMultiplier)
	if err != nil {
		return nil, err
	}

	embedding, err := s.llm.Embed(query)
	if err != nil {
		return nil, err
	}

	semanticResults, err := s.queue.SearchSemantic(ctx, embedding, limit*constants.SearchOverFetchMultiplier, s.config.Search.MinSemanticScore)
	if err != nil {
		return nil, err
	}

	return mergeResultsRRF(keywordResults, semanticResults, s.config.Search.RRFK, limit), nil
}

func mergeResultsRRF(keywordResults, semanticResults []queue.SearchResult, k, limit int) []queue.SearchResult {
	type scoredResult struct {
		result queue.SearchResult
		score  float64
	}

	scoreMap := make(map[string]scoredResult)

	for _, r := range keywordResults {
		r.Score = 1.0
		scoreMap[r.NotePath] = scoredResult{result: r, score: 1.0}
	}

	for _, r := range semanticResults {
		if _, exists := scoreMap[r.NotePath]; exists {
			continue
		}
		scoreMap[r.NotePath] = scoredResult{result: r, score: r.Score}
	}

	results := make([]queue.SearchResult, 0, len(scoreMap))
	for _, sr := range scoreMap {
		results = append(results, sr.result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}
