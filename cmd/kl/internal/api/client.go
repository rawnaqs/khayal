package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	Status       string       `json:"status"`
	Version      string       `json:"version"`
	Dependencies Dependencies `json:"dependencies"`
	Queue        QueueStats   `json:"queue"`
}

type Dependencies struct {
	DB    Dependency `json:"db"`
	Vault Dependency `json:"vault"`
	LLM   Dependency `json:"llm"`
}

type Dependency struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
	Host   string `json:"host,omitempty"`
}

type QueueStats struct {
	Pending    int `json:"pending"`
	Queued     int `json:"queued"`
	Processing int `json:"processing"`
	Done       int `json:"done"`
	Failed     int `json:"failed"`
}

type SearchResult struct {
	ID        string   `json:"id"`
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
	Total   int            `json:"total"`
	TookMs  int64          `json:"took_ms"`
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

func (c *Client) Search(query, mode string, limit, excerptLength int, from, to string, connections bool) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("mode", mode)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("excerpt_length", strconv.Itoa(excerptLength))
	if from != "" {
		params.Set("from", from)
	}
	if to != "" {
		params.Set("to", to)
	}
	if connections {
		params.Set("connections", "true")
	}
	searchURL := "/v1/search?" + params.Encode()

	resp, err := c.doRequest("GET", searchURL, nil)
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

type StreakStats struct {
	Current         int     `json:"current"`
	Best            int     `json:"best"`
	NextMilestone   int     `json:"next_milestone"`
	DaysToMilestone int     `json:"days_to_milestone"`
	ThisWeek        [7]bool `json:"this_week"`
}

type TodayStats struct {
	Count     int     `json:"count"`
	ByHour    [24]int `json:"by_hour"`
	AvgPerDay float64 `json:"avg_per_day"`
}

type VaultStats struct {
	TotalNotes    int    `json:"total_notes"`
	TodayDelta    int    `json:"today_delta"`
	LastCaptureAt string `json:"last_capture_at"`
	Last7Days     [7]int `json:"last_7_days"`
}

type StatsResponse struct {
	Streak StreakStats `json:"streak"`
	Today  TodayStats  `json:"today"`
	Vault  VaultStats  `json:"vault"`
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
