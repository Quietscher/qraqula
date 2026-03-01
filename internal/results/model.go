package results

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/qraqula/qla/internal/highlight"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// ANSI codes for search match highlighting (yellow background, preserves foreground).
const (
	matchBgOn  = "\x1b[43m"
	matchBgOff = "\x1b[49m"
)

type matchPos struct {
	line int
	col  int
}

type Model struct {
	vp                 viewport.Model
	width              int
	height             int
	rawContent         string
	highlightedContent string

	searching   bool
	searchQuery string
	matches     []matchPos
	matchIdx    int
	searchInput textinput.Model
}

func New(width, height int) Model {
	vp := viewport.New()
	vp.SetWidth(width)
	vp.SetHeight(height - 1) // title line
	si := textinput.New()
	si.Placeholder = "search..."
	si.CharLimit = 200
	return Model{vp: vp, width: width, height: height, searchInput: si}
}

func (m *Model) SetContent(s string) {
	m.rawContent = s
	m.highlightedContent = s
	m.vp.SetContent(s)
}

func (m *Model) SetPrettyJSON(data []byte) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return err
	}
	plain := buf.String()
	m.rawContent = plain
	highlighted := highlight.Colorize(plain, "json")
	m.highlightedContent = highlighted
	m.vp.SetContent(highlighted)
	return nil
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.vp.SetWidth(w - 2)
	searchH := 0
	if m.searching {
		searchH = 1
	}
	m.vp.SetHeight(h - 3 - searchH)
}

func (m *Model) Focus() {}
func (m *Model) Blur()  {}

// Content returns the raw (uncolored) result content.
func (m Model) Content() string { return m.rawContent }

// Search accessors

func (m Model) Searching() bool   { return m.searching }
func (m Model) MatchCount() int   { return len(m.matches) }
func (m Model) CurrentMatch() int { return m.matchIdx }

func (m *Model) ToggleSearch() {
	m.searching = !m.searching
	if m.searching {
		m.searchInput.SetValue("")
		m.searchQuery = ""
		m.matches = nil
		m.matchIdx = 0
		m.searchInput.Focus()
	} else {
		m.searchInput.Blur()
		m.searchQuery = ""
		m.matches = nil
		// Restore content without search highlights
		m.vp.SetContent(m.highlightedContent)
	}
}

func (m *Model) SetSearchQuery(q string) {
	m.searchQuery = q
	m.matches = nil
	m.matchIdx = 0
	if q == "" {
		m.vp.SetContent(m.highlightedContent)
		return
	}
	qLower := strings.ToLower(q)
	lines := strings.Split(m.rawContent, "\n")
	for i, line := range lines {
		lineLower := strings.ToLower(line)
		idx := 0
		for {
			pos := strings.Index(lineLower[idx:], qLower)
			if pos < 0 {
				break
			}
			m.matches = append(m.matches, matchPos{line: i, col: idx + pos})
			idx += pos + len(qLower)
		}
	}
	m.updateViewportContent()
}

func (m *Model) NextMatch() {
	if len(m.matches) == 0 {
		return
	}
	m.matchIdx = (m.matchIdx + 1) % len(m.matches)
	m.scrollToMatch()
}

func (m *Model) PrevMatch() {
	if len(m.matches) == 0 {
		return
	}
	m.matchIdx = (m.matchIdx - 1 + len(m.matches)) % len(m.matches)
	m.scrollToMatch()
}

func (m *Model) scrollToMatch() {
	if m.matchIdx >= len(m.matches) {
		return
	}
	target := m.matches[m.matchIdx].line
	m.vp.SetYOffset(target)
}

// updateViewportContent re-renders the viewport with search match highlights.
func (m *Model) updateViewportContent() {
	if m.searchQuery == "" || len(m.matches) == 0 {
		m.vp.SetContent(m.highlightedContent)
		return
	}

	hLines := strings.Split(m.highlightedContent, "\n")
	rLines := strings.Split(m.rawContent, "\n")

	// Group match ranges by line
	qLen := len(m.searchQuery)
	lineRanges := make(map[int][][2]int)
	for _, mp := range m.matches {
		lineRanges[mp.line] = append(lineRanges[mp.line], [2]int{mp.col, mp.col + qLen})
	}

	for lineIdx, ranges := range lineRanges {
		if lineIdx < len(hLines) && lineIdx < len(rLines) {
			hLines[lineIdx] = insertMatchHighlights(hLines[lineIdx], ranges)
		}
	}

	m.vp.SetContent(strings.Join(hLines, "\n"))
}

// insertMatchHighlights adds yellow background ANSI codes around match ranges
// in an already-highlighted (ANSI-coded) line. ranges are byte positions in the
// raw (uncolored) text; the function maps them through ANSI escape sequences.
func insertMatchHighlights(highlighted string, ranges [][2]int) string {
	if len(ranges) == 0 {
		return highlighted
	}

	var buf strings.Builder
	buf.Grow(len(highlighted) + len(ranges)*20)

	visPos := 0 // byte position in raw text (skipping ANSI codes)
	ri := 0     // current range index
	inMatch := false

	for i := 0; i < len(highlighted); {
		// Skip ANSI escape sequences
		if highlighted[i] == '\x1b' && i+1 < len(highlighted) && highlighted[i+1] == '[' {
			j := i + 2
			for j < len(highlighted) && highlighted[j] != 'm' {
				j++
			}
			if j < len(highlighted) {
				j++ // include 'm'
			}
			buf.WriteString(highlighted[i:j])
			// Re-apply our background after any ANSI code inside a match
			if inMatch {
				buf.WriteString(matchBgOn)
			}
			i = j
			continue
		}

		// Start match highlight
		if !inMatch && ri < len(ranges) && visPos == ranges[ri][0] {
			buf.WriteString(matchBgOn)
			inMatch = true
		}

		buf.WriteByte(highlighted[i])
		i++
		visPos++

		// End match highlight
		if inMatch && ri < len(ranges) && visPos == ranges[ri][1] {
			buf.WriteString(matchBgOff)
			inMatch = false
			ri++
			// Check if next range starts immediately
			if ri < len(ranges) && visPos == ranges[ri][0] {
				buf.WriteString(matchBgOn)
				inMatch = true
			}
		}
	}

	if inMatch {
		buf.WriteString(matchBgOff)
	}

	return buf.String()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.searching {
		if kmsg, ok := msg.(tea.KeyPressMsg); ok {
			switch kmsg.String() {
			case "enter":
				// Confirm search and unfocus input (keep search visible, allow n/N)
				m.searchInput.Blur()
				return m, nil
			}
		}

		if m.searchInput.Focused() {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			if m.searchInput.Value() != m.searchQuery {
				m.SetSearchQuery(m.searchInput.Value())
			}
			return m, cmd
		}

		// Search active but input not focused — handle n/N for navigation
		if kmsg, ok := msg.(tea.KeyPressMsg); ok {
			switch kmsg.String() {
			case "n":
				m.NextMatch()
				return m, nil
			case "N", "shift+n":
				m.PrevMatch()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := titleStyle.Render(" Result ")
	if m.searching {
		searchLine := m.searchInput.View()
		if len(m.matches) > 0 {
			info := fmt.Sprintf(" %d/%d", m.matchIdx+1, len(m.matches))
			searchLine += dimStyle.Render(info)
		} else if m.searchQuery != "" {
			searchLine += dimStyle.Render(" no matches")
		}
		return title + "\n" + searchLine + "\n" + m.vp.View()
	}
	return title + "\n" + m.vp.View()
}
