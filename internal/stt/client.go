// Package stt clients speech-to-text services for voice captures.
package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"time"
)

// ErrTimeout is returned when the STT service accepts the request but
// does not answer within the client timeout — typically the model
// loading from disk (or downloading) after an idle period.
var ErrTimeout = errors.New("stt service timed out (model may be loading)")

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
	} else {
		// verbose_json exposes per-segment compression_ratio / no_speech_prob
		if err := writer.WriteField("response_format", "verbose_json"); err != nil {
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
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return "", ErrTimeout
		}
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

	// OpenAI-compatible: request verbose_json so hallucinated segments can
	// be detected and dropped. Small whisper models famously loop on short
	// clips ("Hello? Hello? x30", counting numbers) — those segments show
	// extreme compression ratios (normal speech stays under ~2.4) or
	// high no-speech probability.
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Text     string `json:"text"`
		Segments []struct {
			Text         string  `json:"text"`
			Compression  float64 `json:"compression_ratio"`
			NoSpeechProb float64 `json:"no_speech_prob"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		// plain-text response (whispercpp-style): return as-is
		return string(rawBody), nil
	}
	if len(parsed.Segments) == 0 {
		return strings.TrimSpace(parsed.Text), nil
	}

	var kept []string
	for _, seg := range parsed.Segments {
		if seg.Compression > maxSegmentCompression || seg.NoSpeechProb > maxNoSpeechProb {
			continue // hallucinated loop / silence
		}
		kept = append(kept, seg.Text)
	}
	return strings.TrimSpace(strings.Join(kept, " ")), nil
}

// thresholds mirror faster-whisper's own defaults
const (
	maxSegmentCompression = 2.4
	maxNoSpeechProb       = 0.9
)

// PreloadModel asks a speaches-style service to load the model from its
// disk cache into memory (POST {base}/v1/models/{model}). Best-effort:
// errors swallowed — this is a warmup, not a requirement. Generic
// OpenAI-compatible servers without this route just 404 harmlessly.
func (c *Client) PreloadModel(ctx context.Context) {
	if c.model == "" {
		return
	}
	base := c.endpoint
	for _, suffix := range []string{"/v1/audio/transcriptions", "/inference"} {
		base = strings.TrimSuffix(base, suffix)
	}
	callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, base+"/v1/models/"+c.model, nil)
	if err != nil {
		return
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// UnloadModel asks a speaches-style service to release the model from
// memory (DELETE {base}/api/ps/{model}). Best-effort by contract: the
// call uses its own short timeout and swallows all errors — the capture
// has already succeeded, and a hung unload must never delay the
// response. The base URL is derived from the transcription endpoint.
func (c *Client) UnloadModel(ctx context.Context) {
	if c.model == "" {
		return
	}
	base := c.endpoint
	for _, suffix := range []string{"/v1/audio/transcriptions", "/inference"} {
		base = strings.TrimSuffix(base, suffix)
	}
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodDelete, base+"/api/ps/"+c.model, nil)
	if err != nil {
		return
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
