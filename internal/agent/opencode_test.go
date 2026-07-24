package agent

import (
	"database/sql"
	"path/filepath"
	"testing"
)

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
	sessions, err := d.ListSessions(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("ListSessions returned %d sessions; want 1: %#v", len(sessions), sessions)
	}
	if sessions[0].ID != "matching" {
		t.Fatalf("ListSessions returned session %q; want matching", sessions[0].ID)
	}
}
