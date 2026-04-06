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

	var from, to *time.Time
	if f := r.URL.Query().Get("from"); f != "" {
		t, err := time.Parse("2006-01-02", f)
		if err != nil {
			WriteError(w, "invalid from date (use YYYY-MM-DD)", "SEARCH_INVALID_DATE", http.StatusBadRequest)
			return
		}
		from = &t
	}
	if t := r.URL.Query().Get("to"); t != "" {
		end, err := time.Parse("2006-01-02", t)
		if err != nil {
			WriteError(w, "invalid to date (use YYYY-MM-DD)", "SEARCH_INVALID_DATE", http.StatusBadRequest)
			return
		}
		end = end.Add(24*time.Hour - time.Second)
		to = &end
	}

	queryForLog := query
	if len(queryForLog) > 50 {
		queryForLog = queryForLog[:50] + "..."
	}

	var results []queue.SearchResult
	var err error

	switch mode {
	case "keyword":
		results, err = s.queue.SearchKeyword(ctx, query, limit, from, to)
	case "semantic":
		results, err = s.searchSemantic(ctx, query, limit, from, to)
	case "hybrid":
		results, err = s.searchHybrid(ctx, query, limit, from, to)
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

func (s *Server) searchSemantic(ctx context.Context, query string, limit int, from, to *time.Time) ([]queue.SearchResult, error) {
	embedding, err := s.llm.Embed(query)
	if err != nil {
		return nil, err
	}
	return s.queue.SearchSemantic(ctx, embedding, limit, s.config.Search.MinSemanticScore, from, to)
}

func (s *Server) searchHybrid(ctx context.Context, query string, limit int, from, to *time.Time) ([]queue.SearchResult, error) {
	keywordResults, err := s.queue.SearchKeyword(ctx, query, limit*constants.SearchOverFetchMultiplier, from, to)
	if err != nil {
		return nil, err
	}

	embedding, err := s.llm.Embed(query)
	if err != nil {
		return nil, err
	}

	semanticResults, err := s.queue.SearchSemantic(ctx, embedding, limit*constants.SearchOverFetchMultiplier, s.config.Search.MinSemanticScore, from, to)
	if err != nil {
		return nil, err
	}

	return mergeResultsRRF(keywordResults, semanticResults, s.config.Search.RRFK, limit), nil
}

func mergeResultsRRF(keywordResults, semanticResults []queue.SearchResult, k, limit int) []queue.SearchResult {
	scoreMap := make(map[string]float64)
	resultMap := make(map[string]queue.SearchResult)

	for rank, r := range keywordResults {
		scoreMap[r.NotePath] += 1.0 / float64(k+rank)
		resultMap[r.NotePath] = r
	}

	for rank, r := range semanticResults {
		scoreMap[r.NotePath] += 1.0 / float64(k+rank)
		if _, ok := resultMap[r.NotePath]; !ok {
			resultMap[r.NotePath] = r
		}
	}

	type scored struct {
		result queue.SearchResult
		score  float64
	}
	scoredResults := make([]scored, 0, len(scoreMap))
	for path, score := range scoreMap {
		r := resultMap[path]
		r.Score = score * float64(k) / 2.0
		scoredResults = append(scoredResults, scored{r, score})
	}

	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].score > scoredResults[j].score
	})

	if len(scoredResults) > limit {
		scoredResults = scoredResults[:limit]
	}

	results := make([]queue.SearchResult, len(scoredResults))
	for i, sr := range scoredResults {
		results[i] = sr.result
	}
	return results
}
