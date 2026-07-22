package agent

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agres/internal/session"
)

type AiderDetector struct{}

func (d *AiderDetector) Name() session.Agent { return session.AgentAider }
func (d *AiderDetector) Icon() string         { return "\u2693\ufe0f" }

func (d *AiderDetector) historyFile(cwd string) string {
	abs, _ := filepath.Abs(cwd)
	return filepath.Join(abs, ".aider.chat.history.md")
}

func (d *AiderDetector) Detect(cwd string) bool {
	info, err := os.Stat(d.historyFile(cwd))
	return err == nil && !info.IsDir()
}

func (d *AiderDetector) ListSessions(cwd string) ([]session.Session, error) {
	path := d.historyFile(cwd)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, _ := f.Stat()

	var sessions []session.Session
	var current *session.Session
	var content strings.Builder

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "# ") {
			if current != nil {
				current.Summary = truncate(strings.TrimSpace(content.String()), 80)
				sessions = append(sessions, *current)
				content.Reset()
			}
			current = &session.Session{
				ID:        filepath.Base(path),
				Agent:     session.AgentAider,
				Title:     strings.TrimPrefix(line, "# "),
				ResumeCmd: []string{"aider", "--resume"},
				UpdatedAt: stat.ModTime(),
			}
		} else if current != nil {
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	if current != nil {
		current.Summary = truncate(strings.TrimSpace(content.String()), 80)
		sessions = append(sessions, *current)
	}

	if len(sessions) == 0 {
		sessions = append(sessions, session.Session{
			ID:        filepath.Base(path),
			Agent:     session.AgentAider,
			Title:     "Chat history",
			ResumeCmd: []string{"aider", "--resume"},
			UpdatedAt: stat.ModTime(),
		})
	}

	for i := range sessions {
		sessions[i].CreatedAt = stat.ModTime()
	}

	return sessions, nil
}

func parseTime(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
