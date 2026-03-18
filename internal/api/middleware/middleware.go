package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

const (
	TokenHeader = "X-Khayal-Token"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
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
			logger.Info("request",
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
