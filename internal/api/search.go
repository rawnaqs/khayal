package api

import (
	"context"
	"net/http"
	"sort"
	"time"

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
		WriteError(w, "search failed", "SEARCH_FAILED", http.StatusInternalServerError)
		return
	}

	if results == nil {
		results = []queue.SearchResult{}
	}

	took := time.Since(start).Milliseconds()

	WriteJSON(w, http.StatusOK, SearchResponse{
		Query:   query,
		Mode:    mode,
		Results: results,
		Total:   len(results),
		TookMs:  took,
	})
}

func (s *Server) searchSemantic(ctx context.Context, query string, limit int) ([]queue.SearchResult, error) {
	embedding := mockEmbeddings(query)
	return s.queue.SearchSemantic(ctx, embedding, limit)
}

func (s *Server) searchHybrid(ctx context.Context, query string, limit int) ([]queue.SearchResult, error) {
	keywordResults, err := s.queue.SearchKeyword(ctx, query, limit*2)
	if err != nil {
		return nil, err
	}

	embedding := mockEmbeddings(query)
	semanticResults, err := s.queue.SearchSemantic(ctx, embedding, limit*2)
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

	for i, r := range keywordResults {
		rrfScore := 1.0 / (float64(k) + float64(i+1))
		if existing, ok := scoreMap[r.NotePath]; !ok || rrfScore > existing.score {
			scoreMap[r.NotePath] = scoredResult{result: r, score: rrfScore}
		}
	}

	for i, r := range semanticResults {
		rrfScore := 1.0 / (float64(k) + float64(i+1))
		if existing, ok := scoreMap[r.NotePath]; !ok || existing.score+rrfScore > existing.score {
			existing.score += rrfScore
			if existing.result.NotePath == "" {
				existing.result = r
			}
			scoreMap[r.NotePath] = existing
		}
	}

	results := make([]queue.SearchResult, 0, len(scoreMap))
	for _, sr := range scoreMap {
		sr.result.Score = sr.score
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

func mockEmbeddings(text string) []float32 {
	embedding := make([]float32, 384)

	hash := uint64(0)
	for i, c := range text {
		hash = hash*31 + uint64(c) + uint64(i)
	}

	for i := range embedding {
		embedding[i] = float32((hash>>uint(i%64))&0xFF) / 255.0
	}

	norm := float64(0)
	for _, v := range embedding {
		norm += float64(v) * float64(v)
	}
	norm = 1.0

	for i := range embedding {
		embedding[i] = float32(float64(embedding[i]) / norm)
	}

	return embedding
}
