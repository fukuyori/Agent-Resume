package agent

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"agres/internal/session"

	_ "modernc.org/sqlite"
)

type OpenCodeDetector struct {
	dir string
}

func (d *OpenCodeDetector) Name() session.Agent { return session.AgentOpenCode }
func (d *OpenCodeDetector) Icon() string        { return "opencode" }

func (d *OpenCodeDetector) dataDir() string {
	if d.dir != "" {
		return d.dir
	}
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

// Delete removes the session row; message/part rows cascade. The database
// file is compacted afterwards so the space is actually released.
func (d *OpenCodeDetector) Delete(s session.Session) error {
	db, err := sql.Open("sqlite", d.dbPath())
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return err
	}
	for _, table := range []string{"part", "message", "session"} {
		if _, err := db.Exec("DELETE FROM "+table+" WHERE "+sessionColumn(table)+" = ?", s.ID); err != nil {
			return err
		}
	}
	_, err = db.Exec("VACUUM")
	return err
}

func sessionColumn(table string) string {
	if table == "session" {
		return "id"
	}
	return "session_id"
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

func (d *OpenCodeDetector) ListSessions(cwd string, allProjects bool) ([]session.Session, error) {
	dbPath := d.dbPath()
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, title, time_created, time_updated, model, directory
		FROM session
		ORDER BY time_updated DESC
	`)
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
		var directory string

		if err := rows.Scan(&s.ID, &title, &timeCreated, &timeUpdated, &modelJSON, &directory); err != nil {
			continue
		}
		if directory == "" || (!allProjects && !samePath(directory, cwd)) {
			continue
		}

		s.Agent = session.AgentOpenCode
		s.Title = title
		s.WorkDir = directory
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
	rows.Close()

	sizes := d.querySizes(db)
	for i := range sessions {
		sessions[i].Size = sizes[sessions[i].ID]
	}
	return sessions, nil
}

// querySizes returns the total stored bytes (message + part data) per session.
func (d *OpenCodeDetector) querySizes(db *sql.DB) map[string]int64 {
	sizes := make(map[string]int64)
	for _, table := range []string{"message", "part"} {
		rows, err := db.Query(`SELECT session_id, COALESCE(SUM(LENGTH(data)), 0) FROM ` + table + ` GROUP BY session_id`)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id string
			var n int64
			if err := rows.Scan(&id, &n); err == nil {
				sizes[id] += n
			}
		}
		rows.Close()
	}
	return sizes
}

func extractModelName(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}
	var m struct {
		ID         string `json:"id"`
		ModelID    string `json:"modelID"`
		ProviderID string `json:"providerID"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return ""
	}
	if m.ModelID != "" {
		return m.ModelID
	}
	if m.ID != "" {
		return m.ID
	}
	return m.ProviderID
}
