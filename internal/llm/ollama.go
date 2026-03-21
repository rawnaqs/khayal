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
}

func NewOllamaClient(baseURL, embedModel, textModel, visionModel string) *OllamaClient {
	return NewOllamaClientWithConcurrency(baseURL, embedModel, textModel, visionModel, 4)
}

func NewOllamaClientWithConcurrency(baseURL, embedModel, textModel, visionModel string, maxConcurrent int) *OllamaClient {
	if maxConcurrent < 1 {
		maxConcurrent = 4
	}
	return &OllamaClient{
		baseURL:               strings.TrimSuffix(baseURL, "/"),
		embedModel:            embedModel,
		textModel:             textModel,
		visionModel:           visionModel,
		truncateTextTokens:    2000,
		truncateImageTokens:   3000,
		truncateArticleTokens: 12000,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		logger:    slog.Default(),
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

func NewOllamaClientWithLogger(baseURL, embedModel, textModel, visionModel string, logger *slog.Logger) *OllamaClient {
	c := NewOllamaClient(baseURL, embedModel, textModel, visionModel)
	if logger != nil {
		c.logger = logger
	}
	return c
}

func NewOllamaClientWithConfig(baseURL, embedModel, textModel, visionModel string, truncateText, truncateImage, truncateArticle int) *OllamaClient {
	client := NewOllamaClient(baseURL, embedModel, textModel, visionModel)
	client.truncateTextTokens = truncateText
	client.truncateImageTokens = truncateImage
	client.truncateArticleTokens = truncateArticle
	return client
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
	if c.semaphore == nil {
		return nil
	}
	select {
	case c.semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *OllamaClient) release() {
	if c.semaphore == nil {
		return
	}
	<-c.semaphore
}

func (c *OllamaClient) Type() string {
	return ProviderOllama
}

func (c *OllamaClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *OllamaClient) Embed(text string) ([]float32, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
		c.logger.Debug("llm embed failed",
			"model", c.embedModel,
			"error", err.Error(),
		)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("embed request failed with status %d", resp.StatusCode)
		c.logger.Debug("llm embed failed",
			"model", c.embedModel,
			"error", err.Error(),
		)
		return nil, err
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Debug("llm embed failed",
			"model", c.embedModel,
			"error", err.Error(),
		)
		return nil, err
	}

	if len(result.Embedding) == 0 {
		err := fmt.Errorf("empty embedding returned")
		c.logger.Debug("llm embed failed",
			"model", c.embedModel,
			"error", err.Error(),
		)
		return nil, err
	}

	c.logger.Debug("llm embed",
		"model", c.embedModel,
		"text_length", len(text),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result.Embedding, nil
}

func (c *OllamaClient) Generate(prompt string) (string, error) {
	return c.generateWithModel(c.textModel, prompt)
}

func (c *OllamaClient) generateWithModel(model, prompt string) (string, error) {
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
		"temperature": 0.7,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		c.logger.Debug("llm generate failed",
			"model", model,
			"error", err.Error(),
		)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("generate request failed with status %d", resp.StatusCode)
		c.logger.Debug("llm generate failed",
			"model", model,
			"error", err.Error(),
		)
		return "", err
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Debug("llm generate failed",
			"model", model,
			"error", err.Error(),
		)
		return "", err
	}

	c.logger.Debug("llm generate",
		"model", model,
		"prompt_length", len(prompt),
		"response_length", len(result.Response),
		"duration_ms", time.Since(start).Milliseconds(),
	)

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
		"prompt": "Describe this image in detail. Include any text visible in the image.",
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
		c.logger.Debug("llm vision failed",
			"model", c.visionModel,
			"image_path", imagePath,
			"error", err.Error(),
		)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("vision request failed with status %d", resp.StatusCode)
		c.logger.Debug("llm vision failed",
			"model", c.visionModel,
			"image_path", imagePath,
			"error", err.Error(),
		)
		return "", err
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Debug("llm vision failed",
			"model", c.visionModel,
			"image_path", imagePath,
			"error", err.Error(),
		)
		return "", err
	}

	c.logger.Debug("llm vision",
		"model", c.visionModel,
		"image_path", imagePath,
		"image_size", len(imgData),
		"description_length", len(result.Response),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result.Response, nil
}

func (c *OllamaClient) ExtractTags(content string, bucket string) ([]string, error) {
	truncated := truncateForLLM(content, c.truncateLimit(bucket))

	prompt := fmt.Sprintf(`Extract 3-5 relevant tags for the following content. 
Return only a JSON array of strings, nothing else. No markdown.

Content:
%s

Tags:`, truncated)

	result, err := c.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tags: %w", err)
	}

	var tags []string
	result = strings.TrimSpace(result)

	if strings.HasPrefix(result, "```") {
		lines := strings.Split(result, "\n")
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "```") {
				break
			}
			tags = append(tags, strings.TrimSpace(line))
		}
	} else {
		if err := json.Unmarshal([]byte(result), &tags); err != nil {
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				line = strings.Trim(line, "-*[]")
				if line != "" && !strings.HasPrefix(line, "#") {
					tags = append(tags, line)
				}
			}
		}
	}

	if len(tags) > 5 {
		tags = tags[:5]
	}

	return tags, nil
}

func (c *OllamaClient) Summarize(content string, bucket string) (string, error) {
	truncated := truncateForLLM(content, c.truncateLimit(bucket))

	prompt := fmt.Sprintf(`Summarize the following content in 2-3 sentences.

Content:
%s

Summary:`, truncated)

	result, err := c.Generate(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to summarize: %w", err)
	}

	return strings.TrimSpace(result), nil
}

func (c *OllamaClient) ExtractKeyIdeas(content string, bucket string) ([]string, error) {
	truncated := truncateForLLM(content, c.truncateLimit(bucket))

	prompt := fmt.Sprintf(`Extract 3-5 key ideas from the following content.
Return only a JSON array of strings, nothing else. No markdown.

Content:
%s

Key Ideas:`, truncated)

	result, err := c.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract key ideas: %w", err)
	}

	var ideas []string
	result = strings.TrimSpace(result)

	if strings.HasPrefix(result, "```") {
		lines := strings.Split(result, "\n")
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "```") {
				break
			}
			ideas = append(ideas, strings.TrimSpace(line))
		}
	} else {
		if err := json.Unmarshal([]byte(result), &ideas); err != nil {
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				line = strings.Trim(line, "-*[]")
				if line != "" && !strings.HasPrefix(line, "#") {
					ideas = append(ideas, line)
				}
			}
		}
	}

	if len(ideas) > 5 {
		ideas = ideas[:5]
	}

	return ideas, nil
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

func (c *OllamaClient) EmbedBatchWithModel(model string, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	prompts := make([]map[string]string, len(texts))
	for i, text := range texts {
		prompts[i] = map[string]string{"prompt": text}
	}

	reqBody := map[string]any{
		"model":   model,
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

	return result.Embeddings, nil
}

func truncateForLLM(content string, maxTokens int) string {
	maxChars := maxTokens * 4

	if len(content) <= maxChars {
		return content
	}

	truncated := content[:maxChars]
	lastPeriod := strings.LastIndex(truncated, ".")
	if lastPeriod > maxChars/2 {
		return truncated[:lastPeriod+1]
	}

	return truncated + "..."
}
