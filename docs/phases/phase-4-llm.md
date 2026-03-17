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
│   ├── interface.go
│   ├── ollama.go
│   ├── groq.go
│   └── openai.go
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
        "input":   text,
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

## Step 4.5: Factory & Fallback

**File:** `internal/llm/factory.go`

```go
func NewLLM(cfg config.LLMConfig) (LLM, error) {
    // Try primary provider
    var primary LLM
    var err error
    
    switch cfg.Provider {
    case ProviderOllama:
        primary = NewOllamaClient(
            cfg.OllamaHost,
            cfg.EmbedModel,
            cfg.TextModel,
            cfg.VisionModel,
        )
    case ProviderGroq:
        primary = NewGroqClient(cfg.FallbackAPIKey, cfg.TextModel)
    case ProviderOpenAI:
        primary = NewOpenAIClient(cfg.FallbackAPIKey, cfg.TextModel, cfg.VisionModel)
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
    
    // Ping primary
    if err := primary.Ping(); err == nil {
        return primary, nil
    }
    
    // Try fallback
    if cfg.FallbackProvider == "" {
        return nil, fmt.Errorf("primary provider (%s) unavailable and no fallback configured", cfg.Provider)
    }
    
    var fallback LLM
    switch cfg.FallbackProvider {
    case ProviderOllama:
        fallback = NewOllamaClient(cfg.OllamaHost, cfg.EmbedModel, cfg.TextModel, cfg.VisionModel)
    case ProviderGroq:
        fallback = NewGroqClient(cfg.FallbackAPIKey, cfg.TextModel)
    case ProviderOpenAI:
        fallback = NewOpenAIClient(cfg.FallbackAPIKey, cfg.TextModel, cfg.VisionModel)
    }
    
    if err := fallback.Ping(); err != nil {
        return nil, fmt.Errorf("primary and fallback providers unavailable: %w", err)
    }
    
    log.Warn().Str("primary", cfg.Provider).Str("fallback", cfg.FallbackProvider).Msg("using fallback provider")
    
    return fallback, nil
}
```

### Graceful Degradation

If no LLM is available:
1. Save raw note to vault
2. Queue job for retry when LLM recovers
3. Mark status as `pending` not `failed`

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
- [ ] Groq fallback client
- [ ] OpenAI fallback client
- [ ] Factory with fallback logic
- [ ] Graceful degradation
- [ ] Tests passing
- [ ] go vet clean

## Next Phase

[Phase 5: CLI](phase-5-cli.md)

## Notes

- Default models:
  - Embeddings: nomic-embed-text (274MB)
  - Text: llama3.2:3b (2GB)
  - Vision: moondream (1.8GB)
- Fallback activates if Ollama unreachable
- If no fallback and Ollama down: raw note saved, job queued for retry
