package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	token := "test-token-123"
	var capturedCode string
	var capturedStatus int

	authMw := AuthMiddleware(token, func(w http.ResponseWriter, message string, code string, status int) {
		capturedCode = code
		capturedStatus = status
	})

	handler := authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TokenHeader, token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if capturedCode != "" || capturedStatus != 0 {
		t.Error("expected writeError not to be called for valid token")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	token := "test-token-123"
	var capturedCode string
	var capturedStatus int

	authMw := AuthMiddleware(token, func(w http.ResponseWriter, message string, code string, status int) {
		capturedCode = code
		capturedStatus = status
	})

	handler := authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TokenHeader, "wrong-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if capturedCode != "AUTH_TOKEN_INVALID" {
		t.Errorf("expected error code AUTH_TOKEN_INVALID, got %s", capturedCode)
	}
	if capturedStatus != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", capturedStatus)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	token := "test-token-123"
	var capturedCode string
	var capturedStatus int

	authMw := AuthMiddleware(token, func(w http.ResponseWriter, message string, code string, status int) {
		capturedCode = code
		capturedStatus = status
	})

	handler := authMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if capturedCode != "AUTH_TOKEN_MISSING" {
		t.Errorf("expected error code AUTH_TOKEN_MISSING, got %s", capturedCode)
	}
	if capturedStatus != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", capturedStatus)
	}
}

func TestRequestLogger(t *testing.T) {
	logger := slog.Default()
	middleware := RequestLogger(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRequestLogger_PanicRecovery(t *testing.T) {
	logger := slog.Default()
	middleware := RequestLogger(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 after panic, got %d", rec.Code)
	}
}

func TestGetTokenFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TokenHeader, "my-token")

	token := GetTokenFromRequest(req)

	if token != "my-token" {
		t.Errorf("expected token 'my-token', got '%s'", token)
	}
}
