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

type AntigravityDetector struct{}

func (d *AntigravityDetector) Name() session.Agent { return session.AgentAntigravity }
func (d *AntigravityDetector) Icon() string         { return "agy" }

func (d *AntigravityDetector) baseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gemini", "antigravity-cli")
}

func (d *AntigravityDetector) brainDir() string {
	return filepath.Join(d.baseDir(), "brain")
}

func (d *AntigravityDetector) cacheFile() string {
	return filepath.Join(d.baseDir(), "cache", "last_conversations.json")
}

func (d *AntigravityDetector) Detect(cwd string) bool {
	cache := d.cacheFile()
	f, err := os.Open(cache)
	if err != nil {
		return false
	}
	defer f.Close()

	var mappings map[string]string
	if err := json.NewDecoder(f).Decode(&mappings); err != nil {
		return false
	}

	normCwd := normalizePath(cwd)
	for wsPath := range mappings {
		if normalizePath(wsPath) == normCwd {
			return true
		}
	}
	return false
}

func (d *AntigravityDetector) ListSessions(cwd string) ([]session.Session, error) {
	cache := d.cacheFile()
	f, err := os.Open(cache)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var mappings map[string]string
	if err := json.NewDecoder(f).Decode(&mappings); err != nil {
		return nil, err
	}

	normCwd := normalizePath(cwd)
	var sessions []session.Session

	for wsPath, uuid := range mappings {
		if normalizePath(wsPath) != normCwd {
			continue
		}

		transcript := filepath.Join(d.brainDir(), uuid, ".system_generated", "logs", "transcript.jsonl")
		s, err := d.parseTranscript(transcript, uuid)
		if err != nil {
			continue
		}
		sessions = append(sessions, *s)
	}

	return sessions, nil
}

func (d *AntigravityDetector) parseTranscript(path, uuid string) (*session.Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := &session.Session{
		ID:        uuid,
		Agent:     session.AgentAntigravity,
		ResumeCmd: []string{"agy", "--conversation", uuid},
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry struct {
			Source    string `json:"source"`
			Type      string `json:"type"`
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, entry.CreatedAt); err == nil {
				if s.CreatedAt.IsZero() || t.Before(s.CreatedAt) {
					s.CreatedAt = t
				}
				if t.After(s.UpdatedAt) {
					s.UpdatedAt = t
				}
			}
		}

		if s.Title == "" && entry.Source == "USER_EXPLICIT" && entry.Type == "USER_INPUT" {
			s.Title = extractUserRequest(entry.Content)
		}
	}

	if s.Title == "" {
		s.Title = "Untitled session"
	}

	return s, nil
}

var userRequestRe = regexp.MustCompile(`(?s)<USER_REQUEST>\s*(.*?)\s*</USER_REQUEST>`)

func extractUserRequest(content string) string {
	m := userRequestRe.FindStringSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	s := strings.TrimSpace(m[1])
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 50 {
		s = s[:47] + "..."
	}
	return s
}
