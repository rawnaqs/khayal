package api

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = (wsPongWait * 9) / 10
)

// wsAuthWait bounds how long the server waits for the auth frame.
// Var so tests can shorten it.
var wsAuthWait = 5 * time.Second

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Auth happens via the first message frame; any origin may connect
	// (first-party PWA is same-origin, but installed PWAs may vary).
	CheckOrigin: func(r *http.Request) bool { return true },
}

// queueWSHandler streams live queue job updates over WebSocket.
//
// Auth: browsers cannot set custom headers on the handshake, and tokens
// in query parameters leak into access logs. Instead, the client must
// send {"type":"auth","token":"..."} as its first frame within
// wsAuthWait; until then nothing is streamed and bad auth closes the
// connection.
func (s *Server) queueWSHandler(w http.ResponseWriter, r *http.Request) {
	if s.hub == nil {
		WriteError(w, "realtime updates unavailable", "WS_UNAVAILABLE", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Warn("ws upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	if !s.wsAuthenticate(conn) {
		return
	}

	ch := s.hub.Subscribe()
	defer s.hub.Unsubscribe(ch)

	// Reader pump: drains client messages (pings handled by gorilla),
	// detects disconnects so the subscriber can be released.
	go func() {
		conn.SetReadLimit(512)
		_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
			return nil
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				s.hub.Unsubscribe(ch)
				return
			}
		}
	}()

	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteJSON(evt); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// wsAuthenticate blocks until the client presents a valid auth frame.
func (s *Server) wsAuthenticate(conn *websocket.Conn) bool {
	_ = conn.SetReadDeadline(time.Now().Add(wsAuthWait))
	defer func() { _ = conn.SetReadDeadline(time.Time{}) }()

	var frame struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&frame); err != nil ||
		frame.Type != "auth" ||
		frame.Token != s.config.Server.Token {
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"),
			time.Now().Add(wsWriteWait))
		return false
	}
	_ = conn.WriteJSON(map[string]string{"event": "authenticated"})
	return true
}
