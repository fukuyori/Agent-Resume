package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	sessions, err := d.ListSessions(cwd)
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
	sessions, err := d.ListSessions(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions; want 1: %#v", len(sessions), sessions)
	}
	if sessions[0].ID != "matching-cwd" {
		t.Fatalf("ListSessions returned session %q; want matching-cwd", sessions[0].ID)
	}
}
