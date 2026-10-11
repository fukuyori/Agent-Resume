package agent

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"agres/internal/session"
)

func writeJSONLines(t *testing.T, path string, entries ...any) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, entry := range entries {
		if err := enc.Encode(entry); err != nil {
			t.Fatal(err)
		}
	}
}

func TestClaudeSearchSessionSearchesMessagesButNotToolResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeJSONLines(t, path,
		map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": "Authentication Error"}},
		map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "content": "tool-only-secret"}}}},
		map[string]any{"type": "assistant", "message": map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "Fixed successfully"}}}},
	)
	d := &ClaudeDetector{}
	s := session.Session{Paths: []string{path}}
	for _, tt := range []struct {
		keyword string
		want    bool
	}{
		{"authentication error", true},
		{"FIXED SUCCESSFULLY", true},
		{"tool-only-secret", false},
	} {
		got, err := d.SearchSession(s, tt.keyword)
		if err != nil || got != tt.want {
			t.Errorf("SearchSession(%q) = %v, %v; want %v", tt.keyword, got, err, tt.want)
		}
	}
}

func TestCodexSearchSessionSearchesUserAndAssistantMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	writeJSONLines(t, path,
		map[string]any{"type": "event_msg", "payload": map[string]any{"type": "user_message", "message": "Find the login bug"}},
		map[string]any{"type": "response_item", "payload": map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": "The bug is fixed"}}}},
		map[string]any{"type": "response_item", "payload": map[string]any{"type": "message", "role": "user", "content": []any{map[string]any{"type": "input_text", "text": "<environment_context>metadata-only-secret</environment_context>"}}}},
		map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "output": "tool-only-secret"}},
	)
	d := &CodexDetector{}
	s := session.Session{Paths: []string{path}}
	for _, tt := range []struct {
		keyword string
		want    bool
	}{
		{"LOGIN BUG", true},
		{"bug is fixed", true},
		{"metadata-only-secret", false},
		{"tool-only-secret", false},
	} {
		got, err := d.SearchSession(s, tt.keyword)
		if err != nil || got != tt.want {
			t.Errorf("SearchSession(%q) = %v, %v; want %v", tt.keyword, got, err, tt.want)
		}
	}
}

func TestOpenCodeSearchSessionSearchesTextPartsOnly(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "opencode.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, data TEXT NOT NULL);
		CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, data TEXT NOT NULL);
		INSERT INTO message VALUES
			('m1', 'session-a', '{"role":"user"}'),
			('m2', 'session-a', '{"role":"assistant"}');
		INSERT INTO part VALUES
			('p1', 'm1', 'session-a', '{"type":"text","text":"Search this request"}'),
			('p2', 'm2', 'session-a', '{"type":"text","text":"Search this response"}'),
			('p3', 'm2', 'session-a', '{"type":"tool","text":"tool-only-secret"}');
	`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	d := &OpenCodeDetector{dir: root}
	s := session.Session{ID: "session-a"}
	for _, tt := range []struct {
		keyword string
		want    bool
	}{
		{"THIS REQUEST", true},
		{"this response", true},
		{"tool-only-secret", false},
	} {
		got, err := d.SearchSession(s, tt.keyword)
		if err != nil || got != tt.want {
			t.Errorf("SearchSession(%q) = %v, %v; want %v", tt.keyword, got, err, tt.want)
		}
	}
}

func TestAiderSearchSessionSearchesEntireSection(t *testing.T) {
	cwd := t.TempDir()
	padding := make([]byte, 100)
	for i := range padding {
		padding[i] = 'x'
	}
	content := "# Session\n" + string(padding) + " final-keyword\n"
	if err := os.WriteFile(filepath.Join(cwd, ".aider.chat.history.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &AiderDetector{}
	sessions, err := d.ListSessions(cwd, false)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("ListSessions() = %#v, %v", sessions, err)
	}
	matched, err := d.SearchSession(sessions[0], "FINAL-KEYWORD")
	if err != nil || !matched {
		t.Fatalf("SearchSession() = %v, %v; want match", matched, err)
	}
}

func TestAntigravitySearchSessionSearchesConversationTextOnly(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "brain", "session-a", ".system_generated", "logs", "transcript.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONLines(t, path,
		map[string]any{"source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "<USER_REQUEST>Find authentication after a deliberately long prefix that exceeds the title truncation boundary and ends with final-keyword</USER_REQUEST><ADDITIONAL_METADATA>metadata-only-secret</ADDITIONAL_METADATA>"},
		map[string]any{"source": "MODEL", "type": "PLANNER_RESPONSE", "content": "Authentication was fixed"},
		map[string]any{"source": "MODEL", "type": "RUN_COMMAND", "content": "tool-only-secret"},
	)
	d := &AntigravityDetector{dir: root}
	s := session.Session{ID: "session-a"}
	for _, tt := range []struct {
		keyword string
		want    bool
	}{
		{"FIND AUTHENTICATION", true},
		{"final-keyword", true},
		{"was fixed", true},
		{"metadata-only-secret", false},
		{"tool-only-secret", false},
	} {
		got, err := d.SearchSession(s, tt.keyword)
		if err != nil || got != tt.want {
			t.Errorf("SearchSession(%q) = %v, %v; want %v", tt.keyword, got, err, tt.want)
		}
	}
}
