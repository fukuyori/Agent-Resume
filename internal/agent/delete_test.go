package agent

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"agres/internal/session"
)

func TestClaudeDeleteRemovesTranscriptAndSubdir(t *testing.T) {
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
	line, _ := json.Marshal(map[string]any{"type": "user", "cwd": cwd, "timestamp": "2026-07-24T00:00:00Z",
		"message": map[string]any{"role": "user", "content": "hi"}})
	file := filepath.Join(projectDir, "abc.jsonl")
	if err := os.WriteFile(file, append(line, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(projectDir, "abc")
	if err := os.MkdirAll(filepath.Join(sub, "tool-results"), 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(projectDir, "keep.jsonl")
	if err := os.WriteFile(keep, append(line, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	sessions, _ := d.ListSessions(cwd, false)
	var target *sessionAlias
	for i := range sessions {
		if sessions[i].ID == "abc" {
			target = &sessionAlias{sessions[i]}
		}
	}
	if target == nil || len(target.Paths) != 2 {
		t.Fatalf("expected transcript + subdir paths, got %+v", target)
	}
	if err := d.Delete(target.Session); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{file, sub} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still exists", p)
		}
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("unrelated session was removed: %v", err)
	}
}

func TestCodexDeleteRemovesRollout(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sessions", "2026", "07", "01")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(root, "proj")
	content := `{"timestamp":"2026-07-01T00:00:00Z","type":"session_meta","payload":{"id":"019d2904-c0b9-7a92-96b8-a238db277d3f","cwd":"` + filepath.ToSlash(cwd) + `"}}
`
	file := filepath.Join(dir, "rollout-2026-07-01T00-00-00-019d2904-c0b9-7a92-96b8-a238db277d3f.jsonl")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	d := &CodexDetector{dir: root}
	sessions, _ := d.ListSessions(cwd, true)
	if len(sessions) != 1 || len(sessions[0].Paths) != 1 {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}
	if err := d.Delete(sessions[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("rollout still exists")
	}
}

func TestOpenCodeDeleteRemovesRowsAndKeepsOthers(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "opencode.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE session (id TEXT PRIMARY KEY, title TEXT NOT NULL, time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL, model TEXT NOT NULL, project_id TEXT NOT NULL, directory TEXT NOT NULL);
		CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL, data TEXT NOT NULL);
		CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL,
			time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL);
		INSERT INTO session VALUES ('a','A',1,2,'','g','/p'), ('b','B',1,3,'','g','/p');
		INSERT INTO message VALUES ('m1','a',1,1,'xxxx'), ('m2','b',1,1,'yy');
		INSERT INTO part VALUES ('p1','m1','a',1,1,'zzzzzz'), ('p2','m2','b',1,1,'w');
	`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	d := &OpenCodeDetector{dir: root}
	sessions, _ := d.ListSessions("/p", true)
	sizes := map[string]int64{}
	for _, s := range sessions {
		sizes[s.ID] = s.Size
	}
	if sizes["a"] != 10 || sizes["b"] != 3 {
		t.Fatalf("unexpected sizes: %v", sizes)
	}
	var target sessionAlias
	for _, s := range sessions {
		if s.ID == "a" {
			target = sessionAlias{s}
		}
	}
	if err := d.Delete(target.Session); err != nil {
		t.Fatal(err)
	}
	after, _ := d.ListSessions("/p", true)
	if len(after) != 1 || after[0].ID != "b" {
		t.Fatalf("after delete: %+v", after)
	}
	db, _ = sql.Open("sqlite", filepath.Join(root, "opencode.db"))
	defer db.Close()
	var n int
	db.QueryRow("SELECT COUNT(*) FROM message WHERE session_id='a'").Scan(&n)
	if n != 0 {
		t.Errorf("message rows for a remain: %d", n)
	}
	db.QueryRow("SELECT COUNT(*) FROM part WHERE session_id='a'").Scan(&n)
	if n != 0 {
		t.Errorf("part rows for a remain: %d", n)
	}
}

type sessionAlias struct{ session.Session }
