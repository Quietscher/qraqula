package schema

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Vampire theme colors
var (
	colorRed     = lipgloss.Color("196")
	colorMagenta = lipgloss.Color("162")
	colorYellow  = lipgloss.Color("214")
	colorCyan    = lipgloss.Color("37")
	colorBlue    = lipgloss.Color("62")
	colorDim     = lipgloss.Color("241")
	colorSubtle  = lipgloss.Color("245")
	colorWhite   = lipgloss.Color("252")

	// Badge styles by kind
	badgeOBJECT       = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	badgeINTERFACE    = lipgloss.NewStyle().Foreground(colorMagenta).Bold(true)
	badgeENUM         = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	badgeINPUT_OBJECT = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	badgeUNION        = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)

	// Item styles
	selectedBar    = lipgloss.NewStyle().Foreground(colorRed).SetString("▌ ")
	normalPrefix   = lipgloss.NewStyle().SetString("  ")
	titleStyle     = lipgloss.NewStyle().Foreground(colorWhite)
	selTitleStyle  = lipgloss.NewStyle().Foreground(colorWhite).Bold(true)
	descStyle      = lipgloss.NewStyle().Foreground(colorSubtle)
	dimTitleStyle  = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	arrowStyle     = lipgloss.NewStyle().Foreground(colorDim)
	dimNoteStyle   = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
	argsStyle         = lipgloss.NewStyle().Foreground(colorDim)
	separatorStyle    = lipgloss.NewStyle().Foreground(colorDim)
	searchParentStyle = lipgloss.NewStyle().Faint(true)
	searchSepStyle    = lipgloss.NewStyle().Faint(true).SetString(" › ")

	// Breadcrumb styles
	crumbDimStyle     = lipgloss.NewStyle().Faint(true)
	crumbCurrentStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	crumbSepStyle     = lipgloss.NewStyle().Faint(true).SetString(" › ")
)

// browserDelegate renders schema browser items with the vampire theme.
type browserDelegate struct{}

func newBrowserDelegate() browserDelegate {
	return browserDelegate{}
}

func (d browserDelegate) Height() int  { return 1 }
func (d browserDelegate) Spacing() int { return 0 }

func (d browserDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d browserDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	bi, ok := item.(browserItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	width := m.Width() - 4 // padding

	// Prefix: selection bar or indent
	var prefix string
	if isSelected {
		prefix = selectedBar.String()
	} else {
		prefix = normalPrefix.String()
	}

	// Cross-level search parent prefix
	var mainText string
	if bi.searchParent != "" {
		mainText = searchParentStyle.Render(bi.searchParent) + searchSepStyle.String()
	}

	// Build the main text: color-coded for fields, plain for non-fields
	if bi.fieldName != "" {
		// Color-coded field rendering
		if bi.deprecated {
			mainText += dimTitleStyle.Render(bi.fieldName)
		} else if isSelected {
			mainText += selTitleStyle.Render(bi.fieldName)
		} else {
			mainText += titleStyle.Render(bi.fieldName)
		}
		if bi.fieldArgs != "" {
			mainText += argsStyle.Render(bi.fieldArgs)
		}
		mainText += separatorStyle.Render(": ")
		mainText += typeStyleFor(bi.fieldTypeKind).Render(bi.fieldType)
	} else {
		// Non-field items (type names, enum values, "implements X")
		if bi.deprecated {
			mainText += dimTitleStyle.Render(bi.name)
		} else if isSelected {
			mainText += selTitleStyle.Render(bi.name)
		} else {
			mainText += titleStyle.Render(bi.name)
		}
	}

	// Inline description after name (subtle, space-permitting)
	if bi.desc != "" && bi.fieldName == "" {
		mainText += "  " + descStyle.Render(bi.desc)
	}

	// Dim note (deprecation reason)
	if bi.dimNote != "" {
		mainText += "  " + dimNoteStyle.Render(bi.dimNote)
	}

	// Right side: badge + arrow
	var right string
	if bi.badge != "" {
		right = badgeFor(bi.badge).Render(bi.badge)
	}
	if bi.Drillable() {
		if right != "" {
			right += " "
		}
		right += arrowStyle.Render("→")
	}

	// Pad to push right content to edge
	mainWidth := lipgloss.Width(prefix) + lipgloss.Width(mainText)
	rightWidth := lipgloss.Width(right)
	gap := width - mainWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	fmt.Fprint(w, prefix+mainText+strings.Repeat(" ", gap)+right)
}

// typeStyleFor returns a color style for a field's return type based on its kind.
func typeStyleFor(kind string) lipgloss.Style {
	switch kind {
	case "OBJECT":
		return lipgloss.NewStyle().Foreground(colorRed)
	case "INTERFACE":
		return lipgloss.NewStyle().Foreground(colorMagenta)
	case "ENUM":
		return lipgloss.NewStyle().Foreground(colorYellow)
	case "INPUT_OBJECT":
		return lipgloss.NewStyle().Foreground(colorCyan)
	case "UNION":
		return lipgloss.NewStyle().Foreground(colorBlue)
	default:
		return lipgloss.NewStyle().Foreground(colorSubtle)
	}
}

func badgeFor(kind string) lipgloss.Style {
	switch kind {
	case "OBJECT":
		return badgeOBJECT
	case "INTERFACE":
		return badgeINTERFACE
	case "ENUM":
		return badgeENUM
	case "INPUT_OBJECT":
		return badgeINPUT_OBJECT
	case "UNION":
		return badgeUNION
	default:
		return lipgloss.NewStyle().Faint(true)
	}
}

// renderBreadcrumbs renders the navigation breadcrumb bar.
func renderBreadcrumbs(titles []string, width int) string {
	if len(titles) <= 1 {
		return ""
	}
	var parts []string
	for i, t := range titles {
		if i == len(titles)-1 {
			parts = append(parts, crumbCurrentStyle.Render(t))
		} else {
			parts = append(parts, crumbDimStyle.Render(t))
		}
	}
	line := strings.Join(parts, crumbSepStyle.String())
	// Truncate if too wide
	if lipgloss.Width(line) > width-2 {
		line = line[:width-2]
	}
	return "  " + line
}
