package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/achernya/autorip/makemkv"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	padding  = 2
	maxWidth = 160
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type model struct {
	progress progress.Model
	total    string
	current  string
	logs     string
	viewport viewport.Model
}

type Eof struct {
	empty int
}

func (m *model) detail() string {
	if len(m.current) > 0 {
		return fmt.Sprintf("%s / %s", m.total, m.current)
	}
	if len(m.total) > 0 {
		return m.total
	}
	return "[ no detailed status yet ]"
}

func NewTui() tea.Model {
	vp := viewport.New(maxWidth-2, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingLeft(1).
		MarginLeft(padding)
	return &model{
		progress: progress.New(progress.WithDefaultGradient()),
		viewport: vp,
	}
}

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		m.viewport.Width = m.progress.Width - padding
		return m, nil

	case *makemkv.StreamResult:
		switch msg := msg.Parsed.(type) {
		case *makemkv.ProgressTitle:
			if msg.Type == makemkv.ProgressTotal {
				m.total = msg.Name
				m.current = ""
			} else {
				m.current = msg.Name
			}
			m.addLog("[stage] " + m.detail())
			return m, nil

		case *makemkv.Message:
			m.addLog(msg.Message)
			return m, nil

		case *makemkv.ProgressUpdate:
			var cmds []tea.Cmd
			// Note that you can also use progress.Model.SetPercent to set the
			// percentage value explicitly, too.
			cmds = append(cmds, m.progress.SetPercent(float64(msg.Total)/float64(msg.Max)))
			return m, tea.Batch(cmds...)
		default:
			return m, nil
		}
	case Eof:
		return m, tea.Sequence(finalPause(), tea.Quit)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m *model) addLog(log string) {
	now := time.Now()
	m.logs += now.Format(time.TimeOnly) + " | " + log + "\n"
	m.viewport.SetContent(m.logs)
	m.viewport.GotoBottom()
}

func (m *model) headerView() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n" +
		pad + m.detail() + "\n\n"

}

func (m *model) footerView() string {
	pad := strings.Repeat(" ", padding)
	return pad + helpStyle(" ↑/↓: Navigate • ctrl+c: Quit\n")
}

func (m *model) View() string {
	return m.headerView() +
		m.viewport.View() + "\n" +
		m.footerView()
}
