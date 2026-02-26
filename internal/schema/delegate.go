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

// browserScrollState holds the marquee state for the schema browser.
type browserScrollState struct {
	offset int
	active bool
}

// browserDelegate renders schema browser items with the vampire theme.
type browserDelegate struct {
	scroll *browserScrollState
}

func newBrowserDelegate(scroll *browserScrollState) browserDelegate {
	return browserDelegate{scroll: scroll}
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

	prefixW := lipgloss.Width(prefix)
	rightW := lipgloss.Width(right)
	contentW := width - prefixW - rightW - 1 // 1 for gap
	if contentW < 1 {
		contentW = 1
	}

	// Cross-level search parent prefix
	var parentPrefix string
	if bi.searchParent != "" {
		parentPrefix = searchParentStyle.Render(bi.searchParent) + searchSepStyle.String()
		contentW -= lipgloss.Width(parentPrefix)
		if contentW < 1 {
			contentW = 1
		}
	}

	var mainText string

	if bi.fieldName != "" {
		// Field item: keep the colored type always visible, truncate/marquee the name+args
		typeStr := typeStyleFor(bi.fieldTypeKind).Render(bi.fieldType)
		typeW := lipgloss.Width(typeStr)
		scrollW := contentW - typeW
		if scrollW < 1 {
			scrollW = 1
		}

		// The scrollable part: "fieldName(args): "
		scrollText := bi.fieldName
		if bi.fieldArgs != "" {
			scrollText += bi.fieldArgs
		}
		scrollText += ": "

		var visibleScroll string
		scrollOffset := 0
		if isSelected && d.scroll != nil {
			scrollOffset = d.scroll.offset
		}
		if isSelected && d.scroll != nil && d.scroll.active {
			visibleScroll = marquee(scrollText, scrollW, scrollOffset)
		} else {
			visibleScroll = truncateVisual(scrollText, scrollW)
		}

		// Style the visible scroll portion: split into name, args, separator parts
		styledScroll := d.styleFieldScroll(visibleScroll, bi, isSelected, scrollOffset)
		// Pad scrollable portion to fixed width so the Type stays in place
		scrollVisW := lipgloss.Width(styledScroll)
		if scrollVisW < scrollW {
			styledScroll += strings.Repeat(" ", scrollW-scrollVisW)
		}
		mainText = styledScroll + typeStr
	} else {
		// Non-field items: truncate/marquee the full text
		scrollText := bi.scrollableText()

		var visible string
		scrollOffset := 0
		if isSelected && d.scroll != nil {
			scrollOffset = d.scroll.offset
		}
		if isSelected && d.scroll != nil && d.scroll.active {
			visible = marquee(scrollText, contentW, scrollOffset)
		} else {
			visible = truncateVisual(scrollText, contentW)
		}

		// Style the visible text
		nameLen := len([]rune(bi.name))
		nameRemaining := nameLen - scrollOffset
		if nameRemaining < 0 {
			nameRemaining = 0
		}
		visRunes := []rune(visible)
		if nameRemaining > len(visRunes) {
			nameRemaining = len(visRunes)
		}

		namePart := string(visRunes[:nameRemaining])
		descPart := string(visRunes[nameRemaining:])

		if namePart != "" {
			if bi.deprecated {
				mainText = dimTitleStyle.Render(namePart)
			} else if isSelected {
				mainText = selTitleStyle.Render(namePart)
			} else {
				mainText = titleStyle.Render(namePart)
			}
		}
		if descPart != "" {
			mainText += descStyle.Render(descPart)
		}
	}

	// Dim note (deprecation reason)
	if bi.dimNote != "" {
		mainText += "  " + dimNoteStyle.Render(bi.dimNote)
	}

	// Pad to push right content to edge
	mainWidth := lipgloss.Width(prefix) + lipgloss.Width(parentPrefix) + lipgloss.Width(mainText)
	rightWidth := lipgloss.Width(right)
	gap := width - mainWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	fmt.Fprint(w, prefix+parentPrefix+mainText+strings.Repeat(" ", gap)+right)
}

// styleFieldScroll applies correct styling to the visible portion of a field's
// scrollable text (fieldName + args + ": "). It tracks which part of the original
// text is visible based on the scroll offset.
func (d browserDelegate) styleFieldScroll(visible string, bi browserItem, isSelected bool, offset int) string {
	visRunes := []rune(visible)
	nameLen := len([]rune(bi.fieldName))
	argsLen := len([]rune(bi.fieldArgs))
	sepLen := 2 // ": "

	// Calculate remaining characters of each part at current offset
	nameRemaining := nameLen - offset
	if nameRemaining < 0 {
		nameRemaining = 0
	}
	argsRemaining := nameLen + argsLen - offset
	if argsRemaining < 0 {
		argsRemaining = 0
	}
	argsStart := nameRemaining
	sepStart := argsRemaining
	if argsStart > len(visRunes) {
		argsStart = len(visRunes)
	}
	if sepStart > len(visRunes) {
		sepStart = len(visRunes)
	}
	_ = sepLen // used conceptually

	var result string

	// Name part
	if argsStart > 0 {
		namePart := string(visRunes[:argsStart])
		if bi.deprecated {
			result += dimTitleStyle.Render(namePart)
		} else if isSelected {
			result += selTitleStyle.Render(namePart)
		} else {
			result += titleStyle.Render(namePart)
		}
	}

	// Args part
	if sepStart > argsStart {
		argsPart := string(visRunes[argsStart:sepStart])
		result += argsStyle.Render(argsPart)
	}

	// Separator part
	if sepStart < len(visRunes) {
		sepPart := string(visRunes[sepStart:])
		result += separatorStyle.Render(sepPart)
	}

	return result
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

// contentWidthFor returns the available content width for an item's main text,
// accounting for the prefix bar, right-side badge/arrow, and gap.
func contentWidthFor(bi browserItem, totalWidth int) int {
	prefixW := lipgloss.Width(selectedBar.String()) // same width whether selected or not
	var rightW int
	if bi.badge != "" {
		rightW += lipgloss.Width(badgeFor(bi.badge).Render(bi.badge))
	}
	if bi.Drillable() {
		if rightW > 0 {
			rightW++ // space
		}
		rightW += lipgloss.Width(arrowStyle.Render("→"))
	}
	if bi.searchParent != "" {
		rightW += lipgloss.Width(searchParentStyle.Render(bi.searchParent) + searchSepStyle.String())
	}
	w := totalWidth - prefixW - rightW - 1
	if w < 1 {
		w = 1
	}
	return w
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
