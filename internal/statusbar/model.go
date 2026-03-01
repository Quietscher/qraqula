package statusbar

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

var (
	barStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	keyStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// Hint represents a single keybinding hint shown in the status bar.
type Hint struct {
	Key   string
	Label string
}

type Model struct {
	text  string
	width int
	hints []Hint
}

func New() Model {
	return Model{text: barStyle.Render("Ready")}
}

func (m *Model) SetWidth(w int) {
	m.width = w
}

func (m *Model) SetResult(statusCode int, duration time.Duration, size int, hasErrors bool) {
	status := fmt.Sprintf("%d", statusCode)
	if hasErrors {
		status += " (with errors)"
		m.text = warnStyle.Render(status) + barStyle.Render(fmt.Sprintf("  %s  %s", formatDuration(duration), formatSize(size)))
	} else {
		m.text = okStyle.Render(status) + barStyle.Render(fmt.Sprintf("  %s  %s", formatDuration(duration), formatSize(size)))
	}
}

func (m *Model) SetError(msg string) {
	m.text = errStyle.Render("Error: " + msg)
}

func (m *Model) SetInfo(msg string) {
	m.text = okStyle.Render(msg)
}

func (m *Model) SetLoading() {
	m.text = barStyle.Render("Executing query...")
}

func (m *Model) SetAborted() {
	m.text = warnStyle.Render("Query aborted")
}

func (m *Model) SetSchemaLoading() {
	m.text = barStyle.Render("Fetching schema...")
}

func (m *Model) SetSchemaLoaded(typeCount int) {
	m.text = okStyle.Render(fmt.Sprintf("Schema loaded (%d types)", typeCount))
}

func (m *Model) Clear() {
	m.text = barStyle.Render("Ready")
}

// SetHints sets the keybinding hints shown on the right side of the status bar.
func (m *Model) SetHints(hints []Hint) {
	m.hints = hints
}

func (m Model) View() string {
	hints := m.renderHints()
	gap := m.width - lipgloss.Width(m.text) - lipgloss.Width(hints) - 2 // 2 for padding
	if gap < 1 {
		gap = 1
	}
	return " " + m.text + strings.Repeat(" ", gap) + hints
}

func (m Model) renderHints() string {
	hints := m.hints
	if len(hints) == 0 {
		// Default hints when none set
		hints = []Hint{
			{"alt+↵", "execute"},
			{"tab", "next"},
			{"^d", "docs"},
			{"^r", "schema"},
			{"/", "search"},
			{"^c", "abort"},
			{"^q", "quit"},
		}
	}
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = keyStyle.Render(h.Key) + " " + labelStyle.Render(h.Label)
	}
	return strings.Join(parts, "  ")
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
}
