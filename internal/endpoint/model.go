package endpoint

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

var (
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

type Model struct {
	input   textinput.Model
	width   int
	envName string
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

func (m *Model) SetValue(s string) {
	m.input.SetValue(s)
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

func (m *Model) SetEnvName(name string) {
	m.envName = name
}

func (m Model) EnvName() string {
	return m.envName
}

func (m *Model) SetWidth(w int) {
	m.width = w
	labelW := 12 // "Endpoint: " + padding
	m.input.SetWidth(w - labelW)
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
