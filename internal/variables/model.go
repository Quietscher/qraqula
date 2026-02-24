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
	ta      textarea.Model
	width   int
	height  int
	editing bool
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
	return nil
}

func (m *Model) Blur() {
	if m.editing {
		m.editing = false
		m.ta.Blur()
	}
}

func (m Model) Focused() bool {
	return m.ta.Focused()
}

func (m Model) Editing() bool {
	return m.editing
}

func (m *Model) StartEditing() tea.Cmd {
	m.editing = true
	return m.ta.Focus()
}

func (m *Model) StopEditing() {
	m.editing = false
	m.ta.Blur()
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
	if !m.editing {
		return m, nil
	}
	if kmsg, ok := msg.(tea.KeyPressMsg); ok && kmsg.String() == "tab" {
		m.ta.InsertString("\t")
		return m, nil
	}
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := titleStyle.Render(" Variables ")
	return title + "\n" + m.ta.View()
}
