package agent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestExtractSessionIDFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{
			filename: "rollout-2026-07-07T09-22-18-019f39f4-46b1-72d0-bc02-4a2562b29027.jsonl",
			expected: "019f39f4-46b1-72d0-bc02-4a2562b29027",
		},
		{
			filename: "rollout-2026-03-26T16-21-10-019d2904-c0b9-7a92-96b8-a238db277d3f.jsonl",
			expected: "019d2904-c0b9-7a92-96b8-a238db277d3f",
		},
		{
			filename: "rollout-2026-09-04T11-57-08-01a06a59-3eba-7913-8f3b-c0d56c7ec3a6.jsonl.zst",
			expected: "01a06a59-3eba-7913-8f3b-c0d56c7ec3a6",
		},
		{
			filename: "custom_session.jsonl",
			expected: "custom_session",
		},
	}

	for _, tt := range tests {
		got := extractSessionIDFromFilename(tt.filename)
		if got != tt.expected {
			t.Errorf("extractSessionIDFromFilename(%q) = %q; want %q", tt.filename, got, tt.expected)
		}
	}
}

func TestCleanCodexText(t *testing.T) {
	input := "<environment_context>\n  <cwd>C:\\Users\\n_fuk</cwd>\n</environment_context>\n  Hello Codex!  "
	expected := "Hello Codex!"
	got := cleanCodexText(input)
	if got != expected {
		t.Errorf("cleanCodexText() = %q; want %q", got, expected)
	}
}

func TestCodexDetectorRealDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("User home dir not found")
	}

	codexDir := filepath.Join(home, ".codex")
	if _, err := os.Stat(codexDir); os.IsNotExist(err) {
		t.Skip("~/.codex does not exist")
	}

	d := &CodexDetector{}
	// Test ListSessions on current working directory or user home
	cwd, _ := os.Getwd()
	sessions, err := d.ListSessions(cwd, false)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	t.Logf("Found %d codex sessions for cwd %s", len(sessions), cwd)
	for i, s := range sessions {
		t.Logf("Session %d: ID=%s Title=%s CreatedAt=%v UpdatedAt=%v Model=%s", i, s.ID, s.Title, s.CreatedAt, s.UpdatedAt, s.Model)
	}
}

func TestCodexIndexRequiresMatchingCwd(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	indexPath := filepath.Join(root, "session_index.jsonl")
	entries := []map[string]string{
		{
			"id":          "missing-cwd",
			"thread_name": "must not leak",
			"updated_at":  "2026-07-24T00:00:00Z",
		},
		{
			"id":          "matching-cwd",
			"thread_name": "matching session",
			"updated_at":  "2026-07-24T00:00:00Z",
			"cwd":         cwd,
		},
		{
			"id":          "foreign-cwd",
			"thread_name": "foreign session",
			"updated_at":  "2026-07-24T00:00:00Z",
			"cwd":         filepath.Join(root, "other"),
		},
	}

	f, err := os.Create(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		line, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	d := &CodexDetector{dir: root}
	sessions, err := d.ListSessions(cwd, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions; want 1: %#v", len(sessions), sessions)
	}
	if sessions[0].ID != "matching-cwd" {
		t.Fatalf("ListSessions returned session %q; want matching-cwd", sessions[0].ID)
	}
	if sessions[0].WorkDir != cwd {
		t.Fatalf("ListSessions returned WorkDir %q; want %q", sessions[0].WorkDir, cwd)
	}

	allSessions, err := d.ListSessions(cwd, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(allSessions) != 2 {
		t.Fatalf("all-project ListSessions returned %d sessions; want 2: %#v", len(allSessions), allSessions)
	}
}

func TestCodexForkedRolloutKeepsOwnID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-2026-03-26T16-21-10-019d2904-c0b9-7a92-96b8-a238db277d3f.jsonl")
	content := `{"timestamp":"2026-03-26T07:21:15.604Z","type":"session_meta","payload":{"id":"019d2904-c0b9-7a92-96b8-a238db277d3f","forked_from_id":"019d28fd-aa1f-78b0-934e-87fde6a61d82","cwd":"D:/proj"}}
{"timestamp":"2026-03-26T07:21:15.606Z","type":"session_meta","payload":{"id":"019d28fd-aa1f-78b0-934e-87fde6a61d82","cwd":"D:/proj"}}
{"timestamp":"2026-03-26T07:21:16.000Z","type":"event_msg","payload":{"type":"user_message","message":"hello"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	d := &CodexDetector{}
	s, err := d.parseRolloutFile(path, "", true)
	if err != nil || s == nil {
		t.Fatalf("parseRolloutFile failed: %v", err)
	}
	if s.ID != "019d2904-c0b9-7a92-96b8-a238db277d3f" {
		t.Errorf("ID = %q; want forked session's own ID, not parent's", s.ID)
	}
	if len(s.ResumeCmd) != 3 || s.ResumeCmd[2] != s.ID {
		t.Errorf("ResumeCmd = %v; want resume of %s", s.ResumeCmd, s.ID)
	}
}

func TestCodexListsAndSearchesZstdRollouts(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	dayDir := filepath.Join(root, "sessions", "2026", "09", "04")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `{"timestamp":"2026-09-04T02:57:08Z","type":"session_meta","payload":{"id":"01a06a59-3eba-7913-8f3b-c0d56c7ec3a6","cwd":` + strconv.Quote(cwd) + `}}
{"timestamp":"2026-09-04T02:57:10Z","type":"event_msg","payload":{"type":"user_message","message":"ボタン押下アニメーションを追加"}}
`
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enc.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dayDir, "rollout-2026-09-04T11-57-08-01a06a59-3eba-7913-8f3b-c0d56c7ec3a6.jsonl.zst")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &CodexDetector{dir: root}
	sessions, err := d.ListSessions(cwd, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions; want 1", len(sessions))
	}
	s := sessions[0]
	if s.ID != "01a06a59-3eba-7913-8f3b-c0d56c7ec3a6" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.Title != "ボタン押下アニメーションを追加" {
		t.Errorf("Title = %q", s.Title)
	}

	matched, err := d.SearchSession(s, "アニメーション")
	if err != nil || !matched {
		t.Errorf("SearchSession = %v, %v; want true", matched, err)
	}
}

func TestCodexTitleSkipsInjectedAgentsInstructions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-2026-09-04T11-57-08-01a06a59-3eba-7913-8f3b-c0d56c7ec3a6.jsonl")
	writeJSONLines(t, path,
		map[string]any{"type": "session_meta", "payload": map[string]any{"id": "01a06a59-3eba-7913-8f3b-c0d56c7ec3a6", "cwd": "D:/proj"}},
		map[string]any{"type": "response_item", "payload": map[string]any{"type": "message", "role": "user", "content": []any{map[string]any{"type": "input_text", "text": "# AGENTS.md instructions\n\n<INSTRUCTIONS>\n## Git\n</INSTRUCTIONS>"}}}},
		map[string]any{"type": "response_item", "payload": map[string]any{"type": "message", "role": "user", "content": []any{map[string]any{"type": "input_text", "text": "add a button animation"}}}},
	)
	d := &CodexDetector{}
	s, err := d.parseRolloutFile(path, "", true)
	if err != nil || s == nil {
		t.Fatalf("parseRolloutFile failed: %v", err)
	}
	if s.Title != "add a button animation" {
		t.Errorf("Title = %q; want the first prompt typed by the user", s.Title)
	}
}

func TestCodexIndexThreadNameTakesPrecedence(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	dayDir := filepath.Join(root, "sessions", "2026", "09", "04")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	const id = "01a06a59-3eba-7913-8f3b-c0d56c7ec3a6"
	writeJSONLines(t, filepath.Join(dayDir, "rollout-2026-09-04T11-57-08-"+id+".jsonl"),
		map[string]any{"type": "session_meta", "payload": map[string]any{"id": id, "cwd": cwd}},
		map[string]any{"type": "event_msg", "payload": map[string]any{"type": "user_message", "message": "first prompt"}},
	)
	writeJSONLines(t, filepath.Join(root, "session_index.jsonl"),
		map[string]any{"id": id, "thread_name": "renamed thread", "updated_at": "2026-09-04T02:58:27Z"},
	)

	d := &CodexDetector{dir: root}
	sessions, err := d.ListSessions(cwd, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].Title != "renamed thread" {
		t.Fatalf("sessions = %+v; want one session titled from the index", sessions)
	}
}
