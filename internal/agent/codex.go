package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"agres/internal/session"
)

type CodexDetector struct{}

func (d *CodexDetector) Name() session.Agent { return session.AgentCodex }
func (d *CodexDetector) Icon() string         { return "codex" }

func (d *CodexDetector) homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return ""
}

func (d *CodexDetector) indexFile() string {
	return filepath.Join(d.homeDir(), ".codex", "session_index.jsonl")
}

func (d *CodexDetector) Detect(cwd string) bool {
	f, err := os.Open(d.indexFile())
	if err != nil {
		return false
	}
	defer f.Close()

	normCwd := normalizePath(cwd)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry struct {
			Cwd string `json:"cwd"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Cwd == "" {
			continue
		}
		if normalizePath(entry.Cwd) == normCwd {
			return true
		}
	}
	return false
}

func (d *CodexDetector) ListSessions(cwd string) ([]session.Session, error) {
	f, err := os.Open(d.indexFile())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	normCwd := normalizePath(cwd)
	var sessions []session.Session

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry struct {
			ID         string `json:"id"`
			ThreadName string `json:"thread_name"`
			UpdatedAt  string `json:"updated_at"`
			Cwd        string `json:"cwd"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Cwd != "" {
			if normalizePath(entry.Cwd) != normCwd {
				continue
			}
		}

		s := session.Session{
			ID:        entry.ID,
			Agent:     session.AgentCodex,
			Title:     entry.ThreadName,
			ResumeCmd: []string{"codex", "resume", entry.ID},
		}

		if entry.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, entry.UpdatedAt); err == nil {
				s.CreatedAt = t
				s.UpdatedAt = t
			}
		}

		if s.Title == "" {
			s.Title = "Untitled session"
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}
