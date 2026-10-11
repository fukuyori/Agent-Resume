package agent

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agres/internal/session"
)

const maxSearchJSONLineSize = 64 * 1024 * 1024

func containsFold(text, keyword string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(keyword))
}

func searchableContentMatches(raw json.RawMessage, keyword string) bool {
	return searchableContentMatchesWith(raw, keyword, func(text string) string { return text })
}

func searchableContentMatchesWith(raw json.RawMessage, keyword string, normalize func(string) string) bool {
	if len(raw) == 0 {
		return false
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return containsFold(normalize(text), keyword)
	}

	var blocks []struct {
		Type    string          `json:"type"`
		Text    string          `json:"text"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return false
	}
	for _, block := range blocks {
		switch strings.ToLower(block.Type) {
		case "", "text", "input_text", "output_text":
			if containsFold(normalize(block.Text), keyword) || searchableContentMatchesWith(block.Content, keyword, normalize) {
				return true
			}
		}
	}
	return false
}

func scanJSONLines(path string, match func([]byte) bool) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maxSearchJSONLineSize)
	for scanner.Scan() {
		if match(scanner.Bytes()) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	return false, nil
}

func (d *ClaudeDetector) SearchSession(s session.Session, keyword string) (bool, error) {
	for _, path := range s.Paths {
		if filepath.Ext(path) != ".jsonl" {
			continue
		}
		return scanJSONLines(path, func(line []byte) bool {
			var entry struct {
				Type    string          `json:"type"`
				Content json.RawMessage `json:"content"`
				Message struct {
					Role    string          `json:"role"`
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &entry) != nil {
				return false
			}
			if entry.Type == "queue-operation" {
				return searchableContentMatches(entry.Content, keyword)
			}
			role := strings.ToLower(entry.Message.Role)
			if role == "" {
				role = strings.ToLower(entry.Type)
			}
			return (role == "user" || role == "assistant") && searchableContentMatches(entry.Message.Content, keyword)
		})
	}
	return false, nil
}

func (d *CodexDetector) SearchSession(s session.Session, keyword string) (bool, error) {
	foundRollout := false
	for _, path := range s.Paths {
		if filepath.Ext(path) != ".jsonl" {
			continue
		}
		foundRollout = true
		matched, err := scanJSONLines(path, func(line []byte) bool {
			var entry struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}
			if json.Unmarshal(line, &entry) != nil {
				return false
			}
			switch entry.Type {
			case "event_msg":
				var event struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}
				if json.Unmarshal(entry.Payload, &event) != nil {
					return false
				}
				return (event.Type == "user_message" || event.Type == "agent_message") && containsFold(cleanCodexText(event.Message), keyword)
			case "response_item":
				var item struct {
					Type    string          `json:"type"`
					Role    string          `json:"role"`
					Content json.RawMessage `json:"content"`
				}
				if json.Unmarshal(entry.Payload, &item) != nil || item.Type != "message" {
					return false
				}
				role := strings.ToLower(item.Role)
				return (role == "user" || role == "assistant") && searchableContentMatchesWith(item.Content, keyword, cleanCodexText)
			}
			return false
		})
		if err != nil || matched {
			return matched, err
		}
	}
	if foundRollout {
		return false, nil
	}

	matched, err := scanJSONLines(d.historyFile(), func(line []byte) bool {
		var entry struct {
			SessionID string `json:"session_id"`
			Text      string `json:"text"`
		}
		return json.Unmarshal(line, &entry) == nil && entry.SessionID == s.ID && containsFold(cleanCodexText(entry.Text), keyword)
	})
	if os.IsNotExist(err) {
		return false, nil
	}
	return matched, err
}

func (d *OpenCodeDetector) SearchSession(s session.Session, keyword string) (bool, error) {
	db, err := sql.Open("sqlite", d.dbPath()+"?mode=ro")
	if err != nil {
		return false, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT m.data, COALESCE(p.data, '')
		FROM message AS m
		LEFT JOIN part AS p ON p.message_id = m.id
		WHERE m.session_id = ?
	`, s.ID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var messageJSON, partJSON string
		if err := rows.Scan(&messageJSON, &partJSON); err != nil {
			return false, err
		}
		var message struct {
			Role    string          `json:"role"`
			Text    string          `json:"text"`
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal([]byte(messageJSON), &message) != nil {
			continue
		}
		role := strings.ToLower(message.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		if containsFold(message.Text, keyword) || searchableContentMatches(message.Content, keyword) {
			return true, nil
		}

		var part struct {
			Type    string          `json:"type"`
			Text    string          `json:"text"`
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal([]byte(partJSON), &part) == nil && strings.ToLower(part.Type) == "text" {
			if containsFold(part.Text, keyword) || searchableContentMatches(part.Content, keyword) {
				return true, nil
			}
		}
	}
	return false, rows.Err()
}

func (d *AiderDetector) SearchSession(s session.Session, keyword string) (bool, error) {
	return containsFold(s.Summary, keyword), nil
}

func (d *AntigravityDetector) SearchSession(s session.Session, keyword string) (bool, error) {
	path := filepath.Join(d.brainDir(), s.ID, ".system_generated", "logs", "transcript.jsonl")
	matched, err := scanJSONLines(path, func(line []byte) bool {
		var entry struct {
			Source  string `json:"source"`
			Type    string `json:"type"`
			Content string `json:"content"`
		}
		if json.Unmarshal(line, &entry) != nil {
			return false
		}
		switch {
		case entry.Source == "USER_EXPLICIT" && entry.Type == "USER_INPUT":
			if request := extractUserRequestForSearch(entry.Content); request != "" {
				return containsFold(request, keyword)
			}
			return containsFold(entry.Content, keyword)
		case entry.Source == "MODEL" && entry.Type == "PLANNER_RESPONSE":
			return containsFold(entry.Content, keyword)
		}
		return false
	})
	if os.IsNotExist(err) {
		return false, nil
	}
	return matched, err
}

func extractUserRequestForSearch(content string) string {
	match := userRequestRe.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}
