package results

import (
	"bytes"
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

type Model struct {
	vp     viewport.Model
	width  int
	height int
}

func New(width, height int) Model {
	vp := viewport.New(width, height-1) // title line
	return Model{vp: vp, width: width, height: height}
}

func (m *Model) SetContent(s string) {
	m.vp.SetContent(s)
}

func (m *Model) SetPrettyJSON(data []byte) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return err
	}
	m.vp.SetContent(buf.String())
	return nil
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.vp.Width = w - 2
	m.vp.Height = h - 3
}

func (m *Model) Focus() {}
func (m *Model) Blur()  {}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := titleStyle.Render(" Result ")
	return title + "\n" + m.vp.View()
}
