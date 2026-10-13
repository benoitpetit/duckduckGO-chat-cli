package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"duckduckgo-chat-cli/internal/command"
	"duckduckgo-chat-cli/internal/config"
	"duckduckgo-chat-cli/internal/intelligence"
	"duckduckgo-chat-cli/internal/media"
	"duckduckgo-chat-cli/internal/persistence"
	"duckduckgo-chat-cli/internal/security"
)

func sessionTestServer(t *testing.T, show bool) (*Server, *persistence.HistoryManager) {
	t.Helper()
	history := persistence.NewHistoryManager(filepath.Join(t.TempDir(), "sessions"))
	server := NewServer(Dependencies{
		History:  NewHistoryStore(filepath.Join(t.TempDir(), "stats.jsonl"), 90),
		Sessions: history, Commands: command.GetCommandRegistry,
		Config: config.DashboardConfig{Port: 8765, RefreshIntervalSeconds: 3, RetentionDays: 90, ShowConversations: show, AnalysisTokenBudget: 8000},
	})
	return server, history
}

func TestSessionRoutesAreHiddenByDefault(t *testing.T) {
	server, _ := sessionTestServer(t, false)
	for _, path := range []string{"/api/sessions", "/api/sessions/session_123"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765"+path, nil)
		server.handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want 404", path, rr.Code)
		}
	}
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		request := httptest.NewRequest(method, "http://127.0.0.1:8765/api/sessions/session_123", strings.NewReader(`{"title":"hidden"}`))
		request.Header.Set("Origin", "http://127.0.0.1:8765")
		response := httptest.NewRecorder()
		server.handler().ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s hidden session status = %d, want 404", method, response.Code)
		}
	}
}

func TestSessionRoutesListSearchAndReadWhenEnabled(t *testing.T) {
	server, history := sessionTestServer(t, true)
	older := &persistence.ConversationSession{ID: "session_old", StartTime: time.Now().Add(-time.Hour), Model: "model-old", Messages: []intelligence.Message{{Role: "user", Content: "old preview"}}}
	newer := &persistence.ConversationSession{ID: "session_new", StartTime: time.Now(), Model: "model-new", Messages: []intelligence.Message{{Role: "user", Content: "needle preview"}, {Role: "assistant", Content: "private transcript"}}}
	if err := history.SaveSession(older); err != nil {
		t.Fatal(err)
	}
	if err := history.SaveSession(newer); err != nil {
		t.Fatal(err)
	}
	if err := history.UpdateSessionMetadata(newer.ID, persistence.SessionMetadata{Title: "Release research", Tags: []string{"work-notes"}, Favorite: true}); err != nil {
		t.Fatal(err)
	}
	list := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/sessions?query=NEEDLE", nil)
	server.handler().ServeHTTP(list, req)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", list.Code, list.Body.String())
	}
	var summaries []persistence.SessionSummary
	if err := json.Unmarshal(list.Body.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].ID != newer.ID || summaries[0].ResumeCommand != "/load session_new" {
		t.Fatalf("search results = %+v", summaries)
	}
	if strings.Contains(list.Body.String(), "private transcript") {
		t.Fatal("session list exposed transcript contents")
	}
	for query, wantID := range map[string]string{"release": newer.ID, "work-notes": newer.ID, "model-old": older.ID} {
		filtered := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/sessions?query="+url.QueryEscape(query), nil)
		server.handler().ServeHTTP(filtered, request)
		var results []persistence.SessionSummary
		if err := json.Unmarshal(filtered.Body.Bytes(), &results); err != nil {
			t.Fatal(err)
		}
		if filtered.Code != http.StatusOK || len(results) != 1 || results[0].ID != wantID {
			t.Errorf("query %q returned status=%d sessions=%+v, want %s", query, filtered.Code, results, wantID)
		}
	}
	var listed []persistence.SessionSummary
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil || len(listed) != 1 || listed[0].Title != "Release research" || !listed[0].Favorite || len(listed[0].Tags) != 1 {
		t.Fatalf("listed session metadata = %+v, error = %v", listed, err)
	}

	detail := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/sessions/session_new", nil)
	server.handler().ServeHTTP(detail, req)
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "private transcript") {
		t.Fatalf("detail response status=%d body=%s", detail.Code, detail.Body.String())
	}
}

func TestSessionMetadataPATCHAndDeleteRoutes(t *testing.T) {
	server, history := sessionTestServer(t, true)
	started := time.Now().Add(-time.Hour).Round(time.Second)
	session := &persistence.ConversationSession{ID: "session_mutate", StartTime: started, Model: "gpt", Messages: []intelligence.Message{{Role: "user", Content: "preserve this"}}, Analytics: persistence.SessionAnalytics{APICallsCount: 2}}
	if err := history.SaveSession(session); err != nil {
		t.Fatal(err)
	}
	savedEnd, savedAnalytics := session.EndTime, session.Analytics
	body, _ := json.Marshal(persistence.SessionMetadata{Title: "Renamed", Tags: []string{"priority"}, Favorite: true})
	request := httptest.NewRequest(http.MethodPatch, "http://127.0.0.1:8765/api/sessions/session_mutate", bytes.NewReader(body))
	request.Header.Set("Origin", "http://127.0.0.1:8765")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("PATCH status=%d body=%s", response.Code, response.Body.String())
	}
	updated, err := history.LoadSession(session.ID)
	if err != nil || updated.Title != "Renamed" || len(updated.Tags) != 1 || !updated.Favorite {
		t.Fatalf("updated session = (%+v, %v)", updated, err)
	}
	if !updated.StartTime.Equal(started) || !updated.EndTime.Equal(savedEnd) || updated.Analytics != savedAnalytics {
		t.Fatalf("PATCH changed conversation metadata: %+v", updated)
	}
	request = httptest.NewRequest(http.MethodDelete, "http://127.0.0.1:8765/api/sessions/session_mutate", nil)
	request.Header.Set("Origin", "http://127.0.0.1:8765")
	response = httptest.NewRecorder()
	server.handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("DELETE status=%d body=%s", response.Code, response.Body.String())
	}
	if _, err := history.LoadSession(session.ID); err == nil {
		t.Fatal("DELETE left the selected session in history")
	}
}

func TestSessionMutationRoutesRejectInvalidIDsAndInvalidMetadata(t *testing.T) {
	server, _ := sessionTestServer(t, true)
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		request := httptest.NewRequest(method, "http://127.0.0.1:8765/api/sessions/session_1", strings.NewReader(`{"title":"ok"}`))
		request.SetPathValue("id", "../outside")
		request.Header.Set("Origin", "http://127.0.0.1:8765")
		response := httptest.NewRecorder()
		if method == http.MethodPatch {
			server.handleSessionMetadataUpdate(response, request)
		} else {
			server.handleSessionDelete(response, request)
		}
		if response.Code != http.StatusBadRequest {
			t.Errorf("%s invalid ID status = %d, want 400", method, response.Code)
		}
	}
	request := httptest.NewRequest(http.MethodPatch, "http://127.0.0.1:8765/api/sessions/session_missing", strings.NewReader(`{"title":"`+strings.Repeat("x", 101)+`"}`))
	request.SetPathValue("id", "session_missing")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.handleSessionMetadataUpdate(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid metadata status = %d, want 400", response.Code)
	}
}

func TestSessionMutationRoutesRequireSameOrigin(t *testing.T) {
	server, _ := sessionTestServer(t, true)
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		for _, origin := range []string{"", "http://evil.example"} {
			request := httptest.NewRequest(method, "http://127.0.0.1:8765/api/sessions/session_123", strings.NewReader(`{"title":"ok"}`))
			if origin != "" {
				request.Header.Set("Origin", origin)
			}
			response := httptest.NewRecorder()
			server.handler().ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Errorf("%s Origin %q status = %d, want 403", method, origin, response.Code)
			}
		}
	}
}

func TestSessionMutationRoutesRequireDashboardAuthentication(t *testing.T) {
	server, history := sessionTestServer(t, true)
	salt, hash, err := security.HashPassword(authTestPassword)
	if err != nil {
		t.Fatal(err)
	}
	settings := server.currentSettings()
	settings.PasswordSalt, settings.PasswordHash = salt, hash
	server.UpdateSettings(settings)
	if err := history.SaveSession(&persistence.ConversationSession{ID: "session_auth", StartTime: time.Now(), Title: "Before"}); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		request := httptest.NewRequest(method, "http://"+server.expectedHost()+"/api/sessions/session_123", strings.NewReader(`{"title":"ok"}`))
		request.Header.Set("Origin", "http://"+server.expectedHost())
		response := httptest.NewRecorder()
		server.handler().ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("unauthenticated %s status = %d, want 401", method, response.Code)
		}
	}
	cookie := loginCookie(t, server)
	patch := authRequest(server, http.MethodPatch, "/api/sessions/session_auth", []byte(`{"title":"After","tags":[],"favorite":false}`), cookie)
	if patch.Code != http.StatusOK {
		t.Fatalf("authenticated PATCH status=%d body=%s", patch.Code, patch.Body.String())
	}
	deleted := authRequest(server, http.MethodDelete, "/api/sessions/session_auth", nil, cookie)
	if deleted.Code != http.StatusOK {
		t.Fatalf("authenticated DELETE status=%d body=%s", deleted.Code, deleted.Body.String())
	}
}

func TestSessionDetailRejectsInvalidIDsBeforeLoading(t *testing.T) {
	server, _ := sessionTestServer(t, true)
	for _, id := range []string{"", "session/a", "../session_1", "session_" + strings.Repeat("a", 100)} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/sessions/session_1", nil)
		req.SetPathValue("id", id)
		server.handleSessionDetail(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("id %q status = %d, want 400", id, rr.Code)
		}
	}
}

func TestSessionListQueryIsLocalAndBounded(t *testing.T) {
	server, _ := sessionTestServer(t, true)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/sessions?query="+url.QueryEscape(strings.Repeat("x", 300)), nil)
	server.handleSessions(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("oversized query status = %d, want 400", rr.Code)
	}
}

func TestSessionDetailRedactsImageBytes(t *testing.T) {
	server, history := sessionTestServer(t, true)
	imageBytes := []byte("private-image-bytes")
	session := &persistence.ConversationSession{
		ID: "session_image", StartTime: time.Now(), Model: "gpt-5.6-luna",
		Messages: []intelligence.Message{{Role: "user", Content: "Describe this", Images: []media.ImageAttachment{{Name: "logo.png", MIMEType: "image/png", Data: imageBytes}}}},
	}
	if err := history.SaveSession(session); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/sessions/session_image", nil)
	server.handler().ServeHTTP(response, request)
	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("detail status = %d body=%s", response.Code, body)
	}
	if !strings.Contains(body, "logo.png") || !strings.Contains(body, "image/png") {
		t.Fatalf("detail response omitted image metadata: %s", body)
	}
	if strings.Contains(body, "private-image-bytes") || strings.Contains(body, "cHJpdmF0ZS1pbWFnZS1ieXRlcw==") || strings.Contains(body, "data:image/") {
		t.Fatalf("detail response exposed image bytes: %s", body)
	}
}
