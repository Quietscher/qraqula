package schema

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
)

// style constants
var (
	cursorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	dimStyle          = lipgloss.NewStyle().Faint(true)
	breadcrumbStyle   = lipgloss.NewStyle().Faint(true)
	badgeStyle        = lipgloss.NewStyle().Faint(true)
	browserTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	searchPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	countStyle        = lipgloss.NewStyle().Faint(true)
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
// with drill-down pages, breadcrumbs, and search filtering.
type Browser struct {
	schema *Schema
	stack  []page // navigation stack; last element is current page
	cursor int    // selected index on the current (filtered) page
	offset int    // scroll offset for viewport content management

	vp        viewport.Model
	search    textinput.Model
	searching bool
	filter    string

	width  int
	height int
}

// NewBrowser returns a Browser with no schema loaded.
func NewBrowser() Browser {
	si := textinput.New()
	si.Placeholder = "type to filter..."
	si.CharLimit = 100

	return Browser{
		vp:     viewport.New(),
		search: si,
	}
}

// SetSchema sets the schema and resets navigation to the root page.
func (b *Browser) SetSchema(s *Schema) {
	b.schema = s
	b.stack = nil
	b.cursor = 0
	b.offset = 0
	b.filter = ""
	b.searching = false
	b.search.SetValue("")
	if s != nil {
		b.pushRoot()
	}
	b.syncViewport()
}

// SetSize sets the viewport dimensions.
func (b *Browser) SetSize(w, h int) {
	b.width = w
	b.height = h
	b.syncViewport()
}

// Focus is a no-op for now.
func (b *Browser) Focus() {}

// Blur exits search mode when the browser loses focus.
func (b *Browser) Blur() {
	if b.searching {
		b.searching = false
		b.search.Blur()
		b.syncViewport()
	}
}

// Update handles key messages for navigation and search.
func (b Browser) Update(msg tea.Msg) (Browser, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if b.searching {
			return b.handleSearchKey(msg)
		}
		return b.handleNavKey(msg)
	}
	return b, nil
}

func (b *Browser) handleSearchKey(msg tea.KeyPressMsg) (Browser, tea.Cmd) {
	switch msg.String() {
	case "esc":
		b.searching = false
		b.filter = ""
		b.search.SetValue("")
		b.search.Blur()
		b.cursor = 0
		b.offset = 0
		b.syncViewport()
		return *b, nil
	case "enter":
		b.searching = false
		b.search.Blur()
		b.syncViewport()
		return *b, nil
	}

	var cmd tea.Cmd
	b.search, cmd = b.search.Update(msg)
	newFilter := b.search.Value()
	if newFilter != b.filter {
		b.filter = newFilter
		b.cursor = 0
		b.offset = 0
		b.syncViewport()
	}
	return *b, cmd
}

func (b *Browser) handleNavKey(msg tea.KeyPressMsg) (Browser, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		b.cursorDown()
	case "k", "up":
		b.cursorUp()
	case "enter", "l":
		b.drillIn()
	case "backspace", "h":
		b.goBack()
	case "/":
		b.searching = true
		cmd := b.search.Focus()
		b.syncViewport()
		return *b, cmd
	case "G":
		b.cursorEnd()
	}
	return *b, nil
}

// View renders the current page with header and scrollable content.
func (b Browser) View() string {
	if b.schema == nil || len(b.stack) == 0 {
		lines := []string{dimStyle.Render("No schema loaded. Press Ctrl+R to fetch.")}
		target := b.height - 2
		for len(lines) < target {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	var header []string

	// Breadcrumbs
	if len(b.stack) > 1 {
		var crumbs []string
		for _, pg := range b.stack[:len(b.stack)-1] {
			crumbs = append(crumbs, pg.title)
		}
		header = append(header, breadcrumbStyle.Render(strings.Join(crumbs, " > ")))
	}

	// Title + item count
	p := b.currentPage()
	filtered := b.filteredItems()
	title := browserTitleStyle.Render(p.title)
	if b.filter != "" {
		title += " " + countStyle.Render(fmt.Sprintf("(%d/%d)", len(filtered), len(p.items)))
	} else {
		title += " " + countStyle.Render(fmt.Sprintf("(%d items)", len(p.items)))
	}
	header = append(header, title)

	// Search bar
	if b.searching {
		header = append(header, searchPromptStyle.Render("/")+b.search.View())
	}

	return strings.Join(header, "\n") + "\n" + b.vp.View()
}

// --- internal helpers ---

// vpHeight returns available height for the viewport content area.
func (b *Browser) vpHeight() int {
	h := b.height - 2 - b.headerLines() // subtract border (2) + header
	if h < 1 {
		h = 1
	}
	return h
}

// headerLines returns the number of lines used by the header above the viewport.
func (b *Browser) headerLines() int {
	n := 1 // title
	if len(b.stack) > 1 {
		n++ // breadcrumbs
	}
	if b.searching {
		n++ // search input
	}
	return n
}

// syncViewport rebuilds the viewport content from current state.
func (b *Browser) syncViewport() {
	b.vp.SetWidth(b.width - 2)
	b.vp.SetHeight(b.vpHeight())

	if b.schema == nil || len(b.stack) == 0 {
		b.vp.SetContent("")
		return
	}

	items := b.filteredItems()
	vh := b.vpHeight()

	if len(items) == 0 {
		b.cursor = 0
		b.offset = 0
		b.vp.SetContent(dimStyle.Render("  No matching items"))
		return
	}
	if b.cursor >= len(items) {
		b.cursor = len(items) - 1
	}

	// Keep cursor visible
	if b.cursor < b.offset {
		b.offset = b.cursor
	}
	if b.cursor >= b.offset+vh {
		b.offset = b.cursor - vh + 1
	}
	if b.offset < 0 {
		b.offset = 0
	}

	// Build visible lines
	end := b.offset + vh
	if end > len(items) {
		end = len(items)
	}
	var lines []string
	for i := b.offset; i < end; i++ {
		lines = append(lines, b.renderItem(i, items[i]))
	}

	b.vp.SetContent(strings.Join(lines, "\n"))
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

func (b *Browser) filteredItems() []pageItem {
	if len(b.stack) == 0 {
		return nil
	}
	p := b.currentPage()
	if b.filter == "" {
		return p.items
	}
	lower := strings.ToLower(b.filter)
	var out []pageItem
	for _, item := range p.items {
		if strings.Contains(strings.ToLower(item.label), lower) {
			out = append(out, item)
		}
	}
	return out
}

func (b *Browser) currentPage() page {
	return b.stack[len(b.stack)-1]
}

func (b *Browser) cursorDown() {
	items := b.filteredItems()
	if b.cursor < len(items)-1 {
		b.cursor++
		b.syncViewport()
	}
}

func (b *Browser) cursorUp() {
	if b.cursor > 0 {
		b.cursor--
		b.syncViewport()
	}
}

func (b *Browser) cursorEnd() {
	items := b.filteredItems()
	if len(items) > 0 {
		b.cursor = len(items) - 1
		b.syncViewport()
	}
}

func (b *Browser) drillIn() {
	if len(b.stack) == 0 {
		return
	}
	items := b.filteredItems()
	if b.cursor >= len(items) {
		return
	}
	item := items[b.cursor]
	if item.typeName == "" {
		return
	}
	// Clear filter when drilling in
	b.filter = ""
	b.search.SetValue("")
	b.searching = false
	b.pushType(item.typeName)
	b.syncViewport()
}

func (b *Browser) goBack() {
	if len(b.stack) <= 1 {
		return
	}
	b.stack = b.stack[:len(b.stack)-1]
	b.cursor = 0
	b.offset = 0
	b.filter = ""
	b.search.SetValue("")
	b.searching = false
	b.syncViewport()
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
	b.syncViewport()
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
