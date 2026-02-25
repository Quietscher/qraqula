package history

import (
	"fmt"
	"time"

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
	hSepLabel     = lipgloss.NewStyle().Foreground(colorSubtle).Bold(true)
	hSepLine      = lipgloss.NewStyle().Foreground(colorDim)
)

type itemKind int

const (
	kindFolder itemKind = iota
	kindEntry
)

// sidebarItem represents a row in the history sidebar.
type sidebarItem struct {
	kind      itemKind
	name      string    // display name
	folder    string    // parent folder name (empty for folders/unsorted)
	entryID   string    // entry ID (empty for folders)
	endpoint  string    // dim suffix for entries
	collapsed bool      // only for kindFolder
	createdAt time.Time // entry timestamp
}

// scrollState holds the marquee scroll state.
type scrollState struct {
	offset  int  // current rune offset into the name
	active  bool // whether scrolling is active (name is truncated)
	paused  bool // manual scroll pauses auto-scroll until re-selection
	lastIdx int  // last selected index (to detect changes)
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

// renderFolderLine renders a folder item as a single line.
func renderFolderLine(si sidebarItem, selected bool, width, scrollOffset int) string {
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
	if selected {
		name = marquee(si.name, nameMax, scrollOffset)
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

// formatTimestamp formats a timestamp for display in the sidebar.
// Today: "14:05", otherwise: "14:05 DD.MM.YYYY".
func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	timeStr := fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
	if t.After(today) {
		return timeStr
	}
	return timeStr + " " + fmt.Sprintf("%02d.%02d.%d", t.Day(), int(t.Month()), t.Year())
}

// timestampWidth returns the visual width a timestamp would occupy (including leading space).
func timestampWidth(ts string) int {
	if ts == "" {
		return 0
	}
	return 1 + len(ts) // " " + timestamp
}

// renderEntryLine renders an entry item as a single line.
func renderEntryLine(si sidebarItem, selected bool, width, scrollOffset int) string {
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

	availW := width - overhead
	if availW < 1 {
		availW = 1
	}

	// Determine if timestamp fits alongside the name (need at least 3 chars for name)
	ts := formatTimestamp(si.createdAt)
	tsW := timestampWidth(ts)
	showTs := tsW > 0 && availW-tsW >= 3

	nameMax := availW
	if showTs {
		nameMax = availW - tsW
	}

	var name string
	if selected {
		name = marquee(si.name, nameMax, scrollOffset)
	} else {
		name = truncateVisual(si.name, nameMax)
	}
	var nameStr string
	if selected {
		nameStr = hSelTitle.Render(name)
	} else {
		nameStr = hTitleStyle.Render(name)
	}

	var tsSuffix string
	if showTs {
		tsSuffix = " " + hDimStyle.Render(ts)
	}

	return prefix + indent + bullet + nameStr + tsSuffix
}
