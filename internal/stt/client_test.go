package stt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTranscribeOpenAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse: %v", err)
		}
		if r.FormValue("model") != "whisper-tiny" {
			t.Errorf("model: %s", r.FormValue("model"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"text":"hello from the fixture"}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1/audio/transcriptions", "openai", "whisper-tiny", 5*time.Second)
	text, err := c.Transcribe(context.Background(), "a.webm", []byte("AUDIO"), "audio/webm")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello from the fixture" {
		t.Errorf("got %q", text)
	}
}

func TestTranscribeWhisperCpp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inference" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse: %v", err)
		}
		if r.FormValue("response_format") != "text" {
			t.Errorf("response_format: %s", r.FormValue("response_format"))
		}
		w.Write([]byte("plain transcript"))
	}))
	defer srv.Close()

	c := New(srv.URL+"/inference", "whispercpp", "", 5*time.Second)
	text, err := c.Transcribe(context.Background(), "a.wav", []byte("AUDIO"), "audio/wav")
	if err != nil {
		t.Fatal(err)
	}
	if text != "plain transcript" {
		t.Errorf("got %q", text)
	}
}

func TestTranscribeServerErrorFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("model exploded"))
	}))
	defer srv.Close()

	c := New(srv.URL, "openai", "", 5*time.Second)
	if _, err := c.Transcribe(context.Background(), "a.wav", []byte("AUDIO"), "audio/wav"); err == nil {
		t.Error("expected error on 500")
	}
}

// Looping hallucinations ("Hello? x30", counting numbers) must be dropped:
// segments with extreme compression ratios or high no-speech probability
// are filtered; clean segments survive.
func TestTranscribeFiltersHallucinatedSegments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("response_format") != "verbose_json" {
			t.Errorf("expected verbose_json request, got %q", r.FormValue("response_format"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"text": "Hello? Hello? Hello? actual words here",
			"segments": [
				{"text": "Hello? Hello? Hello?", "compression_ratio": 13.5, "no_speech_prob": 0.49},
				{"text": "actual words here", "compression_ratio": 1.4, "no_speech_prob": 0.02}
			]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1/audio/transcriptions", "openai", "m", 5*time.Second)
	text, err := c.Transcribe(context.Background(), "a.webm", []byte("A"), "audio/webm")
	if err != nil {
		t.Fatal(err)
	}
	if text != "actual words here" {
		t.Errorf("looped segment survived: %q", text)
	}
}

// All-hallucinated audio (noise, music) yields an empty transcript so the
// caller reports "couldn't hear clear speech" instead of saving garbage.
func TestTranscribeAllHallucinatedYieldsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"text": "2 2 2 2 3 3 4 4",
			"segments": [
				{"text": "2 2 2 2 3 3 4 4", "compression_ratio": 8.0, "no_speech_prob": 0.7}
			]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1/audio/transcriptions", "openai", "m", 5*time.Second)
	text, err := c.Transcribe(context.Background(), "a.webm", []byte("A"), "audio/webm")
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("expected empty after filtering, got %q", text)
	}
}

// Plain-text responses (non-JSON) still work via fallback.
func TestTranscribePlainTextFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain transcript no json"))
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1/audio/transcriptions", "openai", "m", 5*time.Second)
	text, err := c.Transcribe(context.Background(), "a.wav", []byte("A"), "audio/wav")
	if err != nil {
		t.Fatal(err)
	}
	if text != "plain transcript no json" {
		t.Errorf("got %q", text)
	}
}
