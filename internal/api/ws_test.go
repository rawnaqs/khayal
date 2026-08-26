package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/rawnaqs/khayal/internal/events"
)

func TestQueueWSAuth(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()
	hub := events.NewHub()
	ts.Server.SetHub(hub)

	srv := httptest.NewServer(ts.Server.router)
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/v1/queue/ws"

	t.Run("missing token rejected", func(t *testing.T) {
		_, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err == nil {
			t.Fatal("expected rejection without token")
		}
	})

	t.Run("wrong token rejected", func(t *testing.T) {
		_, _, err := websocket.DefaultDialer.Dial(url+"?token=wrong", nil)
		if err == nil {
			t.Fatal("expected rejection with wrong token")
		}
	})

	t.Run("valid token receives broadcasts", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(url+"?token=test-token", nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		defer conn.Close()

		got := make(chan events.Event, 1)
		go func() {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var evt events.Event
			if json.Unmarshal(msg, &evt) == nil {
				got <- evt
			}
		}()

		time.Sleep(100 * time.Millisecond) // let the subscriber attach
		hub.Publish(events.Event{Event: "job_updated", Job: nil})

		select {
		case evt := <-got:
			if evt.Event != "job_updated" {
				t.Errorf("unexpected event %q", evt.Event)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("no broadcast received")
		}
	})
}
