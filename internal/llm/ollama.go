package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type OllamaClient struct {
	baseURL     string
	embedModel  string
	textModel   string
	visionModel string
	httpClient  *http.Client
}

func NewOllamaClient(baseURL, embedModel, textModel, visionModel string) *OllamaClient {
	return &OllamaClient{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		embedModel:  embedModel,
		textModel:   textModel,
		visionModel: visionModel,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return result.Embedding, nil
}

func (c *OllamaClient) Generate(prompt string) (string, error) {
	return c.generateWithModel(c.textModel, prompt)
}

func (c *OllamaClient) generateWithModel(model, prompt string) (string, error) {
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
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("generate request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Response, nil
}

func (c *OllamaClient) DescribeImage(imagePath string) (string, error) {
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

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

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vision request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Response, nil
}

func (c *OllamaClient) ExtractTags(content string) ([]string, error) {
	truncated := truncateForLLM(content, 2000)

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

func (c *OllamaClient) Summarize(content string) (string, error) {
	truncated := truncateForLLM(content, 4000)

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

func (c *OllamaClient) ExtractKeyIdeas(content string) ([]string, error) {
	truncated := truncateForLLM(content, 3000)

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
