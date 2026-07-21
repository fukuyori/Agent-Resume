package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"agent-hub/internal/session"
)

type ClaudeDetector struct{}

func (d *ClaudeDetector) Name() session.Agent { return session.AgentClaude }
func (d *ClaudeDetector) Icon() string         { return "claude" }

func (d *ClaudeDetector) homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return ""
}

func (d *ClaudeDetector) projectSlug(cwd string) string {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}
	slug := strings.ReplaceAll(abs, "/", "-")
	return slug
}

func (d *ClaudeDetector) projectsDir() string {
	return filepath.Join(d.homeDir(), ".claude", "projects")
}

func (d *ClaudeDetector) Detect(cwd string) bool {
	slug := d.projectSlug(cwd)
	if slug == "" {
		return false
	}
	dir := filepath.Join(d.projectsDir(), slug)
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func (d *ClaudeDetector) ListSessions(cwd string) ([]session.Session, error) {
	slug := d.projectSlug(cwd)
	dir := filepath.Join(d.projectsDir(), slug)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var sessions []session.Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		s, err := d.parseSessionFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		sessions = append(sessions, *s)
	}
	return sessions, nil
}

func (d *ClaudeDetector) parseSessionFile(path string) (*session.Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, _ := f.Stat()
	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")

	s := &session.Session{
		ID:        sessionID,
		Agent:     session.AgentClaude,
		UpdatedAt: stat.ModTime(),
		ResumeCmd: []string{"claude", "--resume", sessionID},
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg struct {
			Type      string `json:"type"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp"`
			Message   struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			Model string `json:"model"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, msg.Timestamp); err == nil {
				if s.CreatedAt.IsZero() || t.Before(s.CreatedAt) {
					s.CreatedAt = t
				}
				if t.After(s.UpdatedAt) {
					s.UpdatedAt = t
				}
			}
		}

		if s.Title == "" {
			switch msg.Type {
			case "queue-operation":
				if content := truncate(cleanContent(msg.Content), 50); content != "" {
					s.Title = content
				}
			case "user":
				if content := truncate(cleanContent(msg.Message.Content), 50); content != "" {
					s.Title = content
				}
			}
		}

		if msg.Model != "" && s.Model == "" {
			s.Model = msg.Model
		}
	}

	if s.Title == "" {
		s.Title = "Untitled session"
	}

	return s, nil
}

var jsonlCleaner = regexp.MustCompile(`\{"type":"[^"]*"[^}]*\}`)

func cleanContent(s string) string {
	s = strings.TrimSpace(s)
	s = jsonlCleaner.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
