# API Client Package

> Specification for `internal/api/client/`. Updated: 2026-03-17

## Overview

The `internal/api/client` package provides a typed Go client for the Khayal API. Used by:
- `kl` CLI
- Future interfaces (browser extension, iOS, Android wrappers)

## Design Goals

1. **Thin wrapper** - Just HTTP calls, no business logic
2. **Typed** - Full type safety for requests/responses
3. **Shared** - Single package for all Go clients
4. **Simple** - Minimal dependencies (just stdlib + Cobra)

## Usage

```go
import "github.com/rawnaqs/khayal/internal/api/client"

func main() {
    c := client.New(
        "http://localhost:1133",
        "your-token",
    )
    
    // Capture
    resp, err := c.CaptureText(context.Background(), "my thought")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Saved: %s\n", resp.NotePath)
}
```

## Package Structure

```
internal/api/client/
├── client.go        # Main client struct and options
├── capture.go       # Capture methods
├── search.go        # Search methods
├── queue.go         # Queue methods
├── health.go        # Health methods
└── types.go         # Request/response types
```

## Client Definition

```go
// client.go

type Client struct {
    baseURL string
    token   string
    http    *http.Client
}

type Option func(*Client)

func New(baseURL, token string, opts ...Option) *Client {
    c := &Client{
        baseURL: strings.TrimSuffix(baseURL, "/"),
        token:   token,
        http:    &http.Client{Timeout: 30 * time.Second},
    }
    
    for _, opt := range opts {
        opt(c)
    }
    
    return c
}

func WithTimeout(timeout time.Duration) Option {
    return func(c *Client) {
        c.http.Timeout = timeout
    }
}

func (c *Client) request(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
    var rBody io.Reader
    if body != nil {
        b, _ := json.Marshal(body)
        rBody = bytes.NewReader(b)
    }
    
    req, _ := http.NewRequestWithContext(ctx, method, c.baseURL+path, rBody)
    req.Header.Set("X-Khayal-Token", c.token)
    req.Header.Set("Content-Type", "application/json")
    
    return c.http.Do(req)
}
```

## Capture Methods

```go
// capture.go

func (c *Client) CaptureText(ctx context.Context, content string) (*CaptureResponse, error) {
    resp, err := c.request(ctx, "POST", "/v1/capture", CaptureRequest{
        Type:    "text",
        Content: content,
    })
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result CaptureResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

func (c *Client) CaptureURL(ctx context.Context, url string) (*CaptureResponse, error) {
    resp, err := c.request(ctx, "POST", "/v1/capture", CaptureRequest{
        Type:    "url",
        Content: url,
    })
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result CaptureResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

func (c *Client) CaptureImage(ctx context.Context, imagePath, note string) (*CaptureResponse, error) {
    // Multipart upload
    var b bytes.Buffer
    w := multipart.NewWriter(&b)
    w.WriteField("type", "image")
    if note != "" {
        w.WriteField("note", note)
    }
    
    f, _ := os.Open(imagePath)
    defer f.Close()
    
    part, _ := w.CreateFormFile("file", filepath.Base(imagePath))
    io.Copy(part, f)
    w.Close()
    
    req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/capture", &b)
    req.Header.Set("X-Khayal-Token", c.token)
    req.Header.Set("Content-Type", w.FormDataContentType())
    
    resp, err := c.http.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result CaptureResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

## Search Methods

```go
// search.go

type SearchOptions struct {
    Limit         int
    Mode          string // "keyword", "semantic", "hybrid"
    ExcerptLength int
}

func (c *Client) Search(ctx context.Context, query string, opts ...SearchOptions) (*SearchResponse, error) {
    options := SearchOptions{
        Limit:         10,
        Mode:          "hybrid",
        ExcerptLength: 200,
    }
    if len(opts) > 0 {
        options = opts[0]
    }
    
    params := url.Values{
        "q": {query},
        "limit": {strconv.Itoa(options.Limit)},
        "mode": {options.Mode},
    }
    
    resp, err := c.request(ctx, "GET", "/v1/search?"+params.Encode(), nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result SearchResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

## Queue Methods

```go
// queue.go

type QueueFilter struct {
    Status string
    Limit  int
    Offset int
}

func (c *Client) ListQueue(ctx context.Context, filter QueueFilter) (*QueueListResponse, error) {
    params := url.Values{
        "limit": {strconv.Itoa(filter.Limit)},
        "offset": {strconv.Itoa(filter.Offset)},
    }
    if filter.Status != "" {
        params.Set("status", filter.Status)
    }
    
    resp, err := c.request(ctx, "GET", "/v1/queue?"+params.Encode(), nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result QueueListResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

func (c *Client) GetJob(ctx context.Context, id string) (*Job, error) {
    resp, err := c.request(ctx, "GET", "/v1/queue/"+id, nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result Job
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

func (c *Client) RetryJob(ctx context.Context, id string) (*Job, error) {
    resp, err := c.request(ctx, "POST", "/v1/queue/"+id+"/retry", nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result Job
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

func (c *Client) DiscardJob(ctx context.Context, id string) error {
    resp, err := c.request(ctx, "POST", "/v1/queue/"+id+"/discard", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

## Health Methods

```go
// health.go

func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
    resp, err := c.request(ctx, "GET", "/v1/health", nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result HealthResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

## Types

```go
// types.go

// Capture
type CaptureRequest struct {
    Type    string `json:"type"`
    Content string `json:"content"`
}

type CaptureResponse struct {
    ID        string `json:"id"`
    Type      string `json:"type"`
    Status    string `json:"status"`
    NotePath  string `json:"note_path"`
    CreatedAt string `json:"created_at"`
}

// Search
type SearchResponse struct {
    Query   string         `json:"query"`
    Mode    string         `json:"mode"`
    Results []SearchResult `json:"results"`
    Total   int            `json:"total"`
    TookMs  int64         `json:"took_ms"`
}

type SearchResult struct {
    ID        string   `json:"id"`
    NotePath  string   `json:"note_path"`
    Title     string   `json:"title"`
    Excerpt   string   `json:"excerpt"`
    Score     float64  `json:"score"`
    Type      string   `json:"type"`
    CreatedAt string   `json:"created_at"`
    Tags      []string `json:"tags,omitempty"`
}

// Queue
type QueueListResponse struct {
    Total  int   `json:"total"`
    Limit  int   `json:"limit"`
    Offset int   `json:"offset"`
    Jobs   []Job `json:"jobs"`
}

type Job struct {
    ID          string  `json:"id"`
    Type        string  `json:"type"`
    Status      string  `json:"status"`
    NotePath    string  `json:"note_path"`
    SourceURL   string  `json:"source_url,omitempty"`
    SourceFile  string  `json:"source_file,omitempty"`
    CreatedAt   string  `json:"created_at"`
    ProcessedAt string  `json:"processed_at,omitempty"`
    Error       string  `json:"error,omitempty"`
    Retries     int     `json:"retries"`
}

// Health
type HealthResponse struct {
    Status       string                `json:"status"`
    Version      string                `json:"version"`
    Dependencies map[string]Dependency `json:"dependencies"`
    Queue        QueueStats            `json:"queue"`
}

type Dependency struct {
    Status string `json:"status"`
    Host   string `json:"host,omitempty"`
    Path   string `json:"path,omitempty"`
}

type QueueStats struct {
    Pending    int `json:"pending"`
    Queued     int `json:"queued"`
    Processing int `json:"processing"`
    Done       int `json:"done"`
    Failed     int `json:"failed"`
}
```

## CLI Integration

The `kl` CLI uses this client:

```go
// cmd/kl/main.go

package main

import (
    "github.com/rawnaqs/khayal/internal/api/client"
    "github.com/spf13/cobra"
)

var (
    host  string
    token string
)

func main() {
    c := client.New(host, token)
    
    rootCmd.AddCommand(&cobra.Command{
        Use:   "capture [text]",
        Short: "Capture a thought",
        RunE: func(cmd *cobra.Command, args []string) error {
            resp, err := c.CaptureText(cmd.Context(), args[0])
            if err != nil {
                return err
            }
            fmt.Println("Saved:", resp.NotePath)
            return nil
        },
    })
}
```

## Error Handling

```go
func (c *Client) handleError(resp *http.Response) error {
    if resp.StatusCode < 400 {
        return nil
    }
    
    var errResp Error
    json.NewDecoder(resp.Body).Decode(&errResp)
    
    return &APIError{
        Code:    errResp.Code,
        Message: errResp.Error,
        Status:  resp.StatusCode,
    }
}

type APIError struct {
    Code    string
    Message string
    Status  int
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s (status %d)", e.Code, e.Message, e.Status)
}
```

## Notes

- No retry logic in client - let caller handle
- Context support for cancellation
- Timeout configurable
- Minimal dependencies (stdlib only for core)
- Can be used by any Go program (not just CLI)
