package history

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Vampire theme colors (matching schema browser)
var (
	colorRed    = lipgloss.Color("196")
	colorDim    = lipgloss.Color("241")
	colorSubtle = lipgloss.Color("245")
	colorWhite  = lipgloss.Color("252")

	hSelectedBar  = lipgloss.NewStyle().Foreground(colorRed).SetString("▌")
	hNormalPrefix = lipgloss.NewStyle().SetString(" ")
	hTitleStyle   = lipgloss.NewStyle().Foreground(colorWhite)
	hSelTitle     = lipgloss.NewStyle().Foreground(colorWhite).Bold(true)
	hDimStyle     = lipgloss.NewStyle().Foreground(colorDim)
	hFolderStyle  = lipgloss.NewStyle().Foreground(colorSubtle)
	hSepLabel     = lipgloss.NewStyle().Foreground(colorSubtle).Bold(true)
	hSepLine      = lipgloss.NewStyle().Foreground(colorDim)
)

type itemKind int

const (
	kindFolder    itemKind = iota
	kindEntry
	kindSeparator
)

// sidebarItem is a list item for the history sidebar.
type sidebarItem struct {
	kind      itemKind
	name      string // display name
	folder    string // parent folder name (empty for folders/separator)
	entryID   string // entry ID (empty for folders/separator)
	endpoint  string // dim suffix for entries
	collapsed bool   // only for kindFolder
}

func (i sidebarItem) Title() string       { return i.name }
func (i sidebarItem) Description() string { return i.endpoint }
func (i sidebarItem) FilterValue() string { return i.name + " " + i.endpoint }

// scrollState holds the marquee scroll state shared between sidebar and delegate.
type scrollState struct {
	offset  int  // current rune offset into the name
	active  bool // whether scrolling is active (name is truncated)
	paused  bool // manual scroll pauses auto-scroll until re-selection
	lastIdx int  // last selected index (to detect changes)
}

// sidebarDelegate renders history sidebar items with the vampire theme.
type sidebarDelegate struct {
	scroll *scrollState
}

func newSidebarDelegate() sidebarDelegate {
	return sidebarDelegate{scroll: &scrollState{lastIdx: -1}}
}

func (d sidebarDelegate) Height() int  { return 1 }
func (d sidebarDelegate) Spacing() int { return 0 }

func (d sidebarDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d sidebarDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(sidebarItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	// Match schema browser: subtract padding from list width
	width := m.Width() - 4
	if width < 4 {
		width = 4
	}

	var line string
	switch si.kind {
	case kindFolder:
		line = d.renderFolder(si, isSelected, width)
	case kindEntry:
		line = d.renderEntry(si, isSelected, width)
	case kindSeparator:
		line = d.renderSeparator(width)
	}

	// Hard cap: force exactly `width` visible chars, no wrapping possible
	fmt.Fprint(w, lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line))
}

// truncateVisual cuts a string to fit within maxWidth visible chars, adding … if needed.
func truncateVisual(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return "…"
}

// marquee returns a sliding window of a string at the given rune offset.
// If offset is 0 or the string fits, it behaves like truncateVisual.
func marquee(s string, maxWidth, offset int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	if offset >= len(runes) {
		offset = 0
	}
	// Take a substring starting at offset
	sub := runes[offset:]
	candidate := string(sub)
	if lipgloss.Width(candidate) <= maxWidth {
		return candidate
	}
	return truncateVisual(candidate, maxWidth)
}

// needsScroll returns true if the name would be truncated at the given width.
func needsScroll(s string, maxWidth int) bool {
	return lipgloss.Width(s) > maxWidth
}

func (d sidebarDelegate) renderFolder(si sidebarItem, selected bool, width int) string {
	var prefix string
	if selected {
		prefix = hSelectedBar.String()
	} else {
		prefix = hNormalPrefix.String()
	}

	var icon string
	if si.collapsed {
		icon = "📁 "
	} else {
		icon = "📂 "
	}

	prefixW := lipgloss.Width(prefix)
	iconW := lipgloss.Width(icon)
	nameMax := width - prefixW - iconW
	if nameMax < 1 {
		nameMax = 1
	}

	var name string
	if selected && d.scroll != nil {
		name = marquee(si.name, nameMax, d.scroll.offset)
	} else {
		name = truncateVisual(si.name, nameMax)
	}
	var nameStr string
	if selected {
		nameStr = hSelTitle.Render(name)
	} else {
		nameStr = hTitleStyle.Render(name)
	}

	return prefix + icon + nameStr
}

func (d sidebarDelegate) renderEntry(si sidebarItem, selected bool, width int) string {
	var prefix string
	if selected {
		prefix = hSelectedBar.String()
	} else {
		prefix = hNormalPrefix.String()
	}

	var indent string
	if si.folder != "" {
		indent = "  "
	}

	bullet := hDimStyle.Render("·") + " "

	prefixW := lipgloss.Width(prefix)
	indentW := lipgloss.Width(indent)
	bulletW := lipgloss.Width(bullet)
	overhead := prefixW + indentW + bulletW

	nameMax := width - overhead
	if nameMax < 1 {
		nameMax = 1
	}

	var name string
	if selected && d.scroll != nil {
		name = marquee(si.name, nameMax, d.scroll.offset)
	} else {
		name = truncateVisual(si.name, nameMax)
	}
	var nameStr string
	if selected {
		nameStr = hSelTitle.Render(name)
	} else {
		nameStr = hTitleStyle.Render(name)
	}

	return prefix + indent + bullet + nameStr
}

func (d sidebarDelegate) renderSeparator(width int) string {
	label := hSepLabel.Render(" Unsorted ")
	lineChar := "─"
	labelWidth := lipgloss.Width(label)
	prefixW := 1
	remaining := width - labelWidth - prefixW
	if remaining < 2 {
		remaining = 2
	}
	left := remaining / 3
	right := remaining - left
	leftLine := hSepLine.Render(strings.Repeat(lineChar, left))
	rightLine := hSepLine.Render(strings.Repeat(lineChar, right))
	return " " + leftLine + label + rightLine
}
