package session

import "time"

type Agent string

const (
	AgentClaude      Agent = "claude"
	AgentOpenCode    Agent = "opencode"
	AgentAider       Agent = "aider"
	AgentCodex       Agent = "codex"
	AgentAntigravity Agent = "agy"
)

type Session struct {
	ID        string
	Agent     Agent
	Title     string
	Summary   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Model     string
	ResumeCmd []string
}

type Detector interface {
	Name() Agent
	Icon() string
	Detect(cwd string) bool
	ListSessions(cwd string) ([]Session, error)
}
