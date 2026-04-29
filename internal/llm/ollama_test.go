package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/constants"
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

func TestParseJSONArray_ValidArray(t *testing.T) {
	result := parseJSONArray(`["tag1", "tag2", "tag3"]`)
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(result), result)
	}
}

func TestParseJSONArray_MarkdownWrapped(t *testing.T) {
	result := parseJSONArray("Here are the tags:\n```json\n[\"go\", \"rust\"]\n```")
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d: %v", len(result), result)
	}
	if result[0] != "go" || result[1] != "rust" {
		t.Errorf("unexpected items: %v", result)
	}
}

func TestParseJSONArray_ObjectStyle(t *testing.T) {
	result := parseJSONArray(`[{"idea": "first idea"}, {"idea": "second idea"}]`)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d: %v", len(result), result)
	}
}

func TestParseJSONArray_TextWithIntro(t *testing.T) {
	result := parseJSONArray("Sure, here are the key ideas:\n[\"idea 1\", \"idea 2\", \"idea 3\"]\nHope that helps!")
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(result), result)
	}
}

func TestParseJSONArray_EmptyResult(t *testing.T) {
	result := parseJSONArray("No key ideas found in this content.")
	if len(result) != 0 {
		t.Errorf("expected 0 items for empty content, got %d: %v", len(result), result)
	}
}

func TestParseJSONArray_LineByLineFallback(t *testing.T) {
	result := parseJSONArray("- idea one\n- idea two\n* idea three")
	if len(result) != 3 {
		t.Fatalf("expected 3 items from line fallback, got %d: %v", len(result), result)
	}
}

func TestParseJSONArray_MaxLimit(t *testing.T) {
	result := parseJSONArray(`["a", "b", "c", "d", "e", "f", "g", "h"]`)
	if len(result) != 5 {
		t.Fatalf("expected max 5 items, got %d", len(result))
	}
}

func TestTruncateForLLM_ShortContent(t *testing.T) {
	content := "hello world"
	result := truncateForLLM(content, 100)
	if result != content {
		t.Errorf("short content should not be truncated, got %q", result)
	}
}

func TestTruncateForLLM_LongContent(t *testing.T) {
	content := strings.Repeat("abcdefghij", 200) // 2000 chars
	result := truncateForLLM(content, 100)       // max 400 chars

	if len(result) > 400 {
		t.Errorf("truncated content too long: %d > 400", len(result))
	}
	if !strings.Contains(result, "...[truncated]...") {
		t.Error("truncated content should contain separator")
	}
}

func TestTruncateForLLM_ExactBoundary(t *testing.T) {
	content := strings.Repeat("a", 400) // exactly 100 tokens * 4
	result := truncateForLLM(content, 100)
	if result != content {
		t.Error("content at exact boundary should not be truncated")
	}
}

func TestGetSystemPrompt_Default(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "e", "t", "v")

	tests := []struct {
		op, bucket string
		expected   string
	}{
		{"extract_tags", "text", constants.DefaultSystemPrompts.ExtractTags},
		{"summarize", "article", constants.DefaultSystemPrompts.Summarize},
		{"extract_key_ideas", "text", constants.DefaultSystemPrompts.ExtractKeyIdeas},
		{"describe_image", "", constants.DefaultSystemPrompts.DescribeImage},
	}

	for _, tt := range tests {
		t.Run(tt.op+"_"+tt.bucket, func(t *testing.T) {
			got := client.getSystemPrompt(tt.op, tt.bucket)
			if got != tt.expected {
				t.Errorf("getSystemPrompt(%q, %q) = %q, want %q", tt.op, tt.bucket, got, tt.expected)
			}
		})
	}
}

func TestGetSystemPrompt_PerBucketOverride(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "e", "t", "v")
	customPrompt := "Custom extract tags for articles"
	client.SetPerBucketSystem(map[string]string{
		"extract_tags:article": customPrompt,
	})

	got := client.getSystemPrompt("extract_tags", "article")
	if got != customPrompt {
		t.Errorf("expected per-bucket override, got %q", got)
	}

	got = client.getSystemPrompt("extract_tags", "text")
	if got != constants.DefaultSystemPrompts.ExtractTags {
		t.Errorf("text bucket should use default, got %q", got)
	}
}

func TestGetTemperature_Defaults(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "e", "t", "v")

	if got := client.getTemperature("extract_tags"); got != 0.3 {
		t.Errorf("tags temp = %f, want 0.3", got)
	}
	if got := client.getTemperature("summarize"); got != 0.4 {
		t.Errorf("summarize temp = %f, want 0.4", got)
	}
	if got := client.getTemperature("extract_key_ideas"); got != 0.7 {
		t.Errorf("key_ideas temp = %f, want 0.7", got)
	}
	if got := client.getTemperature("describe_image"); got != 0.7 {
		t.Errorf("vision temp = %f, want 0.7", got)
	}
}

func TestGetTemperature_Overrides(t *testing.T) {
	client := NewOllamaClient("http://localhost:11434", "e", "t", "v")
	client.SetTempTag(0.5)
	client.SetTempSummarize(0.6)

	if got := client.getTemperature("extract_tags"); got != 0.5 {
		t.Errorf("overridden tags temp = %f, want 0.5", got)
	}
	if got := client.getTemperature("summarize"); got != 0.6 {
		t.Errorf("overridden summarize temp = %f, want 0.6", got)
	}
}

func TestFactory_AppliesPromptConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
	}))
	defer server.Close()

	customTagPrompt := "Custom tag system prompt"
	customArticleTemplate := "Custom article template: %s"

	cfg := config.LLMConfig{
		Provider:    "ollama",
		OllamaHost:  server.URL,
		EmbedModel:  "e",
		TextModel:   "t",
		VisionModel: "v",
		MaxLLMConcurrency: 1,
		Prompts: &config.PromptConfig{
			ExtractTags:               customTagPrompt,
			ExtractTagsArticleTemplate: customArticleTemplate,
		},
	}

	client, err := NewLLM(cfg)
	if err != nil {
		t.Fatalf("NewLLM failed: %v", err)
	}

	ollamaClient, ok := client.(*OllamaClient)
	if !ok {
		t.Fatal("client is not *OllamaClient")
	}

	if ollamaClient.systemPrompts.ExtractTags != customTagPrompt {
		t.Errorf("systemPrompts.ExtractTags = %q, want %q", ollamaClient.systemPrompts.ExtractTags, customTagPrompt)
	}

	if tmpl := ollamaClient.prompts.ExtractTags["article"]; tmpl != customArticleTemplate {
		t.Errorf("prompts.ExtractTags[article] = %q, want %q", tmpl, customArticleTemplate)
	}
}

func TestFactory_AppliesTemperatureOverrides(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
	}))
	defer server.Close()

	cfg := config.LLMConfig{
		Provider:            "ollama",
		OllamaHost:          server.URL,
		EmbedModel:          "e",
		TextModel:           "t",
		VisionModel:         "v",
		MaxLLMConcurrency:   1,
		TemperatureTags:     0.2,
		TemperatureSummarize: 0.35,
		TemperatureKeyIdeas:  0.8,
		TemperatureVision:    0.9,
	}

	client, err := NewLLM(cfg)
	if err != nil {
		t.Fatalf("NewLLM failed: %v", err)
	}

	ollamaClient := client.(*OllamaClient)

	if got := ollamaClient.getTemperature("extract_tags"); got != 0.2 {
		t.Errorf("tags temp = %f, want 0.2", got)
	}
	if got := ollamaClient.getTemperature("summarize"); got != 0.35 {
		t.Errorf("summarize temp = %f, want 0.35", got)
	}
	if got := ollamaClient.getTemperature("extract_key_ideas"); got != 0.8 {
		t.Errorf("key_ideas temp = %f, want 0.8", got)
	}
	if got := ollamaClient.getTemperature("describe_image"); got != 0.9 {
		t.Errorf("vision temp = %f, want 0.9", got)
	}
}
