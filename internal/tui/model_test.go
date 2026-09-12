package tui

import (
	"strings"
	"testing"

	"agres/internal/session"

	"github.com/charmbracelet/x/ansi"
)

func TestAllProjectsViewShowsWorkingDirectory(t *testing.T) {
	sessions := []session.Session{{
		Agent:   session.AgentCodex,
		Title:   "test session",
		WorkDir: "/projects/example",
	}}

	view := NewModel(sessions, "0.4.0", true).View()
	if !strings.Contains(view, "[all projects]") {
		t.Fatalf("all-project view does not show its mode: %q", view)
	}
	if !strings.Contains(view, sessions[0].WorkDir) {
		t.Fatalf("all-project view does not show working directory: %q", view)
	}
}

func TestSelectedRowUsesFullLineHighlightWithoutCursorMarker(t *testing.T) {
	sessions := []session.Session{{
		Agent:   session.AgentCodex,
		Title:   "selected session",
		WorkDir: "/projects/example",
	}}
	m := NewModel(sessions, "0.4.0", true)
	m.width = 60

	view := m.View()
	if strings.Contains(view, " > ") {
		t.Fatalf("selected row still contains cursor marker: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "[codex]") {
			if width := ansi.StringWidth(line); width != m.width-1 {
				t.Fatalf("selected row width = %d; want %d: %q", width, m.width-1, line)
			}
			return
		}
	}
	t.Fatal("selected row was not rendered")
}

func TestAllProjectsViewHasStableHeightWhenCursorMoves(t *testing.T) {
	sessions := []session.Session{
		{Agent: session.AgentCodex, Title: "first", WorkDir: "/projects/first"},
		{Agent: session.AgentClaude, Title: "second", WorkDir: "/projects/second"},
	}
	m := NewModel(sessions, "0.4.0", true)
	m.width = 80

	firstView := m.View()
	m.cursor = 1
	secondView := m.View()
	if strings.Count(firstView, "\n") != strings.Count(secondView, "\n") {
		t.Fatalf("view height changed when cursor moved:\nfirst=%q\nsecond=%q", firstView, secondView)
	}
}

func TestProjectColumnIsAlignedAcrossAgents(t *testing.T) {
	sessions := []session.Session{
		{Agent: session.AgentCodex, Title: "first", WorkDir: "/projects/project-one"},
		{Agent: session.AgentClaude, Title: "second", WorkDir: "/projects/project-two"},
		{Agent: session.AgentOpenCode, Title: "third", WorkDir: "/projects/project-three"},
	}
	m := NewModel(sessions, "0.4.0", true)
	m.width = 120

	plainView := ansi.Strip(m.View())
	wantColumn := -1
	for i, project := range []string{"[project-one]", "[project-two]", "[project-three]"} {
		column := -1
		for _, line := range strings.Split(plainView, "\n") {
			if strings.Contains(line, project) {
				column = strings.Index(line, project)
				break
			}
		}
		if column < 0 {
			t.Fatalf("project %s was not rendered: %q", project, plainView)
		}
		if i == 0 {
			wantColumn = column
		} else if column != wantColumn {
			t.Fatalf("project %s starts at column %d; want %d", project, column, wantColumn)
		}
	}
}

func TestModelColumnFollowsAgentAndAlignsTitles(t *testing.T) {
	sessions := []session.Session{
		{Agent: session.AgentCodex, Model: "gpt-5", Title: "first title"},
		{Agent: session.AgentClaude, Model: "claude-sonnet", Title: "second title"},
		{Agent: session.AgentAider, Title: "third title"},
	}
	m := NewModel(sessions, "0.4.0", false)
	m.width = 120

	plainView := ansi.Strip(m.View())
	wantTitleColumn := -1
	for i, tt := range []struct {
		agent, model, title string
	}{
		{"[codex]", "[gpt-5]", "first title"},
		{"[claude]", "[claude-sonnet]", "second title"},
		{"[aider]", "", "third title"},
	} {
		var row string
		for _, line := range strings.Split(plainView, "\n") {
			if strings.Contains(line, tt.title) {
				row = line
				break
			}
		}
		if row == "" {
			t.Fatalf("title %q was not rendered: %q", tt.title, plainView)
		}
		agentColumn := strings.Index(row, tt.agent)
		titleColumn := strings.Index(row, tt.title)
		if tt.model != "" {
			modelColumn := strings.Index(row, tt.model)
			if modelColumn <= agentColumn || titleColumn <= modelColumn {
				t.Fatalf("row order is not agent, model, title: %q", row)
			}
		}
		if i == 0 {
			wantTitleColumn = titleColumn
		} else if titleColumn != wantTitleColumn {
			t.Fatalf("title %q starts at column %d; want %d: %q", tt.title, titleColumn, wantTitleColumn, row)
		}
	}
}

func TestFormatModelTruncatesInsideBrackets(t *testing.T) {
	got := formatModel("a-very-long-model-name", 12)
	if got[0] != '[' || got[len(got)-1] != ']' {
		t.Fatalf("formatModel() = %q; want brackets", got)
	}
	if width := ansi.StringWidth(got); width != 12 {
		t.Fatalf("formatModel() width = %d; want 12: %q", width, got)
	}
}

func TestViewLinesFitTerminalWidth(t *testing.T) {
	sessions := []session.Session{{
		Agent:   session.AgentCodex,
		Title:   "非常に長い日本語のタイトルが端末の右端を越えて折り返されないことを確認するセッション",
		WorkDir: "/projects/a-very-long-project-directory-that-must-be-truncated",
		Model:   "a-very-long-model-name",
	}}
	m := NewModel(sessions, "0.4.0", true)
	m.width = 50

	for _, line := range strings.Split(m.View(), "\n") {
		if width := ansi.StringWidth(line); width > m.width {
			t.Fatalf("rendered line width = %d; want <= %d: %q", width, m.width, line)
		}
	}
}

func TestCurrentProjectViewHidesWorkingDirectory(t *testing.T) {
	sessions := []session.Session{{
		Agent:   session.AgentCodex,
		Title:   "test session",
		WorkDir: "/projects/example",
	}}

	view := NewModel(sessions, "0.4.0", false).View()
	if strings.Contains(view, sessions[0].WorkDir) {
		t.Fatalf("current-project view unexpectedly shows working directory: %q", view)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "-"},
		{512, "512B"},
		{1536, "1.5K"},
		{3 * 1024 * 1024, "3.0M"},
		{2 * 1024 * 1024 * 1024, "2.0G"},
	}
	for _, tt := range tests {
		if got := formatSize(tt.n); got != tt.want {
			t.Errorf("formatSize(%d) = %q; want %q", tt.n, got, tt.want)
		}
	}
}

func TestSizeThresholds(t *testing.T) {
	const mb = 1024 * 1024
	tests := []struct {
		n           int64
		warn, large bool
	}{
		{3*mb - 1, false, false},
		{3 * mb, true, false},
		{10*mb - 1, true, false},
		{10 * mb, false, true},
	}
	for _, tt := range tests {
		if got := isWarnSize(tt.n); got != tt.warn {
			t.Errorf("isWarnSize(%d) = %v; want %v", tt.n, got, tt.warn)
		}
		if got := isLargeSize(tt.n); got != tt.large {
			t.Errorf("isLargeSize(%d) = %v; want %v", tt.n, got, tt.large)
		}
	}
}
