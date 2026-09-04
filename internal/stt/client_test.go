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
