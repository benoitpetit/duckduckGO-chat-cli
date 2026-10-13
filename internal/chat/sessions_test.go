package chat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"duckduckgo-chat-cli/internal/intelligence"
	"duckduckgo-chat-cli/internal/persistence"
)

func TestFilterSessionSummariesMatchesMetadataAndContent(t *testing.T) {
	sessions := []persistence.SessionSummary{
		{ID: "session_one", Title: "Release planning", Tags: []string{"Docs", "Go"}, Model: "gpt-5.6-luna", FirstMessage: "Plan the next release"},
		{ID: "session_two", Title: "Weekend ideas", Tags: []string{"Travel"}, Model: "gpt-5.6-sol", FirstMessage: "Where should I go?"},
	}
	for query, wantID := range map[string]string{
		"release": "session_one",
		"DOCS":    "session_one",
		"luna":    "session_one",
		"where":   "session_two",
	} {
		got := filterSessionSummaries(sessions, query)
		if len(got) != 1 || got[0].ID != wantID {
			t.Errorf("filterSessionSummaries(%q) = %+v, want only %s", query, got, wantID)
		}
	}
}

func TestBuildSessionManagerOptionsKeepsSessionIDsAligned(t *testing.T) {
	summaries := []persistence.SessionSummary{
		{ID: "session_one", Title: "First session", Tags: []string{"docs"}, Favorite: true, StartTime: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC), Model: "gpt-5.6-luna", MessageCount: 4, FirstMessage: "one"},
		{ID: "session_two", Title: "Second session", StartTime: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC), Model: "gpt-5.6-sol", MessageCount: 2, FirstMessage: "two"},
	}
	options, ids := buildSessionManagerOptions(summaries)
	if len(options) != len(summaries) || len(ids) != len(summaries) || ids[0] != summaries[0].ID || ids[1] != summaries[1].ID {
		t.Fatalf("options/IDs = %v / %v, want one-to-one summary order", options, ids)
	}
	if !strings.Contains(options[0], "First session") || !strings.Contains(options[0], "★") || !strings.Contains(options[0], "docs") {
		t.Fatalf("first session option = %q, want title, favorite, and tags", options[0])
	}
}

func TestSessionManagerMetadataActionsUpdateSelectedSession(t *testing.T) {
	manager := persistence.NewHistoryManager(t.TempDir())
	for _, session := range []*persistence.ConversationSession{
		{ID: "session_selected", StartTime: time.Now(), Title: "Before", Tags: []string{"old"}, Favorite: false, Messages: []intelligence.Message{{Role: "user", Content: "selected"}}},
		{ID: "session_other", StartTime: time.Now(), Title: "Other", Messages: []intelligence.Message{{Role: "user", Content: "other"}}},
	} {
		if err := manager.SaveSession(session); err != nil {
			t.Fatal(err)
		}
	}
	selected := persistence.SessionSummary{ID: "session_selected", Title: "Before", Tags: []string{"old"}}
	if err := renameManagedSession(manager, selected, "After"); err != nil {
		t.Fatalf("renameManagedSession() error = %v", err)
	}
	selected.Title = "After"
	if err := setManagedSessionTags(manager, selected, "planning, docs, PLANNING"); err != nil {
		t.Fatalf("setManagedSessionTags() error = %v", err)
	}
	selected.Tags = []string{"planning", "docs"}
	if err := toggleManagedSessionFavorite(manager, selected); err != nil {
		t.Fatalf("toggleManagedSessionFavorite() error = %v", err)
	}
	loaded, err := manager.LoadSession("session_selected")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "After" || strings.Join(loaded.Tags, ",") != "planning,docs" || !loaded.Favorite {
		t.Fatalf("selected metadata = title:%q tags:%v favorite:%t", loaded.Title, loaded.Tags, loaded.Favorite)
	}
	other, err := manager.LoadSession("session_other")
	if err != nil || other.Title != "Other" || len(other.Tags) != 0 || other.Favorite {
		t.Fatalf("other session metadata changed: (%+v, %v)", other, err)
	}
}

func TestSessionManagerDeleteRequiresConfirmationForSelectedID(t *testing.T) {
	manager := persistence.NewHistoryManager(t.TempDir())
	for _, id := range []string{"session_selected", "session_other"} {
		if err := manager.SaveSession(&persistence.ConversationSession{ID: id, StartTime: time.Now(), Title: id}); err != nil {
			t.Fatal(err)
		}
	}
	selected := persistence.SessionSummary{ID: "session_selected", Title: "Selected report"}
	if prompt := sessionDeleteConfirmation(selected); !strings.Contains(prompt, "Selected report") || !strings.Contains(prompt, selected.ID) {
		t.Fatalf("delete confirmation = %q, want selected title and ID", prompt)
	}
	if err := deleteManagedSession(manager, selected, false); err != nil {
		t.Fatalf("canceling delete returned error: %v", err)
	}
	if _, err := manager.LoadSession(selected.ID); err != nil {
		t.Fatalf("canceling delete removed selected session: %v", err)
	}
	if err := deleteManagedSession(manager, selected, true); err != nil {
		t.Fatalf("confirmed delete error = %v", err)
	}
	if _, err := manager.LoadSession(selected.ID); err == nil {
		t.Fatal("confirmed delete kept the selected session")
	}
	if _, err := manager.LoadSession("session_other"); err != nil {
		t.Fatalf("delete removed another session: %v", err)
	}
}

func TestSessionManagerMetadataActionsReportInvalidMetadata(t *testing.T) {
	manager := persistence.NewHistoryManager(t.TempDir())
	if err := manager.SaveSession(&persistence.ConversationSession{ID: "session_invalid_metadata", StartTime: time.Now()}); err != nil {
		t.Fatal(err)
	}
	selected := persistence.SessionSummary{ID: "session_invalid_metadata", Title: "Title"}
	if err := renameManagedSession(manager, selected, strings.Repeat("界", 101)); err == nil {
		t.Fatal("renameManagedSession() accepted a title beyond the documented limit")
	}
}

func TestDeletedManagerSessionRemovesOnlyItsArchive(t *testing.T) {
	dir := t.TempDir()
	manager := persistence.NewHistoryManager(dir)
	for _, id := range []string{"session_selected", "session_other"} {
		if err := manager.SaveSession(&persistence.ConversationSession{ID: id, StartTime: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	selected := persistence.SessionSummary{ID: "session_selected", Title: "Selected"}
	if err := deleteManagedSession(manager, selected, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "session_session_other.json.gz")); err != nil {
		t.Fatalf("other session archive was changed: %v", err)
	}
}
