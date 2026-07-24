package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	got, err := d.parseSessionFile(sessionPath, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("foreign Claude session was included: %#v", got)
	}

	writeClaudeTestSession(cwd)
	got, err = d.parseSessionFile(sessionPath, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Title != "test request" {
		t.Fatalf("matching Claude session was not included: %#v", got)
	}
}
