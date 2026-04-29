package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rawnaqs/khayal/internal/constants"
)

type OllamaClient struct {
	baseURL               string
	embedModel            string
	textModel             string
	visionModel           string
	truncateTextTokens    int
	truncateImageTokens   int
	truncateArticleTokens int
	httpClient            *http.Client
	logger                *slog.Logger
	semaphore             chan struct{}
	temperature           float64
	systemPrompts         constants.SystemPrompts
	prompts               constants.PromptTemplates
}

func NewOllamaClient(baseURL, embedModel, textModel, visionModel string) *OllamaClient {
	return NewOllamaClientWithConcurrency(baseURL, embedModel, textModel, visionModel, 4)
}

func NewOllamaClientWithConcurrency(baseURL, embedModel, textModel, visionModel string, maxConcurrent int) *OllamaClient {
	if maxConcurrent < 1 {
		maxConcurrent = constants.DefaultMaxConcurrent
	}
	return &OllamaClient{
		baseURL:               strings.TrimSuffix(baseURL, "/"),
		embedModel:            embedModel,
		textModel:             textModel,
		visionModel:           visionModel,
		truncateTextTokens:    constants.DefaultTruncateTextTokens,
		truncateImageTokens:   constants.DefaultTruncateImageTokens,
		truncateArticleTokens: constants.DefaultTruncateArticleTokens,
		httpClient: &http.Client{
			Timeout: constants.OllamaClientTimeout,
		},
		logger:        slog.Default(),
		semaphore:     make(chan struct{}, maxConcurrent),
		temperature:   constants.DefaultTemperature,
		systemPrompts: constants.DefaultSystemPrompts,
		prompts:       constants.DefaultPromptTemplates,
	}
}

func NewOllamaClientWithConfig(baseURL, embedModel, textModel, visionModel string, maxConcurrent int, temperature float64, overridePrompts map[string]string) *OllamaClient {
	if maxConcurrent < 1 {
		maxConcurrent = constants.DefaultMaxConcurrent
	}
	if temperature <= 0 {
		temperature = constants.DefaultTemperature
	}
	return &OllamaClient{
		baseURL:               strings.TrimSuffix(baseURL, "/"),
		embedModel:            embedModel,
		textModel:             textModel,
		visionModel:           visionModel,
		truncateTextTokens:    constants.DefaultTruncateTextTokens,
		truncateImageTokens:   constants.DefaultTruncateImageTokens,
		truncateArticleTokens: constants.DefaultTruncateArticleTokens,
		httpClient: &http.Client{
			Timeout: constants.OllamaClientTimeout,
		},
		logger:        slog.Default(),
		semaphore:     make(chan struct{}, maxConcurrent),
		temperature:   temperature,
		systemPrompts: constants.DefaultSystemPrompts,
		prompts:       constants.DefaultPromptTemplates,
	}
}

func NewOllamaClientWithLogger(baseURL, embedModel, textModel, visionModel string, logger *slog.Logger) *OllamaClient {
	c := NewOllamaClient(baseURL, embedModel, textModel, visionModel)
	if logger != nil {
		c.logger = logger
	}
	return c
}

func (c *OllamaClient) truncateLimit(bucket string) int {
	switch bucket {
	case "text":
		return c.truncateTextTokens
	case "image":
		return c.truncateImageTokens
	case "article":
		return c.truncateArticleTokens
	default:
		return c.truncateTextTokens
	}
}

func (c *OllamaClient) acquire(ctx context.Context) error {
	select {
	case c.semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *OllamaClient) release() {
	<-c.semaphore
}

func (c *OllamaClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.OllamaPingTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *OllamaClient) Type() string {
	return "ollama"
}

func (c *OllamaClient) Embed(text string) ([]float32, error) {
	if len(text) == 0 {
		return nil, fmt.Errorf("empty text")
	}

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), constants.OllamaEmbedTimeout)
	defer cancel()

	if err := c.acquire(ctx); err != nil {
		return nil, fmt.Errorf("llm concurrency limit reached: %w", err)
	}
	defer c.release()

	reqBody := map[string]any{
		"model":  c.embedModel,
		"prompt": text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		c.logger.Debug("llm embed failed", "model", c.embedModel, "error", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("llm embed failed", "model", c.embedModel, "status", resp.StatusCode)
		return nil, fmt.Errorf("embed request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Debug("llm embed failed", "model", c.embedModel, "error", err.Error())
		return nil, err
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	c.logger.Debug("llm embed", "model", c.embedModel, "text_length", len(text), "duration_ms", time.Since(start).Milliseconds())

	return result.Embedding, nil
}

func (c *OllamaClient) Generate(prompt string) (string, error) {
	return c.generateWithModel(c.textModel, "", prompt)
}

func (c *OllamaClient) GenerateWithSystem(system, user string) (string, error) {
	return c.generateWithModel(c.textModel, system, user)
}

func (c *OllamaClient) generateWithModel(model, system, prompt string) (string, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := c.acquire(ctx); err != nil {
		return "", fmt.Errorf("llm concurrency limit reached: %w", err)
	}
	defer c.release()

	reqBody := map[string]any{
		"model":       model,
		"prompt":      prompt,
		"stream":      false,
		"temperature": c.temperature,
	}
	if system != "" {
		reqBody["system"] = system
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		c.logger.Debug("llm generate failed", "model", model, "error", err.Error())
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("llm generate failed", "model", model, "status", resp.StatusCode)
		return "", fmt.Errorf("generate request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Debug("llm generate failed", "model", model, "error", err.Error())
		return "", err
	}

	c.logger.Debug("llm generate", "model", model, "prompt_length", len(prompt), "response_length", len(result.Response), "duration_ms", time.Since(start).Milliseconds())

	return result.Response, nil
}

func (c *OllamaClient) DescribeImage(imagePath string) (string, error) {
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := c.acquire(ctx); err != nil {
		return "", fmt.Errorf("llm concurrency limit reached: %w", err)
	}
	defer c.release()

	base64Img := base64.StdEncoding.EncodeToString(imgData)

	reqBody := map[string]any{
		"model":  c.visionModel,
		"system": c.systemPrompts.DescribeImage,
		"prompt": c.prompts.DescribeImage,
		"images": []string{base64Img},
		"stream": false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	start := time.Now()
	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		c.logger.Debug("llm vision failed", "model", c.visionModel, "image_path", imagePath, "error", err.Error())
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("llm vision failed", "model", c.visionModel, "image_path", imagePath, "status", resp.StatusCode)
		return "", fmt.Errorf("vision request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Debug("llm vision failed", "model", c.visionModel, "image_path", imagePath, "error", err.Error())
		return "", err
	}

	c.logger.Debug("llm vision", "model", c.visionModel, "image_path", imagePath, "image_size", len(imgData), "description_length", len(result.Response), "duration_ms", time.Since(start).Milliseconds())

	return result.Response, nil
}

func (c *OllamaClient) ExtractTags(content string, bucket string) ([]string, error) {
	truncated := truncateForLLM(content, c.truncateLimit(bucket))

	tmpl, ok := c.prompts.ExtractTags[bucket]
	if !ok {
		tmpl = c.prompts.ExtractTags["text"]
	}
	userPrompt := fmt.Sprintf(tmpl, truncated)

	result, err := c.GenerateWithSystem(c.systemPrompts.ExtractTags, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tags: %w", err)
	}

	return parseJSONArray(result), nil
}

func (c *OllamaClient) Summarize(content string, bucket string) (string, error) {
	truncated := truncateForLLM(content, c.truncateLimit(bucket))

	tmpl, ok := c.prompts.Summarize[bucket]
	if !ok {
		tmpl = c.prompts.Summarize["text"]
	}
	userPrompt := fmt.Sprintf(tmpl, truncated)

	result, err := c.GenerateWithSystem(c.systemPrompts.Summarize, userPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to summarize: %w", err)
	}

	return strings.TrimSpace(result), nil
}

func (c *OllamaClient) ExtractKeyIdeas(content string, bucket string) ([]string, error) {
	truncated := truncateForLLM(content, c.truncateLimit(bucket))

	tmpl, ok := c.prompts.ExtractKeyIdeas[bucket]
	if !ok {
		tmpl = c.prompts.ExtractKeyIdeas["text"]
	}
	userPrompt := fmt.Sprintf(tmpl, truncated)

	result, err := c.GenerateWithSystem(c.systemPrompts.ExtractKeyIdeas, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract key ideas: %w", err)
	}

	return parseJSONArray(result), nil
}

func (c *OllamaClient) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	prompts := make([]map[string]string, len(texts))
	for i, text := range texts {
		prompts[i] = map[string]string{"prompt": text}
	}

	reqBody := map[string]any{
		"model":   c.embedModel,
		"prompts": prompts,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed batch request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Embeddings [][]float32 `json:"embeddings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embeddings returned")
	}

	return result.Embeddings, nil
}

func parseJSONArray(result string) []string {
	result = strings.TrimSpace(result)

	start := strings.Index(result, "[")
	end := strings.LastIndex(result, "]")
	if start >= 0 && end > start {
		jsonStr := result[start : end+1]

		// Try []string first
		var items []string
		if err := json.Unmarshal([]byte(jsonStr), &items); err == nil {
			return limitItems(items)
		}

		// Try []map[string]string (e.g., [{"idea": "..."}, {"tag": "..."}])
		var objects []map[string]string
		if err := json.Unmarshal([]byte(jsonStr), &objects); err == nil {
			items := make([]string, 0, len(objects))
			for _, obj := range objects {
				for _, v := range obj {
					if v != "" {
						items = append(items, v)
					}
				}
			}
			if len(items) > 0 {
				return limitItems(items)
			}
		}

		// Try []map[string]any (mixed types)
		var rawObjects []map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &rawObjects); err == nil {
			items := make([]string, 0, len(rawObjects))
			for _, obj := range rawObjects {
				for _, v := range obj {
					if s, ok := v.(string); ok && s != "" {
						items = append(items, s)
					}
				}
			}
			if len(items) > 0 {
				return limitItems(items)
			}
		}
	}

	// Fallback: parse line by line
	items := make([]string, 0, 5)
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "-*[]\"`")
		if line != "" && !strings.HasPrefix(line, "#") {
			items = append(items, line)
		}
	}

	return limitItems(items)
}

// limitItems limits the slice to max 5 items.
func limitItems(items []string) []string {
	if len(items) > 5 {
		return items[:5]
	}
	return items
}

func truncateForLLM(content string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(content) <= maxChars {
		return content
	}
	return content[:maxChars]
}
