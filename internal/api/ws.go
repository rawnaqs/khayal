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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Token auth happens below; any origin may connect (first-party PWA
	// is same-origin, but installed PWAs may vary).
	CheckOrigin: func(r *http.Request) bool { return true },
}

// queueWSHandler streams live queue job updates over WebSocket.
//
// Auth: the regular X-Khayal-Token header cannot be set by browsers during
// the handshake, so the token must arrive as a query parameter. Requests
// are validated through the same middleware contract as every other route.
func (s *Server) queueWSHandler(w http.ResponseWriter, r *http.Request) {
	if s.hub == nil {
		WriteError(w, "realtime updates unavailable", "WS_UNAVAILABLE", http.StatusServiceUnavailable)
		return
	}
	if r.URL.Query().Get("token") != s.config.Server.Token {
		WriteError(w, "unauthorized", "UNAUTHORIZED", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Warn("ws upgrade failed", "error", err)
		return
	}
	defer conn.Close()

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
