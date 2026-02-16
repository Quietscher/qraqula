package schema

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// page represents a single view in the browser navigation stack.
type page struct {
	title string
	items []pageItem
}

// pageItem is a single selectable row in a page.
type pageItem struct {
	label      string // display text
	typeName   string // target type name for drill-in (empty if not drillable)
	deprecated bool
	dimNote    string // e.g. "deprecated: use name"
}

// Browser is a Bubble Tea model that lets the user navigate a GraphQL schema
// with drill-down pages and breadcrumbs.
type Browser struct {
	schema *Schema
	stack  []page // navigation stack; last element is current page
	cursor int    // selected index on the current page
	offset int    // scroll offset for long lists
	width  int
	height int
}

// style constants
var (
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	dimStyle        = lipgloss.NewStyle().Faint(true)
	breadcrumbStyle = lipgloss.NewStyle().Faint(true)
	badgeStyle      = lipgloss.NewStyle().Faint(true)
	titleStyle_b    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
)

// NewBrowser returns a Browser with no schema loaded.
func NewBrowser() Browser {
	return Browser{}
}

// SetSchema sets the schema and resets navigation to the root page.
func (b *Browser) SetSchema(s *Schema) {
	b.schema = s
	b.stack = nil
	b.cursor = 0
	b.offset = 0
	if s != nil {
		b.pushRoot()
	}
}

// SetSize sets the viewport dimensions.
func (b *Browser) SetSize(w, h int) {
	b.width = w
	b.height = h
}

// Focus is a no-op for now.
func (b *Browser) Focus() {}

// Blur is a no-op for now.
func (b *Browser) Blur() {}

// Update handles key messages for navigation.
func (b Browser) Update(msg tea.Msg) (Browser, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			b.cursorDown()
		case "k", "up":
			b.cursorUp()
		case "enter":
			b.drillIn()
		case "backspace", "h":
			b.goBack()
		}
	}
	return b, nil
}

// View renders the current page.
func (b Browser) View() string {
	if b.schema == nil {
		return "No schema loaded"
	}
	if len(b.stack) == 0 {
		return "No schema loaded"
	}

	p := b.currentPage()
	var sb strings.Builder

	// Breadcrumbs
	if len(b.stack) > 1 {
		var crumbs []string
		for _, pg := range b.stack[:len(b.stack)-1] {
			crumbs = append(crumbs, pg.title)
		}
		bc := strings.Join(crumbs, " > ")
		sb.WriteString(breadcrumbStyle.Render(bc))
		sb.WriteString("\n")
	}

	// Title
	sb.WriteString(titleStyle_b.Render(p.title))
	sb.WriteString("\n")

	// How many lines are used by header (breadcrumbs + title + blank line)
	headerLines := 2 // title + blank
	if len(b.stack) > 1 {
		headerLines = 3 // breadcrumb + title + blank
	}
	sb.WriteString("\n")

	// Visible area for items
	visibleHeight := b.height - headerLines
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Adjust scroll offset
	b.adjustOffset(visibleHeight)

	// Render visible items
	end := b.offset + visibleHeight
	if end > len(p.items) {
		end = len(p.items)
	}
	for i := b.offset; i < end; i++ {
		item := p.items[i]
		line := b.renderItem(i, item)
		sb.WriteString(line)
		if i < end-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// --- internal helpers ---

func (b *Browser) currentPage() page {
	return b.stack[len(b.stack)-1]
}

func (b *Browser) adjustOffset(visibleHeight int) {
	// Ensure cursor is visible
	if b.cursor < b.offset {
		b.offset = b.cursor
	}
	if b.cursor >= b.offset+visibleHeight {
		b.offset = b.cursor - visibleHeight + 1
	}
	if b.offset < 0 {
		b.offset = 0
	}
}

func (b *Browser) renderItem(index int, item pageItem) string {
	prefix := "  "
	if index == b.cursor {
		prefix = cursorStyle.Render("> ")
	}

	label := item.label
	if item.deprecated {
		label = dimStyle.Render(label)
		if item.dimNote != "" {
			label += " " + dimStyle.Render(item.dimNote)
		}
	}

	return prefix + label
}

func (b *Browser) cursorDown() {
	if len(b.stack) == 0 {
		return
	}
	p := b.currentPage()
	if b.cursor < len(p.items)-1 {
		b.cursor++
	}
}

func (b *Browser) cursorUp() {
	if b.cursor > 0 {
		b.cursor--
	}
}

func (b *Browser) drillIn() {
	if len(b.stack) == 0 {
		return
	}
	p := b.currentPage()
	if b.cursor >= len(p.items) {
		return
	}
	item := p.items[b.cursor]
	if item.typeName == "" {
		return
	}
	b.pushType(item.typeName)
}

func (b *Browser) goBack() {
	if len(b.stack) <= 1 {
		return
	}
	b.stack = b.stack[:len(b.stack)-1]
	b.cursor = 0
	b.offset = 0
}

func (b *Browser) pushRoot() {
	roots := b.schema.RootTypes()
	items := make([]pageItem, 0, len(roots))
	for _, rt := range roots {
		items = append(items, pageItem{
			label:    fmt.Sprintf("%s %s", rt.Name, badgeStyle.Render("["+rt.Kind+"]")),
			typeName: rt.Name,
		})
	}
	b.stack = append(b.stack, page{title: "Schema", items: items})
	b.cursor = 0
	b.offset = 0
}

func (b *Browser) pushType(name string) {
	t := b.schema.TypeByName(name)
	if t == nil {
		return
	}

	var items []pageItem

	switch t.Kind {
	case "OBJECT", "INTERFACE":
		// Show interfaces if any
		if len(t.Interfaces) > 0 {
			for _, iface := range t.Interfaces {
				ifName := iface.NamedType()
				items = append(items, pageItem{
					label:    fmt.Sprintf("implements %s", ifName),
					typeName: ifName,
				})
			}
		}
		// Fields
		for _, f := range t.Fields {
			display := b.fieldDisplay(f)
			named := f.Type.NamedType()
			drillable := b.isDrillable(named)
			target := ""
			if drillable {
				target = named
			}
			items = append(items, pageItem{
				label:      display,
				typeName:   target,
				deprecated: f.IsDeprecated,
				dimNote:    deprecationNote(f.DeprecationReason),
			})
		}

	case "ENUM":
		for _, ev := range t.EnumValues {
			items = append(items, pageItem{
				label:      ev.Name,
				deprecated: ev.IsDeprecated,
				dimNote:    deprecationNote(ev.DeprecationReason),
			})
		}

	case "INPUT_OBJECT":
		for _, iv := range t.InputFields {
			display := fmt.Sprintf("%s: %s", iv.Name, iv.Type.DisplayName())
			named := iv.Type.NamedType()
			drillable := b.isDrillable(named)
			target := ""
			if drillable {
				target = named
			}
			items = append(items, pageItem{
				label:    display,
				typeName: target,
			})
		}

	case "UNION":
		for _, pt := range t.PossibleTypes {
			ptName := pt.NamedType()
			items = append(items, pageItem{
				label:    ptName,
				typeName: ptName,
			})
		}
	}

	title := fmt.Sprintf("%s %s", t.Name, badgeStyle.Render("["+t.Kind+"]"))
	b.stack = append(b.stack, page{title: title, items: items})
	b.cursor = 0
	b.offset = 0
}

func (b *Browser) fieldDisplay(f Field) string {
	// Format: name(arg: Type, ...): ReturnType
	var sb strings.Builder
	sb.WriteString(f.Name)
	if len(f.Args) > 0 {
		sb.WriteString("(")
		for i, a := range f.Args {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(a.Name)
			sb.WriteString(": ")
			sb.WriteString(a.Type.DisplayName())
		}
		sb.WriteString(")")
	}
	sb.WriteString(": ")
	sb.WriteString(f.Type.DisplayName())
	return sb.String()
}

func (b *Browser) isDrillable(name string) bool {
	if name == "" {
		return false
	}
	t := b.schema.TypeByName(name)
	if t == nil {
		return false
	}
	switch t.Kind {
	case "OBJECT", "INTERFACE", "ENUM", "INPUT_OBJECT", "UNION":
		return true
	}
	return false
}

func deprecationNote(reason string) string {
	if reason == "" {
		return ""
	}
	return fmt.Sprintf("(deprecated: %s)", reason)
}
