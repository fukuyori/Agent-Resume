package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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

func (d *CodexDetector) codexDir() string {
	return filepath.Join(d.homeDir(), ".codex")
}

func (d *CodexDetector) indexFile() string {
	return filepath.Join(d.codexDir(), "session_index.jsonl")
}

func (d *CodexDetector) historyFile() string {
	return filepath.Join(d.codexDir(), "history.jsonl")
}

func (d *CodexDetector) sessionsDirs() []string {
	return []string{
		filepath.Join(d.codexDir(), "sessions"),
		filepath.Join(d.codexDir(), "archived_sessions"),
	}
}

func (d *CodexDetector) Detect(cwd string) bool {
	sessions, err := d.ListSessions(cwd)
	return err == nil && len(sessions) > 0
}

type historyInfo struct {
	title string
	ts    time.Time
}

func (d *CodexDetector) readHistoryMap() map[string]historyInfo {
	res := make(map[string]historyInfo)
	f, err := os.Open(d.historyFile())
	if err != nil {
		return res
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry struct {
			SessionID string `json:"session_id"`
			Ts        int64  `json:"ts"`
			Text      string `json:"text"`
		}
		if err := json.Unmarshal(line, &entry); err != nil || entry.SessionID == "" {
			continue
		}

		info, ok := res[entry.SessionID]
		t := time.Unix(entry.Ts, 0)
		cleanTxt := cleanCodexText(entry.Text)

		if !ok {
			if cleanTxt != "" {
				res[entry.SessionID] = historyInfo{
					title: truncate(cleanTxt, 50),
					ts:    t,
				}
			}
		} else {
			if info.title == "" && cleanTxt != "" {
				info.title = truncate(cleanTxt, 50)
			}
			if t.After(info.ts) {
				info.ts = t
			}
			res[entry.SessionID] = info
		}
	}
	return res
}

func extractSessionIDFromFilename(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, ".jsonl")
	if strings.HasPrefix(name, "rollout-") {
		trimmed := strings.TrimPrefix(name, "rollout-")
		if len(trimmed) > 20 && trimmed[19] == '-' {
			return trimmed[20:]
		}
	}
	return name
}

var envContextRe = regexp.MustCompile(`(?s)<environment_context>.*?</environment_context>`)

func cleanCodexText(s string) string {
	s = envContextRe.ReplaceAllString(s, "")
	return cleanContent(s)
}

func (d *CodexDetector) parseRolloutFile(path string, normCwd string) (*session.Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	sessionID := extractSessionIDFromFilename(path)

	s := &session.Session{
		ID:        sessionID,
		Agent:     session.AgentCodex,
		UpdatedAt: stat.ModTime(),
		CreatedAt: stat.ModTime(),
		ResumeCmd: []string{"codex", "resume", sessionID},
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	matchedCwd := false
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		if lineCount > 200 && matchedCwd && s.Title != "" {
			break
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw struct {
			Timestamp string          `json:"timestamp"`
			Type      string          `json:"type"`
			Payload   json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		if raw.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, raw.Timestamp); err == nil {
				if s.CreatedAt.IsZero() || t.Before(s.CreatedAt) {
					s.CreatedAt = t
				}
				if t.After(s.UpdatedAt) {
					s.UpdatedAt = t
				}
			}
		}

		switch raw.Type {
		case "session_meta":
			var meta struct {
				ID         string `json:"id"`
				Title      string `json:"title"`
				ThreadName string `json:"thread_name"`
				Cwd        string `json:"cwd"`
			}
			if err := json.Unmarshal(raw.Payload, &meta); err == nil {
				if meta.ID != "" {
					s.ID = meta.ID
					s.ResumeCmd = []string{"codex", "resume", meta.ID}
				}
				if meta.Cwd != "" && normalizePath(meta.Cwd) == normCwd {
					matchedCwd = true
				}
				if meta.Title != "" {
					s.Title = meta.Title
				} else if meta.ThreadName != "" {
					s.Title = meta.ThreadName
				}
			}

		case "turn_context":
			var ctx struct {
				Cwd   string `json:"cwd"`
				Model string `json:"model"`
			}
			if err := json.Unmarshal(raw.Payload, &ctx); err == nil {
				if ctx.Cwd != "" && normalizePath(ctx.Cwd) == normCwd {
					matchedCwd = true
				}
				if ctx.Model != "" && s.Model == "" {
					s.Model = ctx.Model
				}
			}

		case "event_msg":
			if s.Title == "" {
				var msg struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}
				if err := json.Unmarshal(raw.Payload, &msg); err == nil {
					if msg.Type == "user_message" && msg.Message != "" {
						if clean := cleanCodexText(msg.Message); clean != "" {
							s.Title = truncate(clean, 50)
						}
					}
				}
			}

		case "response_item":
			if s.Title == "" {
				var item struct {
					Type    string `json:"type"`
					Role    string `json:"role"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				}
				if err := json.Unmarshal(raw.Payload, &item); err == nil {
					if item.Role == "user" {
						for _, c := range item.Content {
							if clean := cleanCodexText(c.Text); clean != "" {
								s.Title = truncate(clean, 50)
								break
							}
						}
					}
				}
			}
		}
	}

	if !matchedCwd {
		return nil, nil
	}

	return s, nil
}

func (d *CodexDetector) ListSessions(cwd string) ([]session.Session, error) {
	normCwd := normalizePath(cwd)
	sessionMap := make(map[string]*session.Session)

	// 1. Read history.jsonl
	historyMap := d.readHistoryMap()

	// 2. Read session_index.jsonl (for backward compatibility)
	if f, err := os.Open(d.indexFile()); err == nil {
		defer f.Close()
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
			if err := json.Unmarshal(line, &entry); err != nil || entry.ID == "" {
				continue
			}

			if entry.Cwd != "" && normalizePath(entry.Cwd) != normCwd {
				continue
			}

			s := &session.Session{
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

			sessionMap[entry.ID] = s
		}
	}

	// 3. Scan sessions/ and archived_sessions/
	for _, sDir := range d.sessionsDirs() {
		if _, err := os.Stat(sDir); os.IsNotExist(err) {
			continue
		}

		_ = filepath.WalkDir(sDir, func(path string, dEntry os.DirEntry, err error) error {
			if err != nil || dEntry.IsDir() || !strings.HasSuffix(dEntry.Name(), ".jsonl") {
				return nil
			}

			s, parseErr := d.parseRolloutFile(path, normCwd)
			if parseErr != nil || s == nil {
				return nil
			}

			if existing, ok := sessionMap[s.ID]; ok {
				if existing.Title == "" || existing.Title == "Untitled session" {
					existing.Title = s.Title
				}
				if existing.Model == "" {
					existing.Model = s.Model
				}
				if s.UpdatedAt.After(existing.UpdatedAt) {
					existing.UpdatedAt = s.UpdatedAt
				}
				if existing.CreatedAt.IsZero() || s.CreatedAt.Before(existing.CreatedAt) {
					existing.CreatedAt = s.CreatedAt
				}
			} else {
				sessionMap[s.ID] = s
			}
			return nil
		})
	}

	// 4. Finalize sessions and fallback title/timestamp from historyMap
	var sessions []session.Session
	for _, s := range sessionMap {
		if hInfo, ok := historyMap[s.ID]; ok {
			if s.Title == "" || s.Title == "Untitled session" {
				s.Title = hInfo.title
			}
			if s.CreatedAt.IsZero() {
				s.CreatedAt = hInfo.ts
			}
			if s.UpdatedAt.IsZero() {
				s.UpdatedAt = hInfo.ts
			}
		}

		if s.Title == "" {
			s.Title = "Untitled session"
		}

		sessions = append(sessions, *s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

