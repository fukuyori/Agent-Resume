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
	WorkDir   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Model     string
	Size      int64 // stored history size in bytes (0 if unknown)
	ResumeCmd []string
	// Paths lists the files/directories that hold this session's history.
	// Used by Cleaner implementations that delete on the filesystem.
	Paths []string
}

type Detector interface {
	Name() Agent
	Icon() string
	Detect(cwd string) bool
	ListSessions(cwd string, allProjects bool) ([]Session, error)
}

// Cleaner is implemented by detectors whose sessions can be deleted
// individually.
type Cleaner interface {
	Detector
	Delete(s Session) error
}
