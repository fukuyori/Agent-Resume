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
	normCwd := normalizePath(cwd)
	sessionMap := make(map[string]*session.Session)

	// 1. Read metadata cache if available
	metaPath := filepath.Join(d.baseDir(), "cache", "conversation_metadata.json")
	if mf, err := os.Open(metaPath); err == nil {
		defer mf.Close()
		var meta struct {
			Conversations map[string]struct {
				Summary struct {
					ID            string   `json:"ID"`
					Title         string   `json:"Title"`
					Preview       string   `json:"Preview"`
					UpdatedAt     string   `json:"UpdatedAt"`
					WorkspaceURIs []string `json:"WorkspaceURIs"`
				} `json:"summary"`
				LastModifiedTime string `json:"last_modified_time"`
			} `json:"conversations"`
		}
		if err := json.NewDecoder(mf).Decode(&meta); err == nil {
			for uuid, c := range meta.Conversations {
				matched := false
				for _, uri := range c.Summary.WorkspaceURIs {
					cleanURI := strings.TrimPrefix(uri, "file:///")
					cleanURI = strings.TrimPrefix(cleanURI, "file://")
					if normalizePath(cleanURI) == normCwd {
						matched = true
						break
					}
				}
				if matched {
					s := &session.Session{
						ID:        uuid,
						Agent:     session.AgentAntigravity,
						ResumeCmd: []string{"agy", "--conversation", uuid},
					}
					if c.Summary.Title != "" {
						s.Title = c.Summary.Title
					} else if c.Summary.Preview != "" {
						s.Title = c.Summary.Preview
					}
					if c.Summary.UpdatedAt != "" {
						if t, err := parseAntigravityTime(c.Summary.UpdatedAt); err == nil {
							s.UpdatedAt = t
							s.CreatedAt = t
						}
					}
					if c.LastModifiedTime != "" {
						if t, err := parseAntigravityTime(c.LastModifiedTime); err == nil {
							if s.UpdatedAt.IsZero() || t.After(s.UpdatedAt) {
								s.UpdatedAt = t
							}
							if s.CreatedAt.IsZero() || t.Before(s.CreatedAt) {
								s.CreatedAt = t
							}
						}
					}
					sessionMap[uuid] = s
				}
			}
		}
	}

	// 2. Read last_conversations cache
	cache := d.cacheFile()
	if f, err := os.Open(cache); err == nil {
		defer f.Close()
		var mappings map[string]string
		if err := json.NewDecoder(f).Decode(&mappings); err == nil {
			for wsPath, uuid := range mappings {
				if normalizePath(wsPath) == normCwd {
					if _, exists := sessionMap[uuid]; !exists {
						sessionMap[uuid] = &session.Session{
							ID:        uuid,
							Agent:     session.AgentAntigravity,
							ResumeCmd: []string{"agy", "--conversation", uuid},
						}
					}
				}
			}
		}
	}

	// 3. Parse transcripts and populate/override session details
	var sessions []session.Session
	for uuid, baseSession := range sessionMap {
		transcript := filepath.Join(d.brainDir(), uuid, ".system_generated", "logs", "transcript.jsonl")
		parsed, err := d.parseTranscript(transcript, uuid, baseSession)
		if err != nil && baseSession != nil {
			parsed = baseSession
		}
		if parsed != nil {
			sessions = append(sessions, *parsed)
		}
	}

	return sessions, nil
}

func (d *AntigravityDetector) parseTranscript(path, uuid string, base *session.Session) (*session.Session, error) {
	s := &session.Session{
		ID:        uuid,
		Agent:     session.AgentAntigravity,
		ResumeCmd: []string{"agy", "--conversation", uuid},
	}
	if base != nil {
		s.Title = base.Title
		s.CreatedAt = base.CreatedAt
		s.UpdatedAt = base.UpdatedAt
	}

	var modTime time.Time
	dbPath := filepath.Join(d.baseDir(), "conversations", uuid+".db")
	if info, err := os.Stat(dbPath); err == nil {
		modTime = info.ModTime()
	}
	if info, err := os.Stat(path); err == nil {
		if modTime.IsZero() || info.ModTime().After(modTime) {
			modTime = info.ModTime()
		}
	} else {
		// Fallback to session dir stat if transcript file stat fails
		sessionDir := filepath.Join(d.brainDir(), uuid)
		if dirInfo, err := os.Stat(sessionDir); err == nil {
			if modTime.IsZero() || dirInfo.ModTime().After(modTime) {
				modTime = dirInfo.ModTime()
			}
		}
	}

	f, err := os.Open(path)
	if err == nil {
		defer f.Close()

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
				Timestamp string `json:"timestamp"`
			}
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}

			tsStr := entry.CreatedAt
			if tsStr == "" {
				tsStr = entry.Timestamp
			}

			if tsStr != "" {
				if t, err := parseAntigravityTime(tsStr); err == nil {
					if s.CreatedAt.IsZero() || t.Before(s.CreatedAt) {
						s.CreatedAt = t
					}
					if s.UpdatedAt.IsZero() || t.After(s.UpdatedAt) {
						s.UpdatedAt = t
					}
				}
			}

			if (s.Title == "" || s.Title == "Untitled session") && entry.Source == "USER_EXPLICIT" && entry.Type == "USER_INPUT" {
				if req := extractUserRequest(entry.Content); req != "" {
					s.Title = req
				}
			}
		}
	}

	if s.UpdatedAt.IsZero() {
		if !modTime.IsZero() {
			s.UpdatedAt = modTime
		}
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = s.UpdatedAt
	}

	if s.Title == "" {
		s.Title = "Untitled session"
	}

	return s, nil
}

func parseAntigravityTime(ts string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, ts)
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
