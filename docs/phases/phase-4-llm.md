# Phase 4: LLM

> Local AI integration, fallbacks. Updated: 2026-03-17

## Goals

- [ ] LLM interface definition
- [ ] Ollama client (embed, generate, vision)
- [ ] Groq fallback
- [ ] OpenAI fallback
- [ ] Graceful degradation

## Directory Structure

```
internal/
├── llm/
│   ├── interface.go       # LLM interface definition
│   ├── ollama.go          # Ollama client (primary)
│   ├── groq.go            # Groq fallback
│   ├── openai.go          # OpenAI fallback
│   └── factory.go         # LLM factory with fallback
```

## Step 4.1: LLM Interface

**File:** `internal/llm/interface.go`

### Interface Definition

```go
type LLM interface {
    // Embed generates vector embedding for text
    Embed(text string) ([]float32, error)
    
    // Generate generates text from prompt
    Generate(prompt string) (string, error)
    
    // DescribeImage describes an image using vision model
    DescribeImage(path string) (string, error)
    
    // Ping checks if LLM is available
    Ping() error
    
    // Type returns the provider type
    Type() string
}

// Provider types
const (
    ProviderOllama = "ollama"
    ProviderGroq  = "groq"
    ProviderOpenAI = "openai"
)
```

### Extended Operations (for ingest)

```go
type LLMExt interface {
    LLM
    
    // ExtractTags extracts relevant tags from content
    ExtractTags(content string) ([]string, error)
    
    // Summarize generates a summary of content
    Summarize(content string) (string, error)
    
    // ExtractKeyIdeas extracts key ideas as bullet points
    ExtractKeyIdeas(content string) ([]string, error)
}
```

### Options

```go
type Options struct {
    Provider     string // "ollama" | "groq" | "openai"
    APIKey       string
    BaseURL      string // For Ollama: http://localhost:11434
    
    // Models
    EmbedModel   string // Default: "nomic-embed-text"
    TextModel    string // Default: "llama3.2:3b"
    VisionModel  string // Default: "moondream"
}

func NewLLM(opts Options) (LLM, error)
```

## Step 4.2: Ollama Client

**File:** `internal/llm/ollama.go`

### Requirements

- Primary LLM provider
- Local, private, no API costs
- Must implement Embed, Generate, DescribeImage, Ping

### Implementation

```go
type OllamaClient struct {
    baseURL    string
    embedModel string
    textModel  string
    visionModel string
    httpClient *http.Client
}

func NewOllamaClient(baseURL, embedModel, textModel, visionModel string) *OllamaClient {
    return &OllamaClient{
        baseURL:    strings.TrimSuffix(baseURL, "/"),
        embedModel: embedModel,
        textModel:  textModel,
        visionModel: visionModel,
        httpClient: &http.Client{Timeout: 120 * time.Second},
    }
}

func (c *OllamaClient) Type() string { return ProviderOllama }

func (c *OllamaClient) Ping() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("ollama not reachable: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("ollama returned status %d", resp.StatusCode)
    }
    return nil
}

func (c *OllamaClient) Embed(text string) ([]float32, error) {
    reqBody := map[string]interface{}{
        "model":   c.embedModel,
        "prompt":   text,
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
    
    var result struct {
        Embedding []float32 `json:"embedding"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Embedding, nil
}

func (c *OllamaClient) Generate(prompt string) (string, error) {
    reqBody := map[string]interface{}{
        "model":      c.textModel,
        "prompt":     prompt,
        "stream":     false,
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
    
    var result struct {
        Response string `json:"response"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    return result.Response, nil
}

func (c *OllamaClient) DescribeImage(imagePath string) (string, error) {
    // Read image file
    imgData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", err
    }
    
    // Convert to base64
    base64Img := base64.StdEncoding.EncodeToString(imgData)
    
    reqBody := map[string]interface{}{
        "model":   c.visionModel,
        "prompt":  "Describe this image in detail. Include any text visible in the image.",
        "images":  []string{base64Img},
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
    
    var result struct {
        Response string `json:"response"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    return result.Response, nil
}
```

### Extended Operations

```go
func (c *OllamaClient) ExtractTags(content string) ([]string, error) {
    prompt := fmt.Sprintf(`Extract 3-5 relevant tags for the following content. 
Return only a JSON array of strings, nothing else.

Content:
%s

Tags:`, content)
    
    result, err := c.Generate(prompt)
    if err != nil {
        return nil, err
    }
    
    // Parse JSON array from result
    var tags []string
    if err := json.Unmarshal([]byte(result), &tags); err != nil {
        // Fallback: extract from plain text
        lines := strings.Split(result, "\n")
        for _, line := range lines {
            line = strings.TrimSpace(line)
            if line != "" && !strings.HasPrefix(line, "#") {
                tags = append(tags, strings.Trim(line, "-* "))
            }
        }
    }
    
    return tags, nil
}

func (c *OllamaClient) Summarize(content string) (string, error) {
    prompt := fmt.Sprintf(`Summarize the following content in 2-3 sentences.

Content:
%s

Summary:`, content)
    
    return c.Generate(prompt)
}

func (c *OllamaClient) ExtractKeyIdeas(content string) ([]string, error) {
    prompt := fmt.Sprintf(`Extract 3-5 key ideas from the following content.
Return only a JSON array of strings, nothing else.

Content:
%s

Key Ideas:`, content)
    
    result, err := c.Generate(prompt)
    if err != nil {
        return nil, err
    }
    
    var ideas []string
    if err := json.Unmarshal([]byte(result), &ideas); err != nil {
        return nil, err
    }
    
    return ideas, nil
}
```

## Step 4.3: Groq Client

**File:** `internal/llm/groq.go`

### Requirements

- Fast inference API
- Fallback when Ollama unavailable
- Same interface implementation

```go
type GroqClient struct {
    apiKey     string
    textModel  string
    httpClient *http.Client
}

func NewGroqClient(apiKey, textModel string) *GroqClient {
    return &GroqClient{
        apiKey:     apiKey,
        textModel:  textModel,
        httpClient: &http.Client{Timeout: 60 * time.Second},
    }
}

func (c *GroqClient) Type() string { return ProviderGroq }

func (c *GroqClient) Ping() error {
    req, _ := http.NewRequest("GET", "https://api.groq.com/openai/v1/models", nil)
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

func (c *GroqClient) Embed(text string) ([]float32, error) {
    // Groq doesn't have embeddings API, use a small model as fallback
    // Or use a different embedding service
    return nil, fmt.Errorf("embeddings not supported on Groq")
}

func (c *GroqClient) Generate(prompt string) (string, error) {
    reqBody := map[string]interface{}{
        "model":       c.textModel,
        "messages":    []map[string]string{{"role": "user", "content": prompt}},
        "temperature": 0.7,
    }
    
    body, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    if len(result.Choices) == 0 {
        return "", fmt.Errorf("no completion returned")
    }
    
    return result.Choices[0].Message.Content, nil
}

func (c *GroqClient) DescribeImage(path string) (string, error) {
    // Groq doesn't support vision, use OpenAI as fallback for this
    return "", fmt.Errorf("vision not supported on Groq")
}
```

## Step 4.4: OpenAI Client

**File:** `internal/llm/openai.go`

### Requirements

- Universal fallback
- GPT-4 for vision
- Text models for generation

```go
type OpenAIClient struct {
    apiKey     string
    textModel  string
    visionModel string
    httpClient *http.Client
}

func NewOpenAIClient(apiKey, textModel, visionModel string) *OpenAIClient {
    return &OpenAIClient{
        apiKey:     apiKey,
        textModel:  textModel,
        visionModel: visionModel,
        httpClient: &http.Client{Timeout: 120 * time.Second},
    }
}

func (c *OpenAIClient) Type() string { return ProviderOpenAI }

func (c *OpenAIClient) Ping() error {
    req, _ := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

func (c *OpenAIClient) Embed(text string) ([]float32, error) {
    reqBody := map[string]interface{}{
        "model":  "text-embedding-3-small",
        "input":  text,
    }
    
    body, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
        } `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if len(result.Data) == 0 {
        return nil, fmt.Errorf("no embedding returned")
    }
    
    return result.Data[0].Embedding, nil
}

func (c *OpenAIClient) Generate(prompt string) (string, error) {
    reqBody := map[string]interface{}{
        "model":       c.textModel,
        "messages":    []map[string]string{{"role": "user", "content": prompt}},
        "temperature": 0.7,
    }
    
    body, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    if len(result.Choices) == 0 {
        return "", fmt.Errorf("no completion returned")
    }
    
    return result.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) DescribeImage(imagePath string) (string, error) {
    imgData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", err
    }
    
    // Convert to base64
    base64Img := base64.StdEncoding.EncodeToString(imgData)
    
    reqBody := map[string]interface{}{
        "model": c.visionModel,
        "messages": []map[string]interface{}{
            {
                "role": "user",
                "content": []map[string]interface{}{
                    {"type": "text", "text": "Describe this image in detail."},
                    {"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64," + base64Img}},
                },
            },
        },
    }
    
    body, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    if len(result.Choices) == 0 {
        return "", fmt.Errorf("no completion returned")
    }
    
    return result.Choices[0].Message.Content, nil
}
```

## Step 4.5: Factory (No Auto-Fallback)

**File:** `internal/llm/factory.go`

**Design Decision:** No automatic fallback. If primary fails, job stays in queue for user to retry.

```go
func NewLLM(cfg config.LLMConfig) (LLM, error) {
    var client LLM
    var err error
    
    switch cfg.Provider {
    case ProviderOllama:
        client = NewOllamaClient(
            cfg.OllamaHost,
            cfg.EmbedModel,
            cfg.TextModel,
            cfg.VisionModel,
        )
    case ProviderGroq:
        client = NewGroqClient(cfg.FallbackAPIKey, cfg.TextModel)
    case ProviderOpenAI:
        client = NewOpenAIClient(cfg.FallbackAPIKey, cfg.TextModel, cfg.VisionModel)
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
    
    // Verify connectivity
    if err := client.Ping(); err != nil {
        return nil, fmt.Errorf("LLM provider %s unavailable: %w", cfg.Provider, err)
    }
    
    return client, nil
}
```

### Why No Auto-Fallback?

| Reason | Explanation |
|--------|-------------|
| User may not have fallback configured | Don't assume fallback exists |
| Prevent data loss | Without fallback, job stays in queue for retry |
| Clear failure mode | User knows exactly what failed and why |
| Simplicity | Single failure path is easier to debug |

### Worker Behavior Without Fallback

```
LLM call fails → Return error → Job stays "pending" → User can retry/discard
```

This ensures:
- No automatic silent switching
- User is aware of failures
- Data is preserved in queue
- User has full control over retry/discard

## Step 4.6: Adding New Providers (Adapter Pattern)

The LLM package uses the **adapter pattern** — adding a new provider requires no changes to existing code.

### Architecture

```
                    ┌─────────────────────┐
                    │   LLM Interface     │  (internal/llm/interface.go)
                    │   ─────────────     │
                    │ Embed()             │
                    │ Generate()          │
                    │ DescribeImage()     │
                    │ Ping()              │
                    │ Type()              │
                    └──────────┬──────────┘
                               │
      ┌────────────────────────┼────────────────────────┐
      │                        │                        │
      ▼                        ▼                        ▼
┌─────────────┐        ┌─────────────┐         ┌─────────────┐
│   Ollama   │        │    Groq     │         │   OpenAI    │
│  (primary) │        │  (fallback) │         │  (fallback) │
└─────────────┘        └─────────────┘         └─────────────┘
```

### Adding Anthropic (Claude)

1. Create `internal/llm/anthropic.go`:

```go
package llm

type AnthropicClient struct {
    apiKey      string
    textModel   string
    visionModel string
    httpClient  *http.Client
}

const ProviderAnthropic = "anthropic"

func NewAnthropicClient(apiKey, textModel, visionModel string) *AnthropicClient {
    return &AnthropicClient{
        apiKey:      apiKey,
        textModel:   textModel,
        visionModel: visionModel,
        httpClient:  &http.Client{Timeout: 120 * time.Second},
    }
}

func (c *AnthropicClient) Type() string { return ProviderAnthropic }

func (c *AnthropicClient) Ping() error {
    req, _ := http.NewRequest("GET", "https://api.anthropic.com/v1/messages", nil)
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

func (c *AnthropicClient) Embed(text string) ([]float32, error) {
    // Anthropic doesn't have embeddings, use a workaround or return error
    return nil, fmt.Errorf("embeddings not supported on Anthropic")
}

func (c *AnthropicClient) Generate(prompt string) (string, error) {
    reqBody := map[string]interface{}{
        "model":      c.textModel,
        "max_tokens": 1024,
        "messages":   []map[string]string{{"role": "user", "content": prompt}},
    }
    
    body, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result struct {
        Content []struct {
            Text string `json:"text"`
        } `json:"content"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    if len(result.Content) == 0 {
        return "", fmt.Errorf("no completion returned")
    }
    
    return result.Content[0].Text, nil
}

func (c *AnthropicClient) DescribeImage(imagePath string) (string, error) {
    // Anthropic Claude 3 supports vision
    imgData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", err
    }
    
    base64Img := base64.StdEncoding.EncodeToString(imgData)
    
    reqBody := map[string]interface{}{
        "model":      c.visionModel,
        "max_tokens": 1024,
        "messages": []map[string]interface{}{
            {
                "role": "user",
                "content": []map[string]interface{}{
                    {"type": "text", "text": "Describe this image in detail."},
                    {"type": "image", "source": map[string]string{"type": "base64", "media_type": "image/png", "data": base64Img}},
                },
            },
        },
    }
    
    body, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result struct {
        Content []struct {
            Text string `json:"text"`
        } `json:"content"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    if len(result.Content) == 0 {
        return "", fmt.Errorf("no completion returned")
    }
    
    return result.Content[0].Text, nil
}
```

2. Add to factory in `internal/llm/factory.go`:

```go
case ProviderAnthropic:
    primary = NewAnthropicClient(cfg.FallbackAPIKey, cfg.TextModel, cfg.VisionModel)
```

3. Add to config in `internal/config/config.go`:

```go
type LLMConfig struct {
    Provider         string `yaml:"provider"` // "ollama" | "groq" | "openai" | "anthropic"
    // ... other fields
}
```

### Summary

To add a new provider:
1. Create `internal/llm/<provider>.go`
2. Implement the `LLM` interface
3. Add constructor and provider constant
4. Add case to factory switch
5. Add to config validation

**No other code changes required** — the rest of the codebase only knows about the interface.

## Step 4.7: Error Handling

Robust error handling is critical for unreliable models and networks.

### Error Types & Responses

| Error Type | Detection | User Response |
|------------|-----------|---------------|
| **Model not found** | API returns 404, model not in list | Warn user, suggest `ollama pull <model>` |
| **Inference timeout** | Context deadline exceeded | Retry with backoff, then fail gracefully |
| **OOM (out of memory)** | Process killed / API error | Fall back to lighter model or skip processing |
| **Invalid JSON output** | Parse error on LLM response | Retry with stricter prompt, fallback to raw |
| **Rate limiting** | HTTP 429 status | Exponential backoff, use fallback provider |
| **Provider down** | Connection refused | Switch to fallback provider |
| **Partial failure** | Some embeddings fail | Save partial, log affected items |

### Retry Logic Implementation

```go
// internal/llm/retry.go

type RetryConfig struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay time.Duration
}

var DefaultRetryConfig = RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 1 * time.Second,
    MaxDelay:     30 * time.Second,
}

func withRetry[T any](fn func() (T, error), cfg RetryConfig) (T, error) {
    var lastErr error
    
    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        result, err := fn()
        if err == nil {
            return result, nil
        }
        
        lastErr = err
        
        // Check if retryable
        if !isRetryable(err) {
            return result, err
        }
        
        // Calculate backoff
        delay := cfg.InitialDelay * time.Duration(math.Pow(2, float64(attempt)))
        if delay > cfg.MaxDelay {
            delay = cfg.MaxDelay
        }
        
        log.Warn().
            Int("attempt", attempt+1).
            Dur("delay", delay).
            Err(err).
            Msg("retrying LLM call")
        
        time.Sleep(delay)
    }
    
    return *new(T), fmt.Errorf("all retries failed: %w", lastErr)
}

func isRetryable(err error) bool {
    if err == nil {
        return false
    }
    
    // Timeout
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }
    
    errStr := err.Error()
    
    // Rate limiting
    if strings.Contains(errStr, "429") {
        return true
    }
    
    // Temporary unavailability
    if strings.Contains(errStr, "connection refused") ||
       strings.Contains(errStr, "temporary failure") ||
       strings.Contains(errStr, "service unavailable") {
        return true
    }
    
    // OOM
    if strings.Contains(errStr, "out of memory") ||
       strings.Contains(errStr, "insufficient memory") {
        return true
    }
    
    return false
}
```

### Safe JSON Parsing

```go
// Safe JSON parsing with fallback
func extractJSONArray(response string) ([]string, error) {
    // Try direct parse
    var result []string
    if err := json.Unmarshal([]byte(response), &result); err == nil {
        return result, nil
    }
    
    // Try to extract JSON from response
    // LLM might wrap in markdown or add commentary
    jsonStr := extractJSON(response)
    if jsonStr == "" {
        return nil, fmt.Errorf("no JSON found in response")
    }
    
    if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
        return result, nil
    }
    
    // Fallback: extract line by line
    lines := strings.Split(response, "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" && !strings.HasPrefix(line, "#") &&
           !strings.HasPrefix(line, "```") {
            result = append(result, strings.Trim(line, "-* "))
        }
    }
    
    if len(result) > 0 {
        return result, nil
    }
    
    return nil, fmt.Errorf("failed to parse JSON and fallback extraction")
}

func extractJSON(s string) string {
    // Find first [ and last ]
    start := strings.Index(s, "[")
    end := strings.LastIndex(s, "]")
    if start == -1 || end == -1 || end < start {
        return ""
    }
    return s[start : end+1]
}
```

### Error Handling Flow

When LLM fails, the error propagates to the worker which handles it:

```
LLM call → Error → Worker catches → Job stays "pending" → User can retry/discard
```

```go
// Worker handles LLM errors - no automatic fallback
func (w *Worker) processJob(jobID string) {
    job, err := w.queue.GetJob(jobID)
    if err != nil {
        log.Error().Err(err).Msg("failed to get job")
        return
    }
    
    // Process - don't write vault until all succeeds
    var notePath string
    notePath, err = w.ingestText(job) // This calls LLM
    
    if err != nil {
        // LLM failed - keep job pending for user to retry/discard
        w.handleFailure(job, err)
        return
    }
    
    // Only reached if LLM succeeded
    job.NotePath = notePath
    job.Status = "done"
    w.queue.UpdateJob(job)
}

func (w *Worker) handleFailure(job *queue.Job, err error) {
    job.Retries++
    job.Error = err.Error()
    job.Status = "pending" // Keep pending, not failed
    
    // Don't delete anything - user can retry
    w.queue.UpdateJob(job)
    
    log.Warn().Str("job", job.ID).Err(err).Msg("LLM failed, job pending for retry")
}
```

### User Flow for Failed Jobs

1. **User checks queue** → `GET /v1/queue?status=failed`
2. **See error message** → `"error": "ollama timeout after 120s"`
3. **Fix issue** → Start Ollama, pull required model
4. **Retry** → `POST /v1/queue/{id}/retry`
5. **Or discard** → `POST /v1/queue/{id}/discard`

This gives users full control over failed processing.
}
```

### Error Logging Guidelines

| Rule | Reason |
|------|--------|
| Log model name | Helps diagnose provider issues |
| Log prompt length | Helps with token limits |
| Log latency | Helps identify slow models |
| Never log full prompt | Security/privacy |
| Never log API keys | Security |
| Include original error | Debugging |
| Use structured logging | Searchable, parseable |

### User-Facing Error Messages

```go
// Internal error (detailed)
var internalErr = fmt.Errorf("ollama: text_model qwen2.5:3b timeout after 120s (attempt 3/3)")

// User-facing error (safe)
var userErr = "Processing failed. Try a smaller model or check Ollama is running."
```

**Never expose:**
- API keys
- Full prompts
- Internal error details
- Stack traces

### Health Check Enhancement

```go
// In health.go - check model availability
func (s *Server) checkLLMModels() []ModelStatus {
    models, err := s.llm.ListModels()
    if err != nil {
        return []ModelStatus{{Name: "all", Status: "error", Error: err.Error()}}
    }
    
    required := map[string]string{
        s.config.LLM.TextModel:   "text",
        s.config.LLM.VisionModel: "vision",
        s.config.LLM.EmbedModel:  "embed",
    }
    
    var statuses []ModelStatus
    for model, kind := range required {
        found := false
        for _, m := range models {
            if m.Name == model {
                found = true
                break
            }
        }
        
        status := ModelStatus{Name: model, Type: kind}
        if found {
            status.Status = "ok"
        } else {
            status.Status = "missing"
            status.Error = fmt.Sprintf("model not found. Run: ollama pull %s", model)
        }
        statuses = append(statuses, status)
    }
    
    return statuses
}
```

### Implementation Checklist

- [ ] Add retry logic with exponential backoff
- [ ] Implement `isRetryable()` function
- [ ] Add safe JSON parsing with fallback
- [ ] Implement fallback provider switching
- [ ] Add structured error logging
- [ ] Create user-safe error messages
- [ ] Enhance health endpoint with model checks
- [ ] Add model availability check on startup
- [ ] Document error codes for troubleshooting

## Testing

Write tests for:

- [ ] Ollama client (mock or live)
- [ ] Groq client (mock)
- [ ] OpenAI client (mock)
- [ ] Factory fallback logic
- [ ] Graceful degradation

```bash
go test ./internal/llm/... -v
```

## Checklist

- [ ] LLM interface
- [ ] Ollama client (embed, generate, vision)
- [ ] Ollama extended operations (tags, summary, key ideas)
- [ ] Groq client (optional provider)
- [ ] OpenAI client (optional provider)
- [ ] Factory (no auto-fallback - primary only)
- [ ] Retry logic with backoff (worker keeps job pending)
- [ ] Safe JSON parsing with fallback extraction
- [ ] Error logging (structured, safe)
- [ ] User-facing error messages
- [ ] Model availability check on startup
- [ ] Health endpoint model status
- [ ] Tests passing
- [ ] go vet clean

## Next Phase

[Phase 5: CLI](phase-5-cli.md)

## Notes

- Default models:
  - Embeddings: nomic-embed-text (274MB)
  - Text: qwen2.5:3b (structured output reliability)
  - Vision: moondream (fits in 8GB)
- All models configurable via config.yaml
- **No auto-fallback** - primary provider only
- LLM failure → job stays pending → user can retry or discard
- Retry with exponential backoff on transient errors
- Never expose API keys or full prompts in errors
