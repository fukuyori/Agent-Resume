package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"agres/internal/session"
)

type ClaudeDetector struct{}

func (d *ClaudeDetector) Name() session.Agent { return session.AgentClaude }
func (d *ClaudeDetector) Icon() string        { return "claude" }

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
	slashed := filepath.ToSlash(abs)
	slug := strings.ReplaceAll(slashed, "/", "-")
	slug = strings.ReplaceAll(slug, ":", "-")
	return slug
}

func (d *ClaudeDetector) projectsDir() string {
	return filepath.Join(d.homeDir(), ".claude", "projects")
}

func (d *ClaudeDetector) findProjectDir(cwd string) string {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}
	slashed := filepath.ToSlash(abs)

	// Candidate slugs
	candidateSlugs := []string{
		strings.ReplaceAll(strings.ReplaceAll(slashed, "/", "-"), ":", "-"),
		strings.ReplaceAll(strings.ReplaceAll(abs, "\\", "-"), ":", "-"),
		strings.ReplaceAll(slashed, "/", "-"),
	}

	pDir := d.projectsDir()
	for _, c := range candidateSlugs {
		dir := filepath.Join(pDir, c)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	// Case-insensitive directory matching fallback
	entries, err := os.ReadDir(pDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			for _, c := range candidateSlugs {
				if strings.EqualFold(e.Name(), c) {
					return filepath.Join(pDir, e.Name())
				}
			}
		}
	}

	return ""
}

func (d *ClaudeDetector) Detect(cwd string) bool {
	dir := d.findProjectDir(cwd)
	return dir != ""
}

func (d *ClaudeDetector) ListSessions(cwd string) ([]session.Session, error) {
	dir := d.findProjectDir(cwd)
	if dir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var sessions []session.Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		s, err := d.parseSessionFile(filepath.Join(dir, e.Name()), cwd)
		if err != nil || s == nil {
			continue
		}
		sessions = append(sessions, *s)
	}
	return sessions, nil
}

func (d *ClaudeDetector) parseSessionFile(path, cwd string) (*session.Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, _ := f.Stat()
	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")

	var modTime time.Time
	if stat != nil {
		modTime = stat.ModTime()
	}

	s := &session.Session{
		ID:        sessionID,
		Agent:     session.AgentClaude,
		UpdatedAt: modTime,
		ResumeCmd: []string{"claude", "--resume", sessionID},
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	matchedCwd := false

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg struct {
			Type      string          `json:"type"`
			Content   json.RawMessage `json:"content"`
			Timestamp string          `json:"timestamp"`
			Message   struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
				Model   string          `json:"model"`
			} `json:"message"`
			Model string `json:"model"`
			Cwd   string `json:"cwd"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Timestamp != "" {
			if t, err := parseTimestamp(msg.Timestamp); err == nil {
				if s.CreatedAt.IsZero() || t.Before(s.CreatedAt) {
					s.CreatedAt = t
				}
				if t.After(s.UpdatedAt) {
					s.UpdatedAt = t
				}
			}
		}

		if samePath(msg.Cwd, cwd) {
			matchedCwd = true
		}

		if s.Title == "" {
			switch msg.Type {
			case "queue-operation":
				c := extractRawContent(msg.Content)
				if content := truncate(cleanContent(c), 50); content != "" {
					s.Title = content
				}
			case "user":
				c := extractRawContent(msg.Message.Content)
				if content := truncate(cleanContent(c), 50); content != "" {
					s.Title = content
				}
			}
		}

		if s.Model == "" {
			if msg.Model != "" {
				s.Model = msg.Model
			} else if msg.Message.Model != "" {
				s.Model = msg.Message.Model
			}
		}
	}

	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = modTime
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = s.UpdatedAt
	}

	if s.Title == "" {
		s.Title = "Untitled session"
	}

	if !matchedCwd {
		return nil, nil
	}

	return s, nil
}

func parseTimestamp(ts string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, ts)
}

func extractRawContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var strVal string
	if err := json.Unmarshal(raw, &strVal); err == nil {
		return strVal
	}

	var arrVal []json.RawMessage
	if err := json.Unmarshal(raw, &arrVal); err == nil {
		var parts []string
		for _, item := range arrVal {
			var block struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(item, &block); err == nil {
				if block.Type == "text" && block.Text != "" {
					parts = append(parts, block.Text)
				} else if block.Text != "" {
					parts = append(parts, block.Text)
				} else if block.Content != "" {
					parts = append(parts, block.Content)
				}
			}
		}
		return strings.Join(parts, " ")
	}

	return ""
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
