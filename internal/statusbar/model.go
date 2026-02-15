package statusbar

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	barStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

type Model struct {
	text  string
	width int
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

func (m *Model) SetLoading() {
	m.text = barStyle.Render("Executing query...")
}

func (m *Model) SetAborted() {
	m.text = warnStyle.Render("Query aborted")
}

func (m *Model) Clear() {
	m.text = barStyle.Render("Ready")
}

func (m Model) View() string {
	return " " + m.text
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
