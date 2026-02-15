package editor

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

type Model struct {
	ta    textarea.Model
	width int
	height int
}

func New() Model {
	ta := textarea.New()
	ta.Placeholder = "{ query { ... } }"
	ta.ShowLineNumbers = false
	return Model{ta: ta}
}

func (m Model) Value() string {
	return m.ta.Value()
}

func (m *Model) Focus() tea.Cmd {
	return m.ta.Focus()
}

func (m *Model) Blur() {
	m.ta.Blur()
}

func (m Model) Focused() bool {
	return m.ta.Focused()
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.ta.SetWidth(w - 2)  // border
	m.ta.SetHeight(h - 3) // border + title
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := titleStyle.Render(" Query ")
	return title + "\n" + m.ta.View()
}
