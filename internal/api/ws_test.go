package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/rawnaqs/khayal/internal/events"
)

func dialWS(t *testing.T, srvURL string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	return websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srvURL, "http")+"/v1/queue/ws", nil)
}

func authFrame(token string) []byte {
	b, _ := json.Marshal(map[string]string{"type": "auth", "token": token})
	return b
}

func TestQueueWSAuth(t *testing.T) {
	wsAuthWait = 500 * time.Millisecond
	defer func() { wsAuthWait = 5 * time.Second }()

	ts := setupTestServer(t)
	defer ts.close()
	hub := events.NewHub()
	ts.Server.SetHub(hub)

	srv := httptest.NewServer(ts.Server.router)
	defer srv.Close()

	t.Run("no auth frame within window: closed unauthorized", func(t *testing.T) {
		conn, _, err := dialWS(t, srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		_, _, err = conn.ReadMessage() // server closes after the auth window
		if err == nil {
			t.Fatal("expected close, got message")
		}
		if !websocket.IsCloseError(err, websocket.ClosePolicyViolation) &&
			!strings.Contains(err.Error(), "unexpected EOF") &&
			!strings.Contains(err.Error(), "closed connection") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong token: closed with policy violation", func(t *testing.T) {
		conn, _, err := dialWS(t, srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, authFrame("wrong")); err != nil {
			t.Fatal(err)
		}
		_, msg, err := conn.ReadMessage()
		if err == nil {
			t.Fatal("expected close")
		}
		if !websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
			t.Fatalf("expected policy-violation close, got: %v (msg=%s)", err, msg)
		}
	})

	t.Run("valid token: authenticated then receives broadcasts", func(t *testing.T) {
		conn, _, err := dialWS(t, srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, authFrame("test-token")); err != nil {
			t.Fatal(err)
		}

		_, ack, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("auth ack: %v", err)
		}
		var ackEvt struct {
			Event string `json:"event"`
		}
		if json.Unmarshal(ack, &ackEvt) != nil || ackEvt.Event != "authenticated" {
			t.Fatalf("expected authenticated ack, got %s", ack)
		}

		got := make(chan events.Event, 1)
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				var evt events.Event
				if json.Unmarshal(msg, &evt) == nil && evt.Event == "job_updated" {
					got <- evt
					return
				}
			}
		}()

		time.Sleep(100 * time.Millisecond) // let the subscriber attach
		hub.Publish(events.Event{Event: "job_updated"})

		select {
		case <-got:
		case <-time.After(3 * time.Second):
			t.Fatal("no broadcast received")
		}
	})
}
