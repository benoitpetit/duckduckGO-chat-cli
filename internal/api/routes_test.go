package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"duckduckgo-chat-cli/internal/chat"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/models"
	"duckduckgo-chat-cli/internal/persistence"

	"github.com/gin-gonic/gin"
)

func newRouteTestServer(t *testing.T) (*gin.Engine, *chat.Chat) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{API: config.APIConfig{ShowGinLogs: false}}
	conversation := &chat.Chat{
		Model:          models.Default(),
		SessionID:      "session-test",
		HistoryManager: persistence.NewHistoryManager(t.TempDir()),
		Messages: []chat.Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "reply"},
			{Role: "user", Content: "second"},
		},
	}
	return setupRouter(NewSession(conversation, cfg), cfg), conversation
}

func newStreamingRouteTestServer(t *testing.T, cfg *config.Config, processor streamProcessor) (*gin.Engine, *chat.Chat, *Session) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	conversation := &chat.Chat{Model: models.Default(), SessionID: "stream-session", HistoryManager: persistence.NewHistoryManager(t.TempDir())}
	session := NewSession(conversation, cfg)
	session.streamProcessor = processor
	return setupRouter(session, cfg), conversation, session
}

func TestAPIStreamEmitsOrderedDeltasAndDoneMetadata(t *testing.T) {
	processor := func(_ context.Context, _ *chat.Chat, _ string, _ *config.Config, onDelta func(string) error) (string, error) {
		for _, delta := range []string{"Hello", ", ", "world"} {
			if err := onDelta(delta); err != nil {
				return "", err
			}
		}
		return "Hello, world", nil
	}
	router, _, _ := newStreamingRouteTestServer(t, &config.Config{}, processor)
	response := performJSONRequest(router, http.MethodPost, "/api/v1/chat/stream", ChatRequest{Message: "hello"})
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("stream status=%d content-type=%q body=%s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	body := response.Body.String()
	positions := []int{
		strings.Index(body, "event: start\n"),
		strings.Index(body, `"delta":"Hello"`),
		strings.Index(body, `"delta":", "`),
		strings.Index(body, `"delta":"world"`),
		strings.Index(body, "event: done\n"),
	}
	for i, position := range positions {
		if position < 0 || (i > 0 && position <= positions[i-1]) {
			t.Fatalf("SSE events are missing or out of order at index %d: positions=%v body=%s", i, positions, body)
		}
	}
	var done struct {
		Model     string       `json:"model"`
		MessageID string       `json:"message_id"`
		Metadata  ChatMetadata `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(sseDataForEvent(t, body, "done")), &done); err != nil {
		t.Fatalf("decode done event: %v", err)
	}
	if done.Model != string(models.Default()) || done.MessageID == "" || done.Metadata.ConversationCount != 0 {
		t.Fatalf("done payload = %+v", done)
	}
	if strings.Contains(sseDataForEvent(t, body, "done"), `"response"`) {
		t.Fatal("done event duplicated the response instead of letting clients join deltas")
	}
}

func TestAPIStreamEmitsTerminalErrorWithoutDone(t *testing.T) {
	processor := func(context.Context, *chat.Chat, string, *config.Config, func(string) error) (string, error) {
		return "", chat.ErrRateLimited
	}
	router, _, _ := newStreamingRouteTestServer(t, &config.Config{}, processor)
	response := performJSONRequest(router, http.MethodPost, "/api/v1/chat/stream", ChatRequest{Message: "hello"})
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "event: error\n") || strings.Contains(body, "event: done\n") || !strings.Contains(body, ErrorCodeRateLimit) {
		t.Fatalf("rate limit stream = status %d body %q", response.Code, body)
	}
}

func TestAPIStreamStopsAfterClientWriteFailure(t *testing.T) {
	var callbackErr error
	processor := func(_ context.Context, _ *chat.Chat, _ string, _ *config.Config, onDelta func(string) error) (string, error) {
		callbackErr = onDelta("partial")
		return "", callbackErr
	}
	router, _, _ := newStreamingRouteTestServer(t, &config.Config{}, processor)
	writer := &failAfterWritesResponseWriter{failAt: 2}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/stream", strings.NewReader(`{"message":"hello"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(writer, request)
	if callbackErr == nil || !errors.Is(callbackErr, io.ErrClosedPipe) {
		t.Fatalf("delta callback error = %v, want disconnected writer error", callbackErr)
	}
	if strings.Contains(writer.body.String(), "event: done") || strings.Contains(writer.body.String(), "event: error") {
		t.Fatalf("handler wrote a terminal event after disconnect: %q", writer.body.String())
	}
}

type failAfterWritesResponseWriter struct {
	header http.Header
	body   strings.Builder
	writes int
	failAt int
	status int
}

func (w *failAfterWritesResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *failAfterWritesResponseWriter) WriteHeader(status int) { w.status = status }

func (w *failAfterWritesResponseWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.writes >= w.failAt {
		return 0, io.ErrClosedPipe
	}
	return w.body.Write(data)
}

func (w *failAfterWritesResponseWriter) Flush() {}

func sseDataForEvent(t *testing.T, body, event string) string {
	t.Helper()
	for _, frame := range strings.Split(body, "\n\n") {
		if !strings.HasPrefix(frame, "event: "+event+"\n") {
			continue
		}
		for _, line := range strings.Split(frame, "\n") {
			if strings.HasPrefix(line, "data: ") {
				return strings.TrimPrefix(line, "data: ")
			}
		}
	}
	t.Fatalf("SSE event %q missing from %q", event, body)
	return ""
}

func TestAPIStreamSerializesRequestsForOneSession(t *testing.T) {
	firstEntered := make(chan struct{})
	secondEntered := make(chan struct{}, 1)
	releaseFirst := make(chan struct{})
	var calls atomic.Int32
	processor := func(_ context.Context, _ *chat.Chat, _ string, _ *config.Config, _ func(string) error) (string, error) {
		call := calls.Add(1)
		if call == 1 {
			close(firstEntered)
			<-releaseFirst
		} else {
			secondEntered <- struct{}{}
		}
		return fmt.Sprintf("response %d", call), nil
	}
	router, _, _ := newStreamingRouteTestServer(t, &config.Config{}, processor)
	firstDone := make(chan int, 1)
	secondDone := make(chan int, 1)
	go func() {
		firstDone <- performJSONRequest(router, http.MethodPost, "/api/v1/chat/stream", ChatRequest{Message: "first"}).Code
	}()
	select {
	case <-firstEntered:
	case <-time.After(time.Second):
		t.Fatal("first stream did not enter processor")
	}
	go func() {
		secondDone <- performJSONRequest(router, http.MethodPost, "/api/v1/chat/stream", ChatRequest{Message: "second"}).Code
	}()
	select {
	case <-secondEntered:
		t.Fatal("second stream entered while the first still held the session lock")
	case <-time.After(40 * time.Millisecond):
	}
	close(releaseFirst)
	for name, done := range map[string]<-chan int{"first": firstDone, "second": secondDone} {
		select {
		case status := <-done:
			if status != http.StatusOK {
				t.Errorf("%s stream status = %d", name, status)
			}
		case <-time.After(time.Second):
			t.Errorf("%s stream did not complete", name)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("processor calls = %d, want 2 serialized streams", calls.Load())
	}
}

func performJSONRequest(router http.Handler, method, path string, payload any) *httptest.ResponseRecorder {
	var body bytes.Buffer
	if payload != nil {
		_ = json.NewEncoder(&body).Encode(payload)
	}
	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func decodeAPIResponse(t *testing.T, response *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var result APIResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode API response: %v; body=%s", err, response.Body.String())
	}
	return result
}

func TestAPIRoutesModelSessionHistoryAndClearFlow(t *testing.T) {
	router, conversation := newRouteTestServer(t)

	modelsResponse := performJSONRequest(router, http.MethodGet, "/api/v1/models", nil)
	if modelsResponse.Code != http.StatusOK {
		t.Fatalf("GET /models status = %d, body=%s", modelsResponse.Code, modelsResponse.Body.String())
	}
	var modelEnvelope struct {
		Data ModelsResponse `json:"data"`
	}
	if err := json.Unmarshal(modelsResponse.Body.Bytes(), &modelEnvelope); err != nil {
		t.Fatal(err)
	}
	if modelEnvelope.Data.TotalModels != len(models.Available()) || len(modelEnvelope.Data.Models) != len(models.Available()) {
		t.Fatalf("model count = total:%d listed:%d, want %d", modelEnvelope.Data.TotalModels, len(modelEnvelope.Data.Models), len(models.Available()))
	}
	if modelEnvelope.Data.CurrentModel != string(models.Default()) {
		t.Fatalf("current model = %q, want %q", modelEnvelope.Data.CurrentModel, models.Default())
	}

	changeResponse := performJSONRequest(router, http.MethodPost, "/api/v1/models", ModelChangeRequest{Model: string(models.GPT5Luna)})
	if changeResponse.Code != http.StatusOK || conversation.Model != models.GPT5Luna {
		t.Fatalf("POST /models status=%d current model=%q body=%s", changeResponse.Code, conversation.Model, changeResponse.Body.String())
	}

	sessionResponse := performJSONRequest(router, http.MethodGet, "/api/v1/session", nil)
	var sessionEnvelope struct {
		Data struct {
			SessionID    string `json:"session_id"`
			CurrentModel string `json:"current_model"`
			MessageCount int    `json:"message_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(sessionResponse.Body.Bytes(), &sessionEnvelope); err != nil {
		t.Fatal(err)
	}
	if sessionResponse.Code != http.StatusOK || sessionEnvelope.Data.SessionID != "session-test" || sessionEnvelope.Data.CurrentModel != string(models.GPT5Luna) || sessionEnvelope.Data.MessageCount != 3 {
		t.Fatalf("GET /session status=%d data=%+v body=%s", sessionResponse.Code, sessionEnvelope.Data, sessionResponse.Body.String())
	}

	historyResponse := performJSONRequest(router, http.MethodGet, "/api/v1/history?limit=1&offset=1", nil)
	var historyEnvelope struct {
		Data HistoryResponse `json:"data"`
	}
	if err := json.Unmarshal(historyResponse.Body.Bytes(), &historyEnvelope); err != nil {
		t.Fatal(err)
	}
	if historyResponse.Code != http.StatusOK || historyEnvelope.Data.TotalMessages != 3 || len(historyEnvelope.Data.Messages) != 1 || historyEnvelope.Data.Messages[0].Content != "reply" {
		t.Fatalf("GET /history pagination status=%d data=%+v body=%s", historyResponse.Code, historyEnvelope.Data, historyResponse.Body.String())
	}

	clearResponse := performJSONRequest(router, http.MethodDelete, "/api/v1/history", nil)
	if clearResponse.Code != http.StatusOK {
		t.Fatalf("DELETE /history status = %d, body=%s", clearResponse.Code, clearResponse.Body.String())
	}
	if len(conversation.Messages) != 0 {
		t.Fatalf("DELETE /history left %d messages", len(conversation.Messages))
	}
}

func TestAPIRejectsInvalidHistoryPagination(t *testing.T) {
	router, _ := newRouteTestServer(t)
	for _, query := range []string{"limit=invalid", "limit=-2", "offset=invalid", "offset=-1", "offset=999999999999999999999999999999"} {
		t.Run(query, func(t *testing.T) {
			response := performJSONRequest(router, http.MethodGet, "/api/v1/history?"+query, nil)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("GET /history?%s status = %d, want %d; body=%s", query, response.Code, http.StatusBadRequest, response.Body.String())
			}
			if envelope := decodeAPIResponse(t, response); envelope.Success || envelope.Error == nil || envelope.Error.Code != ErrorCodeValidation {
				t.Fatalf("GET /history?%s error envelope = %+v", query, envelope)
			}
		})
	}
}

func TestAPIHistoryTreatsZeroLimitAsDefault(t *testing.T) {
	router, _ := newRouteTestServer(t)
	response := performJSONRequest(router, http.MethodGet, "/api/v1/history?limit=0", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("GET /history?limit=0 status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestAPIRejectsInvalidChatRequestsWithoutChangingModel(t *testing.T) {
	router, conversation := newRouteTestServer(t)

	for name, payload := range map[string]any{
		"empty message":     map[string]string{"message": ""},
		"unknown model":     map[string]string{"message": "hello", "model": "not-a-model"},
		"oversized message": map[string]string{"message": string(bytes.Repeat([]byte("x"), 10001))},
	} {
		t.Run(name, func(t *testing.T) {
			response := performJSONRequest(router, http.MethodPost, "/api/v1/chat", payload)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("POST /chat status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}

	if conversation.Model != models.Default() {
		t.Fatalf("invalid chat request changed model to %q", conversation.Model)
	}
}

func TestAPIDeleteHistoryReportsPersistenceFailure(t *testing.T) {
	router, conversation := newRouteTestServer(t)
	conversation.HistoryManager = nil

	response := performJSONRequest(router, http.MethodDelete, "/api/v1/history", nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("DELETE /history status = %d, want %d; body=%s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	if len(conversation.Messages) == 0 {
		t.Fatal("DELETE /history discarded messages after the archive failed")
	}
	if envelope := decodeAPIResponse(t, response); envelope.Success || envelope.Error == nil || envelope.Error.Code != ErrorCodeInternal {
		t.Fatalf("DELETE /history error envelope = %+v", envelope)
	}
}

func TestAPIKeyMiddlewareAcceptsBearerAuthorization(t *testing.T) {
	cfg := &config.Config{API: config.APIConfig{APIKey: "secret", ShowGinLogs: false}}
	router := setupRouter(NewSession(&chat.Chat{}, cfg), cfg)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status with bearer key = %d, want %d", response.Code, http.StatusOK)
	}
}
