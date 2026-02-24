package variables

import (
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

type Model struct {
	ta     textarea.Model
	width  int
	height int
}

func New() Model {
	ta := textarea.New()
	ta.Placeholder = `{"key": "value"}`
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	return Model{ta: ta}
}

func (m Model) Value() string {
	return m.ta.Value()
}

func (m *Model) SetValue(s string) {
	m.ta.SetValue(s)
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
	m.ta.SetWidth(w - 2)
	m.ta.SetHeight(h - 3)
}

func (m Model) ParsedVariables() (map[string]any, error) {
	v := strings.TrimSpace(m.ta.Value())
	if v == "" {
		return nil, nil
	}
	var vars map[string]any
	if err := json.Unmarshal([]byte(v), &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := titleStyle.Render(" Variables ")
	return title + "\n" + m.ta.View()
}
