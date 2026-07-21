package agent

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"agent-hub/internal/session"

	_ "modernc.org/sqlite"
)

type OpenCodeDetector struct{}

func (d *OpenCodeDetector) Name() session.Agent { return session.AgentOpenCode }
func (d *OpenCodeDetector) Icon() string         { return "opencode" }

func (d *OpenCodeDetector) dataDir() string {
	if v := os.Getenv("OPENCODE_DATA_DIR"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode")
}

func (d *OpenCodeDetector) dbPath() string {
	return filepath.Join(d.dataDir(), "opencode.db")
}

func (d *OpenCodeDetector) Detect(cwd string) bool {
	db, err := sql.Open("sqlite", d.dbPath()+"?mode=ro")
	if err != nil {
		return false
	}
	defer db.Close()

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM session`).Scan(&count)
	return err == nil && count > 0
}

func (d *OpenCodeDetector) findProjectID(db *sql.DB, cwd string) string {
	absCwd, _ := filepath.Abs(cwd)

	var projectID string
	err := db.QueryRow(`SELECT id FROM project WHERE worktree = ?`, absCwd).Scan(&projectID)
	if err == nil {
		return projectID
	}

	err = db.QueryRow(`SELECT id FROM project WHERE worktree = '/'`).Scan(&projectID)
	if err == nil {
		return projectID
	}

	return ""
}

func (d *OpenCodeDetector) ListSessions(cwd string) ([]session.Session, error) {
	dbPath := d.dbPath()
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	projectID := d.findProjectID(db, cwd)
	if projectID == "" {
		return nil, nil
	}

	rows, err := db.Query(`
		SELECT id, title, time_created, time_updated, model
		FROM session
		WHERE project_id = ?
		ORDER BY time_updated DESC
		LIMIT 50
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []session.Session
	for rows.Next() {
		var s session.Session
		var title string
		var timeCreated, timeUpdated int64
		var modelJSON string

		if err := rows.Scan(&s.ID, &title, &timeCreated, &timeUpdated, &modelJSON); err != nil {
			continue
		}

		s.Agent = session.AgentOpenCode
		s.Title = title
		s.Model = extractModelName(modelJSON)
		s.ResumeCmd = []string{"opencode", "--session", s.ID}

		if timeCreated > 0 {
			s.CreatedAt = time.UnixMilli(timeCreated)
		}
		if timeUpdated > 0 {
			s.UpdatedAt = time.UnixMilli(timeUpdated)
		}

		if s.Title == "" {
			s.Title = "Untitled session"
		}

		sessions = append(sessions, s)
	}
	return sessions, nil
}

func extractModelName(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}
	var m struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return ""
	}
	if m.ProviderID != "" {
		return m.ProviderID
	}
	return m.ID
}
