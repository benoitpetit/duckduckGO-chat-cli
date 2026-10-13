package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"duckduckgo-chat-cli/internal/chat"
	"duckduckgo-chat-cli/internal/config"
)

func TestAPIKeyMiddlewareProtectsVersionedAPI(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{APIKey: "secret", ShowGinLogs: false}}
	router := setupRouter(NewSession(&chat.Chat{}, cfg), cfg)

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("status without key = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	authorized := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	request.Header.Set("X-API-Key", "secret")
	router.ServeHTTP(authorized, request)
	if authorized.Code != http.StatusOK {
		t.Fatalf("status with key = %d, want %d", authorized.Code, http.StatusOK)
	}
}

func TestCORSOnlyAllowsConfiguredOrigins(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{AllowedOrigins: []string{"https://example.test"}, ShowGinLogs: false}}
	router := setupRouter(NewSession(&chat.Chat{}, cfg), cfg)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://not-allowed.test")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected CORS origin %q", got)
	}
}

func TestAPIStreamRequiresKeyAndUsesConfiguredCORS(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{
		APIKey: "stream-secret", AllowedOrigins: []string{"https://allowed.test"}, ShowGinLogs: false,
	}}
	router := setupRouter(NewSession(&chat.Chat{}, cfg), cfg)

	unauthorized := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", bytes.NewBufferString(`{"message":"hello"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(unauthorized, request)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("stream status without API key = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	for _, test := range []struct {
		origin string
		want   string
	}{
		{origin: "https://allowed.test", want: "https://allowed.test"},
		{origin: "https://blocked.test", want: ""},
	} {
		t.Run(test.origin, func(t *testing.T) {
			preflight := httptest.NewRecorder()
			options := httptest.NewRequest(http.MethodOptions, "/api/v1/chat/stream", nil)
			options.Header.Set("Origin", test.origin)
			options.Header.Set("Access-Control-Request-Method", http.MethodPost)
			router.ServeHTTP(preflight, options)
			if preflight.Code != http.StatusNoContent || preflight.Header().Get("Access-Control-Allow-Origin") != test.want {
				t.Fatalf("CORS preflight status=%d allow-origin=%q, want %q", preflight.Code, preflight.Header().Get("Access-Control-Allow-Origin"), test.want)
			}
			if test.want != "" && !strings.Contains(preflight.Header().Get("Access-Control-Allow-Methods"), "POST") {
				t.Fatalf("stream CORS methods = %q, want POST", preflight.Header().Get("Access-Control-Allow-Methods"))
			}
		})
	}
}
