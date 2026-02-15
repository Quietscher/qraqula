package endpoint

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

type Model struct {
	input textinput.Model
	width int
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "https://api.example.com/graphql"
	ti.CharLimit = 500
	return Model{input: ti}
}

func (m Model) Value() string {
	return m.input.Value()
}

func (m *Model) Focus() tea.Cmd {
	return m.input.Focus()
}

func (m *Model) Blur() {
	m.input.Blur()
}

func (m Model) Focused() bool {
	return m.input.Focused()
}

func (m *Model) SetWidth(w int) {
	m.width = w
	m.input.Width = w - 12 // account for label
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	label := labelStyle.Render("Endpoint: ")
	return label + m.input.View()
}
