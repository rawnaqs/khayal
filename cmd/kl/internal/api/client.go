package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type Client struct {
	Host   string
	Token  string
	client *http.Client
}

type CaptureResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	NotePath  string `json:"note_path,omitempty"`
	CreatedAt string `json:"created_at"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Queue   struct {
		Pending    int `json:"pending"`
		Processing int `json:"processing"`
		Done       int `json:"done"`
		Failed     int `json:"failed"`
	} `json:"queue"`
}

type SearchResult struct {
	NotePath  string   `json:"note_path"`
	Title     string   `json:"title"`
	Score     float64  `json:"score"`
	Excerpt   string   `json:"excerpt"`
	Type      string   `json:"type"`
	CreatedAt string   `json:"created_at"`
	Tags      []string `json:"tags,omitempty"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Query   string         `json:"query"`
	Mode    string         `json:"mode"`
	Time    int64          `json:"time_ms"`
}

func NewClient(host, token string) *Client {
	return &Client{
		Host:  host,
		Token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := c.Host + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Khayal-Token", c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.client.Do(req)
}

func (c *Client) CaptureText(content string) (*CaptureResponse, error) {
	body := map[string]string{
		"type":    "text",
		"content": content,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/v1/capture", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("capture failed with status %d", resp.StatusCode)
	}

	var result CaptureResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) CaptureURL(url string) (*CaptureResponse, error) {
	body := map[string]string{
		"type":    "url",
		"content": url,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/v1/capture", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("capture failed with status %d", resp.StatusCode)
	}

	var result CaptureResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) CaptureImage(imagePath, note string) (*CaptureResponse, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	if note != "" {
		writer.WriteField("note", note)
	}

	writer.WriteField("type", "image")

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	url := c.Host + "/v1/capture"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Khayal-Token", c.Token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("capture failed with status %d", resp.StatusCode)
	}

	var result CaptureResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) Search(query string, mode string, limit int) (*SearchResponse, error) {
	url := fmt.Sprintf("/v1/search?q=%s&mode=%s&limit=%d", query, mode, limit)

	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) Status() (*HealthResponse, error) {
	resp, err := c.doRequest("GET", "/v1/health", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status check failed with status %d", resp.StatusCode)
	}

	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) CheckConnection() error {
	resp, err := c.client.Get(c.Host + "/v1/health")
	if err != nil {
		return fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid token")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

type StatsResponse struct {
	Total     int            `json:"total"`
	ByType    map[string]int `json:"by_type"`
	ByTag     map[string]int `json:"by_tag"`
	QueueSize int            `json:"queue_size"`
}

func (c *Client) Stats() (*StatsResponse, error) {
	resp, err := c.doRequest("GET", "/v1/stats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stats request failed with status %d", resp.StatusCode)
	}

	var result StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

type BrowseResponse struct {
	Notes []struct {
		Path    string   `json:"path"`
		Tags    []string `json:"tags"`
		Date    string   `json:"date"`
		Excerpt string   `json:"excerpt"`
	} `json:"notes"`
	Total int `json:"total"`
}

func (c *Client) Browse(filter string, value string, limit int) (*BrowseResponse, error) {
	url := fmt.Sprintf("/v1/browse?filter=%s&value=%s&limit=%d", filter, value, limit)
	resp, err := c.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("browse request failed with status %d", resp.StatusCode)
	}

	var result BrowseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
