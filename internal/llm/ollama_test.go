package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaClient_EmbedBatch(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("expected path /api/embeddings, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if receivedBody["model"] != "test-embed-model" {
			t.Errorf("expected model test-embed-model, got %v", receivedBody["model"])
		}

		prompts, ok := receivedBody["prompts"].([]any)
		if !ok {
			t.Fatal("expected prompts to be an array")
		}

		if len(prompts) != 2 {
			t.Errorf("expected 2 prompts, got %d", len(prompts))
		}

		json.NewEncoder(w).Encode(map[string]any{
			"embeddings": [][]float32{
				{0.1, 0.2, 0.3},
				{0.4, 0.5, 0.6},
			},
		})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "test-embed-model", "test-text-model", "test-vision-model")
	results, err := client.EmbedBatch([]string{"text 1", "text 2"})
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if len(results[0]) != 3 {
		t.Errorf("expected 3 dimensions, got %d", len(results[0]))
	}

	if results[0][0] != 0.1 {
		t.Errorf("expected first value 0.1, got %f", results[0][0])
	}
}

func TestOllamaClient_EmbedBatch_SingleText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var receivedBody map[string]any
		json.NewDecoder(r.Body).Decode(&receivedBody)

		prompts := receivedBody["prompts"].([]any)
		if len(prompts) != 1 {
			t.Errorf("expected 1 prompt, got %d", len(prompts))
		}

		json.NewEncoder(w).Encode(map[string]any{
			"embeddings": [][]float32{
				{0.1, 0.2, 0.3},
			},
		})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "test-embed-model", "test-text-model", "test-vision-model")
	results, err := client.EmbedBatch([]string{"single text"})
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestOllamaClient_EmbedBatch_EmptyInput(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "test-embed-model", "test-text-model", "test-vision-model")
	results, err := client.EmbedBatch([]string{})
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if results != nil {
		t.Errorf("expected nil for empty input, got %v", results)
	}
}

func TestOllamaClient_EmbedBatch_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "test-embed-model", "test-text-model", "test-vision-model")
	_, err := client.EmbedBatch([]string{"text"})
	if err == nil {
		t.Error("expected error for HTTP 500")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain 500, got %v", err)
	}
}

func TestOllamaClient_EmbedBatch_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"embeddings": [][]float32{},
		})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "test-embed-model", "test-text-model", "test-vision-model")
	_, err := client.EmbedBatch([]string{"text"})
	if err == nil {
		t.Error("expected error for empty embeddings")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error to contain 'empty', got %v", err)
	}
}
