package schema

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// page represents a single view in the browser navigation stack.
type page struct {
	title string
	items []browserItem
}

// Browser is a Bubble Tea model that lets the user navigate a GraphQL schema
// with drill-down pages, breadcrumbs, and fuzzy search filtering.
type Browser struct {
	schema   *Schema
	stack    []page
	list     list.Model
	allItems []searchableItem // cross-level search index (built once per schema)

	width  int
	height int

	// Filter augmentation state: when the user enters filter mode we inject
	// cross-level items into the list so the built-in fuzzy filter can match
	// them. When filter mode ends we restore the original page items.
	filterAugmented bool
	pageItems       []browserItem // saved page items for restore after filtering
}

// NewBrowser returns a Browser with no schema loaded.
func NewBrowser() Browser {
	delegate := newBrowserDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	l.InfiniteScrolling = false

	// Customize list styles for vampire theme
	s := l.Styles
	s.NoItems = lipgloss.NewStyle().Faint(true).Padding(0, 2)
	s.StatusBar = lipgloss.NewStyle().Foreground(colorSubtle).Padding(0, 0, 0, 2)
	s.StatusBarActiveFilter = lipgloss.NewStyle().Foreground(colorRed)
	s.StatusBarFilterCount = lipgloss.NewStyle().Foreground(colorDim)
	s.ActivePaginationDot = lipgloss.NewStyle().Foreground(colorRed).SetString("●")
	s.InactivePaginationDot = lipgloss.NewStyle().Foreground(colorDim).SetString("○")
	s.PaginationStyle = lipgloss.NewStyle().Padding(0, 0, 0, 2)
	l.Styles = s

	// Customize filter input style
	l.FilterInput.Prompt = "/ "
	fiStyles := l.FilterInput.Styles()
	fiStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	fiStyles.Cursor.Color = colorRed
	l.FilterInput.SetStyles(fiStyles)

	// Disable quit keys (the app handles quit)
	l.DisableQuitKeybindings()

	return Browser{
		list: l,
	}
}

// Schema returns the currently loaded schema, or nil.
func (b *Browser) Schema() *Schema {
	return b.schema
}

// CanClose returns true when an esc press should close the browser
// (not filtering and at the root page).
func (b *Browser) CanClose() bool {
	return !b.list.SettingFilter() &&
		b.list.FilterState() == list.Unfiltered &&
		len(b.stack) <= 1
}

// SetSchema sets the schema and resets navigation to the root page.
func (b *Browser) SetSchema(s *Schema) {
	b.schema = s
	b.stack = nil
	b.allItems = allSearchableItems(s)
	b.filterAugmented = false
	b.pageItems = nil
	b.list.ResetFilter()
	if s != nil {
		b.pushRoot()
	}
	b.syncList()
}

// SetSize sets the viewport dimensions.
func (b *Browser) SetSize(w, h int) {
	b.width = w
	b.height = h
	b.syncList()
}

// Focus is a no-op for now.
func (b *Browser) Focus() {}

// Blur exits filter mode when the browser loses focus.
func (b *Browser) Blur() {
	if b.list.SettingFilter() {
		b.list.ResetFilter()
	}
}

// Update handles key messages for navigation and search.
func (b Browser) Update(msg tea.Msg) (Browser, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// When filtering, let the list handle everything
		if b.list.SettingFilter() {
			var cmd tea.Cmd
			b.list, cmd = b.list.Update(msg)

			// If user just exited filter mode (esc), restore original items
			if !b.list.SettingFilter() && b.list.FilterState() == list.Unfiltered {
				cmd = b.restorePageItems(cmd)
			}
			return b, cmd
		}

		// If filter was applied (user pressed enter in filter), handle nav
		if b.filterAugmented && b.list.FilterState() == list.FilterApplied {
			switch msg.String() {
			case "l", "enter", "right":
				if b.drillIn() {
					return b, nil
				}
			case "h", "backspace", "left":
				if b.goBack() {
					return b, nil
				}
				return b, nil
			case "esc":
				// Esc from FilterApplied resets filter
				cmd := b.restorePageItems(nil)
				b.list.ResetFilter()
				return b, cmd
			}
		}

		// Handle drill-in/back before the list processes the key
		switch msg.String() {
		case "l", "enter", "right":
			if b.drillIn() {
				return b, nil
			}
			// If not drillable, fall through to list
		case "h", "backspace", "left", "esc":
			if b.goBack() {
				return b, nil
			}
			return b, nil
		}
	}

	wasFiltering := b.list.SettingFilter()
	var cmd tea.Cmd
	b.list, cmd = b.list.Update(msg)

	// Detect transition into filter mode: inject cross-level items
	if !wasFiltering && b.list.SettingFilter() && !b.filterAugmented {
		cmd = b.augmentFilterItems(cmd)
	}

	return b, cmd
}

// View renders the current page with breadcrumbs and the list.
func (b Browser) View() string {
	if b.schema == nil || len(b.stack) == 0 {
		noSchema := lipgloss.NewStyle().
			Faint(true).
			Padding(1, 2).
			Width(b.width - 2).
			Height(b.height - 2)
		return noSchema.Render("No schema loaded.\nPress Ctrl+R to fetch.")
	}

	// Breadcrumbs
	titles := make([]string, len(b.stack))
	for i, p := range b.stack {
		titles[i] = p.title
	}
	crumbs := renderBreadcrumbs(titles, b.width-2)

	if crumbs != "" {
		return crumbs + "\n" + b.list.View()
	}
	return b.list.View()
}

// --- internal helpers ---

// syncList updates the list model's dimensions and content.
func (b *Browser) syncList() {
	headerLines := 0
	if len(b.stack) > 1 {
		headerLines = 1 // breadcrumbs
	}
	h := b.height - 2 - headerLines // 2 for border
	if h < 1 {
		h = 1
	}
	b.list.SetSize(b.width-2, h)

	if len(b.stack) == 0 {
		b.list.SetItems(nil)
		return
	}
	p := b.currentPage()
	items := make([]list.Item, len(p.items))
	for i, it := range p.items {
		items[i] = it
	}
	b.list.SetItems(items)
	b.list.SetStatusBarItemName("item", "items")
}

func (b *Browser) currentPage() page {
	return b.stack[len(b.stack)-1]
}

func (b *Browser) drillIn() bool {
	selected := b.list.SelectedItem()
	if selected == nil {
		return false
	}
	bi, ok := selected.(browserItem)
	if !ok || bi.target == "" {
		return false
	}
	b.resetFilterState()
	if bi.target == targetVariableTypes {
		b.pushVariableTypes()
	} else {
		b.pushType(bi.target)
	}
	return true
}

func (b *Browser) goBack() bool {
	if len(b.stack) <= 1 {
		return false
	}
	b.stack = b.stack[:len(b.stack)-1]
	b.resetFilterState()
	b.syncList()
	b.list.Select(0)
	return true
}

// resetFilterState clears filter augmentation and resets the list filter.
func (b *Browser) resetFilterState() {
	b.filterAugmented = false
	b.pageItems = nil
	b.list.ResetFilter()
}

// augmentFilterItems injects cross-level search items into the list when the
// user enters filter mode. The list's built-in fuzzy filter will match against
// both the current page items and these cross-level items.
func (b *Browser) augmentFilterItems(existing tea.Cmd) tea.Cmd {
	if len(b.allItems) == 0 || len(b.stack) == 0 {
		return existing
	}

	// Save the current page items for later restore
	p := b.currentPage()
	b.pageItems = make([]browserItem, len(p.items))
	copy(b.pageItems, p.items)

	// Build a set of current page item names to avoid duplicates
	currentNames := make(map[string]bool, len(p.items))
	currentPageType := ""
	if len(b.stack) > 1 {
		currentPageType = b.stack[len(b.stack)-1].title
	}
	for _, it := range p.items {
		currentNames[it.name] = true
	}

	// Collect cross-level items (items from other types)
	var crossLevel []browserItem
	for _, si := range b.allItems {
		// Skip items that belong to the current page type
		if si.parentName == currentPageType {
			continue
		}
		item := si.item
		item.searchParent = si.parentName
		// Cross-level items navigate to their parent type when selected
		item.target = si.parentName
		crossLevel = append(crossLevel, item)
	}

	// Combine: current page items first, then cross-level items
	combined := make([]list.Item, 0, len(p.items)+len(crossLevel))
	for _, it := range p.items {
		combined = append(combined, it)
	}
	for _, it := range crossLevel {
		combined = append(combined, it)
	}

	setCmd := b.list.SetItems(combined)
	b.filterAugmented = true

	return tea.Batch(existing, setCmd)
}

// restorePageItems restores the original page items after filter mode ends.
func (b *Browser) restorePageItems(existing tea.Cmd) tea.Cmd {
	if !b.filterAugmented || b.pageItems == nil {
		return existing
	}
	items := make([]list.Item, len(b.pageItems))
	for i, it := range b.pageItems {
		items[i] = it
	}
	setCmd := b.list.SetItems(items)
	b.filterAugmented = false
	b.pageItems = nil
	return tea.Batch(existing, setCmd)
}

func (b *Browser) pushRoot() {
	items := rootItems(b.schema)
	b.stack = append(b.stack, page{title: "Schema", items: items})
}

func (b *Browser) pushVariableTypes() {
	items := variableTypeItems(b.schema)
	b.stack = append(b.stack, page{title: "Variable Types", items: items})
	b.syncList()
	b.list.Select(0)
}

func (b *Browser) pushType(name string) {
	t := b.schema.TypeByName(name)
	if t == nil {
		return
	}
	items := typeItems(b.schema, name)
	title := t.Name
	b.stack = append(b.stack, page{title: title, items: items})
	b.syncList()
	b.list.Select(0)
}
