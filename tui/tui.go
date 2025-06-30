package tui

import (
	"strings"
	"time"
	"fmt"

	"github.com/achernya/autorip/makemkv"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type model struct {
	progress progress.Model
	total    string
	current  string
}

type Eof struct{
	empty int
}

func (m model) detail() string {
	if len(m.total) > 0 || len(m.current) > 0 {
		return fmt.Sprintf("%s / %s", m.total, m.current)
	}
	return "[ no detailed status yet ]"
}

func NewTui() tea.Model {
	return model{
		progress: progress.New(progress.WithDefaultGradient()),
	}
}

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case *makemkv.ProgressTitle:
		if msg.Type == makemkv.ProgressTotal {
			m.total = msg.Name
		} else {
			m.current = msg.Name
		}
		return m, nil
		
	case *makemkv.ProgressUpdate:
		var cmds []tea.Cmd
		// Note that you can also use progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		cmds = append(cmds, m.progress.SetPercent(float64(msg.Total) / float64(msg.Max)))
		return m, tea.Batch(cmds...)
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

func (m model) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n" +
		pad + m.detail() + "\n\n" +
		pad + helpStyle("Press any key to quit")
}
