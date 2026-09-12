package agent

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestExtractModelNamePrefersModelID(t *testing.T) {
	tests := []struct {
		name, value, want string
	}{
		{"modelID", `{"modelID":"claude-sonnet","providerID":"anthropic"}`, "claude-sonnet"},
		{"legacy id", `{"id":"gpt-5","providerID":"openai"}`, "gpt-5"},
		{"provider fallback", `{"providerID":"openai"}`, "openai"},
		{"invalid", `{`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractModelName(tt.value); got != tt.want {
				t.Fatalf("extractModelName(%q) = %q; want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestOpenCodeSessionsUseSessionDirectory(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "opencode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE session (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			model TEXT NOT NULL,
			project_id TEXT NOT NULL,
			directory TEXT NOT NULL
		)
	`); err != nil {
		t.Fatal(err)
	}

	cwd := filepath.Join(root, "project")
	foreign := filepath.Join(root, "other")
	if _, err := db.Exec(`
		INSERT INTO session
			(id, title, time_created, time_updated, model, project_id, directory)
		VALUES
			('matching', 'matching session', 1000, 2000, '', 'global', ?),
			('foreign', 'foreign session', 1000, 3000, '', 'global', ?)
	`, cwd, foreign); err != nil {
		t.Fatal(err)
	}

	d := &OpenCodeDetector{dir: root}
	sessions, err := d.ListSessions(cwd, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions; want 1: %#v", len(sessions), sessions)
	}
	if sessions[0].ID != "matching" {
		t.Fatalf("ListSessions returned session %q; want matching", sessions[0].ID)
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
