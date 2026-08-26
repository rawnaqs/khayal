package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

const (
	TokenHeader = "X-Khayal-Token"
	// TokenQueryParam is accepted as a fallback for endpoints where
	// browsers cannot set custom headers — WebSocket handshakes.
	TokenQueryParam = "token"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

// Hijack forwards the WebSocket upgrade requirement through the wrapper.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hj.Hijack()
}

// Flush forwards the Flusher interface through the wrapper.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func AuthMiddleware(token string, writeError func(w http.ResponseWriter, message string, code string, status int)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientToken := r.Header.Get(TokenHeader)
			if clientToken == "" {
				clientToken = r.URL.Query().Get(TokenQueryParam)
			}
			if clientToken == "" {
				writeError(w, "token missing", "AUTH_TOKEN_MISSING", http.StatusUnauthorized)
				return
			}
			if clientToken != token {
				writeError(w, "invalid token", "AUTH_TOKEN_INVALID", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"panic", rec,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(rec, r)

			latency := time.Since(start)
			level := slog.LevelInfo
			if rec.status >= 400 {
				level = slog.LevelWarn
			}
			logger.Log(r.Context(), level, "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"latency_ms", latency.Milliseconds(),
				"ip", r.RemoteAddr,
			)
		})
	}
}

func GetTokenFromRequest(r *http.Request) string {
	return r.Header.Get(TokenHeader)
}
