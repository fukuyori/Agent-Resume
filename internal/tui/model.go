package tui

import (
	"fmt"
	"io"
	"strings"

	"agres/internal/session"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Padding(0, 0, 1, 0)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	agentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	modelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Margin(1, 0)

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Padding(1, 2)
)

var agentLabels = map[session.Agent]string{
	session.AgentClaude:      "claude",
	session.AgentOpenCode:    "opencode",
	session.AgentAider:       "aider",
	session.AgentCodex:       "codex",
	session.AgentAntigravity: "agy",
}

type Model struct {
	sessions  []session.Session
	cursor    int
	width     int
	height    int
	selected  *session.Session
	quitting  bool
	err       error
	version   string
}

func NewModel(sessions []session.Session, version string) Model {
	return Model{
		sessions: sessions,
		version:  version,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}

		case "pgup":
			m.cursor -= 10
			if m.cursor < 0 {
				m.cursor = 0
			}

		case "pgdown":
			m.cursor += 10
			if m.cursor >= len(m.sessions) {
				m.cursor = len(m.sessions) - 1
			}

		case "home", "g":
			m.cursor = 0

		case "end", "G":
			m.cursor = len(m.sessions) - 1

		case "enter":
			if len(m.sessions) > 0 {
				s := m.sessions[m.cursor]
				m.selected = &s
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("  agres " + m.version))
	b.WriteString("\n\n")

	if len(m.sessions) == 0 {
		b.WriteString(emptyStyle.Render("No agent sessions found in this directory."))
		b.WriteString("\n")
		return b.String()
	}

	for i, s := range m.sessions {
		label := agentLabels[s.Agent]
		dateStr := s.UpdatedAt.Format("2006-01-02 15:04:05")

		agentLabel := agentStyle.Render(fmt.Sprintf("[%s]", label))
		title := s.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		timeLabel := timeStyle.Render(dateStr)
		modelLabel := ""
		if s.Model != "" {
			modelLabel = " " + modelStyle.Render(s.Model)
		}

		line := fmt.Sprintf("%s  %s  %s%s", timeLabel, agentLabel, title, modelLabel)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(" > ") + line)
		} else {
			b.WriteString(normalStyle.Render("   ") + line)
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  j/k or \u2191\u2193: navigate  enter: select  q/esc: quit"))

	return b.String()
}

func Run(sessions []session.Session, w io.Writer, version string) (*session.Session, error) {
	p := tea.NewProgram(
		NewModel(sessions, version),
		tea.WithOutput(w),
	)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	m := finalModel.(Model)
	if m.selected != nil {
		return m.selected, nil
	}
	return nil, nil
}
