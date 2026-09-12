package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"agres/internal/session"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Padding(0, 0, 1, 0)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	agentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	projectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	modelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	warnSizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	largeSizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

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

const agentColumnWidth = len("[opencode]")

// Keep enough room for session titles even when a provider stores a long,
// fully-qualified model identifier.
const maxModelColumnWidth = 30

const sizeColumnWidth = len("999.9M")

// Size thresholds. Roughly 1MB of transcript corresponds to one full context
// window (one compaction). warnSizeThreshold marks histories that have been
// compacted a few times and are worth handing off to a new session (yellow);
// largeSizeThreshold marks histories that are clearly past that point and slow
// to resume (red).
const (
	warnSizeThreshold  int64 = 3 * 1024 * 1024
	largeSizeThreshold int64 = 10 * 1024 * 1024
)

func isWarnSize(n int64) bool  { return n >= warnSizeThreshold && n < largeSizeThreshold }
func isLargeSize(n int64) bool { return n >= largeSizeThreshold }

// formatSize renders a byte count compactly (e.g. "512B", "45.2K", "1.3M").
// Zero (unknown) renders as "-".
func formatSize(n int64) string {
	switch {
	case n <= 0:
		return "-"
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1fK", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1fM", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.1fG", float64(n)/(1024*1024*1024))
	}
}

func padLeft(s string, width int) string {
	padding := width - ansi.StringWidth(s)
	if padding <= 0 {
		return s
	}
	return strings.Repeat(" ", padding) + s
}

func padRight(s string, width int) string {
	padding := width - ansi.StringWidth(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func formatModel(model string, width int) string {
	if model == "" || width < 3 {
		return ""
	}
	return "[" + ansi.Truncate(model, width-2, "…") + "]"
}

func modelColumnWidth(sessions []session.Session) int {
	width := 0
	for _, s := range sessions {
		if s.Model == "" {
			continue
		}
		candidate := ansi.StringWidth(s.Model) + 2
		if candidate > maxModelColumnWidth {
			candidate = maxModelColumnWidth
		}
		if candidate > width {
			width = candidate
		}
	}
	return width
}

type Model struct {
	sessions    []session.Session
	cursor      int
	width       int
	height      int
	selected    *session.Session
	quitting    bool
	err         error
	version     string
	allProjects bool
}

func NewModel(sessions []session.Session, version string, allProjects bool) Model {
	return Model{
		sessions:    sessions,
		version:     version,
		allProjects: allProjects,
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

	title := "  agres " + m.version
	if m.allProjects {
		title += "  [all projects]"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if len(m.sessions) == 0 {
		b.WriteString(emptyStyle.Render("No agent sessions found in this directory."))
		b.WriteString("\n")
		return b.String()
	}

	viewWidth := m.width
	if viewWidth <= 0 {
		viewWidth = 80
	}
	if m.allProjects {
		workDir := m.sessions[m.cursor].WorkDir
		if workDir == "" {
			workDir = "(working directory unavailable)"
		}
		b.WriteString(ansi.Truncate(pathStyle.Render("  "+workDir), viewWidth, "…"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	modelWidth := modelColumnWidth(m.sessions)

	for i, s := range m.sessions {
		label := agentLabels[s.Agent]
		dateStr := s.UpdatedAt.Local().Format("2006-01-02 15:04:05")

		agentText := padRight(fmt.Sprintf("[%s]", label), agentColumnWidth)
		agentLabel := agentStyle.Render(agentText)
		sizeText := padLeft(formatSize(s.Size), sizeColumnWidth)
		sizeLabel := timeStyle.Render(sizeText)
		switch {
		case isLargeSize(s.Size):
			sizeLabel = largeSizeStyle.Render(sizeText)
		case isWarnSize(s.Size):
			sizeLabel = warnSizeStyle.Render(sizeText)
		}
		title := s.Title

		timeLabel := timeStyle.Render(dateStr)
		modelLabel := ""
		plainModelLabel := ""
		if modelWidth > 0 {
			modelText := padRight(formatModel(s.Model, modelWidth), modelWidth)
			modelLabel = "  " + modelStyle.Render(modelText)
			plainModelLabel = "  " + modelText
		}

		projectLabel := ""
		plainProjectLabel := ""
		if m.allProjects {
			workDir := s.WorkDir
			if workDir == "" {
				workDir = "(working directory unavailable)"
			}
			project := filepath.Base(filepath.Clean(workDir))
			if project == "." || project == string(filepath.Separator) {
				project = workDir
			}
			projectLabel = "  " + projectStyle.Render("["+project+"]")
			plainProjectLabel = "  [" + project + "]"
		}

		line := fmt.Sprintf("%s  %s%s  %s%s  %s", timeLabel, agentLabel, modelLabel, sizeLabel, projectLabel, title)
		available := viewWidth - 3
		if available < 1 {
			available = 1
		}
		if i == m.cursor {
			plainLine := fmt.Sprintf("%s  %s%s  %s%s  %s", dateStr, agentText, plainModelLabel, sizeText, plainProjectLabel, title)
			rowWidth := viewWidth - 1
			if rowWidth < 1 {
				rowWidth = 1
			}
			row := ansi.Truncate("   "+plainLine, rowWidth, "…")
			b.WriteString(selectedRowStyle.Width(rowWidth).Render(row))
		} else {
			b.WriteString(normalStyle.Render("   ") + ansi.Truncate(line, available, "…"))
		}
		b.WriteString("\n")
	}

	help := "  j/k or \u2191\u2193: navigate  enter: select  q/esc: quit"
	b.WriteString(helpStyle.Render(ansi.Truncate(help, viewWidth, "…")))

	return b.String()
}

func Run(sessions []session.Session, w io.Writer, version string, allProjects bool) (*session.Session, error) {
	p := tea.NewProgram(
		NewModel(sessions, version, allProjects),
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
