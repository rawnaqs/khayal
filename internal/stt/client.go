// Package stt clients speech-to-text services for voice captures.
package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Client transcribes audio via an external STT HTTP service.
type Client struct {
	endpoint string
	api      string // "openai" | "whispercpp"
	model    string
	http     *http.Client
}

// New creates a client for the given endpoint format.
func New(endpoint, api, model string, timeout time.Duration) *Client {
	return &Client{
		endpoint: endpoint,
		api:      api,
		model:    model,
		http:     &http.Client{Timeout: timeout},
	}
}

// Transcribe sends the audio bytes and returns the transcription text.
func (c *Client) Transcribe(ctx context.Context, filename string, audio []byte, contentType string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("write audio: %w", err)
	}
	if c.model != "" {
		if err := writer.WriteField("model", c.model); err != nil {
			return "", err
		}
	}
	if c.api == "whispercpp" {
		// whisper.cpp server: text response, temperature 0
		if err := writer.WriteField("response_format", "text"); err != nil {
			return "", err
		}
		if err := writer.WriteField("temperature", "0"); err != nil {
			return "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("stt request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return "", fmt.Errorf("stt service returned %d: %s", resp.StatusCode, snippet)
	}

	if c.api == "whispercpp" {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	// OpenAI-compatible: {"text": "..."}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode stt response: %w", err)
	}
	return parsed.Text, nil
}
