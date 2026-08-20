package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"
)

func TestClaudeSessionRequiresMatchingCwd(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	foreign := filepath.Join(root, "other")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(foreign, 0o755); err != nil {
		t.Fatal(err)
	}

	sessionPath := filepath.Join(root, "session.jsonl")
	writeClaudeTestSession := func(recordedCwd string) {
		t.Helper()
		entry := map[string]any{
			"type":      "user",
			"cwd":       recordedCwd,
			"timestamp": "2026-07-24T00:00:00Z",
			"message": map[string]any{
				"role":    "user",
				"content": "test request",
			},
		}
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sessionPath, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	d := &ClaudeDetector{}
	writeClaudeTestSession(foreign)
	got, err := d.parseSessionFile(sessionPath, cwd, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("foreign Claude session was included: %#v", got)
	}

	writeClaudeTestSession(cwd)
	got, err = d.parseSessionFile(sessionPath, cwd, false)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Title != "test request" {
		t.Fatalf("matching Claude session was not included: %#v", got)
	}
	if got.WorkDir != cwd {
		t.Fatalf("matching Claude session WorkDir = %q; want %q", got.WorkDir, cwd)
	}

	writeClaudeTestSession(foreign)
	got, err = d.parseSessionFile(sessionPath, cwd, true)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.WorkDir != foreign {
		t.Fatalf("all-project Claude session was not included: %#v", got)
	}
}

func TestTruncatePreservesUTF8(t *testing.T) {
	input := "日本語の長いタイトルを安全に省略する"
	got := truncate(input, 10)
	if !utf8.ValidString(got) {
		t.Fatalf("truncate returned invalid UTF-8: %q", got)
	}
	if runeCount := utf8.RuneCountInString(got); runeCount != 10 {
		t.Fatalf("truncate returned %d runes; want 10: %q", runeCount, got)
	}
}

func TestClaudeAllProjectsScansProjectDirectories(t *testing.T) {
	projectsDir := t.TempDir()
	cwd := filepath.Join(t.TempDir(), "project")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	d := &ClaudeDetector{dir: projectsDir}
	projectDir := filepath.Join(projectsDir, d.projectSlug(cwd))
	if err := os.Mkdir(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{
		"type":      "user",
		"cwd":       cwd,
		"timestamp": "2026-07-24T00:00:00Z",
		"message": map[string]any{
			"role":    "user",
			"content": "test request",
		},
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "session.jsonl"), append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	sessions, err := d.ListSessions(filepath.Join(t.TempDir(), "unrelated"), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].WorkDir != cwd {
		t.Fatalf("all-project Claude sessions = %#v; want session from %q", sessions, cwd)
	}
}
