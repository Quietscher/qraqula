package history

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	scrollDelay    = 800 * time.Millisecond // pause before auto-scroll starts
	scrollInterval = 150 * time.Millisecond // time between scroll steps
)

// scrollTickMsg is sent to advance the marquee scroll.
type scrollTickMsg struct{}

// LoadEntryMsg is sent when an entry is selected for loading.
type LoadEntryMsg struct {
	Entry Entry
}

// SidebarUpdatedMsg signals the sidebar content changed and needs re-render.
type SidebarUpdatedMsg struct{}

// Sidebar is the Bubble Tea model for the history sidebar.
type Sidebar struct {
	store  *Store
	list   list.Model
	width  int
	height int
	open   bool

	// Rename mode
	renaming       bool
	renameInput    textinput.Model
	renameID       string // entry ID or folder name
	renameIsFolder bool

	// Search mode
	searching   bool
	searchInput textinput.Model

	// Move mode: track folders auto-opened during move so we can re-close them
	autoOpened map[string]bool

	// Marquee scroll state (shared with delegate)
	scroll *scrollState
}

// NewSidebar creates a sidebar with the given store.
func NewSidebar(store *Store) Sidebar {
	delegate := newSidebarDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(false) // we handle search ourselves
	l.InfiniteScrolling = false

	// Vampire theme styles
	s := l.Styles
	s.NoItems = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	s.ActivePaginationDot = lipgloss.NewStyle().Foreground(colorRed).SetString("●")
	s.InactivePaginationDot = lipgloss.NewStyle().Foreground(colorDim).SetString("○")
	s.PaginationStyle = lipgloss.NewStyle().Padding(0, 0, 0, 1)
	l.Styles = s

	l.DisableQuitKeybindings()

	ti := textinput.New()
	ti.CharLimit = 50

	si := textinput.New()
	si.Prompt = "/ "
	si.CharLimit = 50
	siStyles := si.Styles()
	siStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	siStyles.Cursor.Color = colorRed
	si.SetStyles(siStyles)

	sb := Sidebar{
		store:       store,
		list:        l,
		open:        store.Meta.SidebarOpen,
		renameInput: ti,
		searchInput: si,
		scroll:      delegate.scroll,
	}
	sb.Rebuild()
	return sb
}

// SetSize sets the viewport dimensions.
func (sb *Sidebar) SetSize(w, h int) {
	sb.width = w
	sb.height = h
	sb.syncListSize()
}

func (sb *Sidebar) syncListSize() {
	lh := sb.height
	if sb.renaming {
		lh -= 2 // rename prompt takes 2 lines (label + input)
	}
	if sb.searching {
		lh-- // search input takes 1 line
	}
	if lh < 1 {
		lh = 1
	}
	sb.list.SetSize(sb.width, lh)
	sb.renameInput.SetWidth(sb.width)
	sb.searchInput.SetWidth(sb.width)
}

// SetOpen sets sidebar visibility.
func (sb *Sidebar) SetOpen(open bool) { sb.open = open }

// IsOpen returns whether the sidebar is visible.
func (sb *Sidebar) IsOpen() bool { return sb.open }

// Focus is a no-op for now (list handles its own focus).
func (sb *Sidebar) Focus() {}

// Blur exits filter mode and rename mode when the sidebar loses focus.
func (sb *Sidebar) Blur() {
	sb.cancelSearch()
	sb.cancelRename()
	sb.commitAutoOpened()
}

// Rebuild rebuilds list items from store data, preserving cursor position.
func (sb *Sidebar) Rebuild() {
	sb.rebuildAndSelect("")
}

// RebuildAndFollow rebuilds list items and selects the entry with the given ID.
func (sb *Sidebar) RebuildAndFollow(entryID string) {
	sb.rebuildAndSelect(entryID)
}

func (sb *Sidebar) rebuildAndSelect(followID string) {
	prevIdx := sb.list.Index()
	var items []list.Item

	folders := sb.store.Folders()
	unsorted := sb.store.Unsorted()

	// Add folders (in order)
	for _, f := range folders {
		collapsed := sb.store.IsCollapsed(f.Name)
		items = append(items, sidebarItem{
			kind:      kindFolder,
			name:      f.Name,
			collapsed: collapsed,
		})
		if !collapsed {
			for _, e := range f.Entries {
				items = append(items, sidebarItem{
					kind:     kindEntry,
					name:     e.Name,
					folder:   f.Name,
					entryID:  e.ID,
					endpoint: e.Endpoint,
				})
			}
		}
	}

	// Only show separator when there are folders AND unsorted entries
	if len(folders) > 0 && len(unsorted) > 0 {
		items = append(items, sidebarItem{
			kind: kindSeparator,
			name: "Unsorted",
		})
	}

	for _, e := range unsorted {
		items = append(items, sidebarItem{
			kind:     kindEntry,
			name:     e.Name,
			folder:   "",
			entryID:  e.ID,
			endpoint: e.Endpoint,
		})
	}

	sb.list.SetItems(items)

	// If following a specific entry, find its index
	targetIdx := -1
	if followID != "" {
		for i, item := range items {
			if si, ok := item.(sidebarItem); ok && si.entryID == followID {
				targetIdx = i
				break
			}
		}
	}

	if targetIdx >= 0 {
		sb.list.Select(targetIdx)
	} else {
		// Restore cursor position, clamped to valid range
		if prevIdx >= len(items) {
			prevIdx = len(items) - 1
		}
		if prevIdx < 0 {
			prevIdx = 0
		}
		sb.list.Select(prevIdx)
	}
	sb.skipSeparator(1) // ensure we're not on a separator
	sb.resetScrollState()
}

// SelectedEntry returns the currently selected entry, or nil if none.
func (sb *Sidebar) SelectedEntry() *Entry {
	sel := sb.list.SelectedItem()
	if sel == nil {
		return nil
	}
	si, ok := sel.(sidebarItem)
	if !ok || si.kind != kindEntry {
		return nil
	}
	return sb.findEntry(si.entryID)
}

// SelectedFolder returns the folder name if a folder is selected.
func (sb *Sidebar) SelectedFolder() string {
	sel := sb.list.SelectedItem()
	if sel == nil {
		return ""
	}
	si, ok := sel.(sidebarItem)
	if !ok || si.kind != kindFolder {
		return ""
	}
	return si.name
}

func (sb *Sidebar) findEntry(id string) *Entry {
	for i := range sb.store.Unsorted() {
		if sb.store.unsorted[i].ID == id {
			return &sb.store.unsorted[i]
		}
	}
	for fi := range sb.store.folders {
		for ei := range sb.store.folders[fi].Entries {
			if sb.store.folders[fi].Entries[ei].ID == id {
				return &sb.store.folders[fi].Entries[ei]
			}
		}
	}
	return nil
}

// skipSeparator moves the cursor off separator items in the given direction.
func (sb *Sidebar) skipSeparator(dir int) {
	items := sb.list.Items()
	if len(items) == 0 {
		return
	}
	idx := sb.list.Index()
	for idx >= 0 && idx < len(items) {
		si, ok := items[idx].(sidebarItem)
		if !ok || si.kind != kindSeparator {
			break
		}
		idx += dir
	}
	// Clamp
	if idx < 0 {
		idx = 0
	}
	if idx >= len(items) {
		idx = len(items) - 1
	}
	// If still on separator (edge case: all items are separators), just stay
	sb.list.Select(idx)
}

// nameMaxForItem computes the max name width for an item, matching delegate overhead.
func (sb *Sidebar) nameMaxForItem(si sidebarItem) int {
	width := sb.width - 4
	if width < 4 {
		width = 4
	}
	overhead := 1 // prefix (▌ or space)
	switch si.kind {
	case kindFolder:
		if si.collapsed {
			overhead += lipgloss.Width("📁 ")
		} else {
			overhead += lipgloss.Width("📂 ")
		}
	case kindEntry:
		if si.folder != "" {
			overhead += 2 // indent
		}
		overhead += lipgloss.Width(hDimStyle.Render("·") + " ")
	}
	nameMax := width - overhead
	if nameMax < 1 {
		nameMax = 1
	}
	return nameMax
}

// resetScrollState resets the marquee scroll to initial position.
func (sb *Sidebar) resetScrollState() {
	if sb.scroll != nil {
		sb.scroll.offset = 0
		sb.scroll.active = false
		sb.scroll.paused = false
	}
}

// resetAndScheduleScroll resets scroll state and schedules an auto-scroll check.
func (sb *Sidebar) resetAndScheduleScroll() tea.Cmd {
	sb.resetScrollState()
	if sb.scroll != nil {
		sb.scroll.lastIdx = sb.list.Index()
	}
	return tea.Tick(scrollDelay, func(time.Time) tea.Msg { return scrollTickMsg{} })
}

// handleScrollTick advances the marquee animation by one step.
func (sb Sidebar) handleScrollTick() (Sidebar, tea.Cmd) {
	if sb.scroll == nil || sb.searching || sb.renaming || sb.scroll.paused {
		return sb, nil
	}
	// Ignore stale ticks from previous selections
	if sb.scroll.lastIdx != sb.list.Index() {
		return sb, nil
	}
	sel := sb.list.SelectedItem()
	if sel == nil {
		return sb, nil
	}
	si, ok := sel.(sidebarItem)
	if !ok || si.kind == kindSeparator {
		return sb, nil
	}

	nameMax := sb.nameMaxForItem(si)
	if !needsScroll(si.name, nameMax) {
		sb.scroll.active = false
		return sb, nil
	}

	sb.scroll.active = true
	runes := []rune(si.name)
	sb.scroll.offset++
	if sb.scroll.offset >= len(runes) {
		sb.scroll.offset = 0
		// Pause at the beginning before restarting
		return sb, tea.Tick(scrollDelay, func(time.Time) tea.Msg { return scrollTickMsg{} })
	}
	return sb, tea.Tick(scrollInterval, func(time.Time) tea.Msg { return scrollTickMsg{} })
}

// handleManualScroll adjusts scroll offset for left/right arrow keys.
// Returns true if the key was handled (item needs scrolling).
func (sb *Sidebar) handleManualScroll(key string) bool {
	if sb.scroll == nil {
		return false
	}
	sel := sb.list.SelectedItem()
	if sel == nil {
		return false
	}
	si, ok := sel.(sidebarItem)
	if !ok || si.kind == kindSeparator {
		return false
	}

	nameMax := sb.nameMaxForItem(si)
	if !needsScroll(si.name, nameMax) {
		return false
	}

	sb.scroll.active = true
	sb.scroll.paused = true
	runes := []rune(si.name)
	switch key {
	case "left":
		if sb.scroll.offset > 0 {
			sb.scroll.offset--
		}
	case "right":
		if sb.scroll.offset < len(runes)-1 {
			sb.scroll.offset++
		}
	}
	return true
}

// Update handles key messages for sidebar navigation.
func (sb Sidebar) Update(msg tea.Msg) (Sidebar, tea.Cmd) {
	switch msg := msg.(type) {
	case scrollTickMsg:
		return sb.handleScrollTick()
	case tea.KeyPressMsg:
		// Rename mode: route all keys to textinput
		if sb.renaming {
			return sb.updateRename(msg)
		}

		// Search mode: route keys to search handler
		if sb.searching {
			return sb.updateSearch(msg)
		}

		prevIdx := sb.list.Index()

		switch msg.String() {
		case "m":
			return sb.handleMove(1)
		case "M":
			return sb.handleMove(-1)
		default:
			// Any non-move key commits auto-opened state
			sb.commitAutoOpened()
		}

		switch msg.String() {
		case "/":
			return sb.startSearch()
		case "enter", "l":
			return sb.handleSelect()
		case "h":
			return sb.handleCollapse()
		case "N":
			return sb.handleCreateFolder()
		case "r":
			return sb.startRename()
		case "d":
			return sb.handleDelete()
		case "left", "right":
			if sb.handleManualScroll(msg.String()) {
				return sb, nil
			}
		}

		// Let list handle navigation (j/k/up/down etc.)
		var cmd tea.Cmd
		sb.list, cmd = sb.list.Update(msg)

		// After navigation, skip separator items and schedule scroll
		newIdx := sb.list.Index()
		if newIdx != prevIdx {
			dir := 1
			if newIdx < prevIdx {
				dir = -1
			}
			sb.skipSeparator(dir)
			scrollCmd := sb.resetAndScheduleScroll()
			return sb, tea.Batch(cmd, scrollCmd)
		}

		return sb, cmd
	}

	var cmd tea.Cmd
	sb.list, cmd = sb.list.Update(msg)
	return sb, cmd
}

func (sb Sidebar) handleSelect() (Sidebar, tea.Cmd) {
	sel := sb.list.SelectedItem()
	if sel == nil {
		return sb, nil
	}
	si, ok := sel.(sidebarItem)
	if !ok {
		return sb, nil
	}

	switch si.kind {
	case kindFolder:
		// Toggle expand/collapse
		collapsed := !si.collapsed
		sb.store.SetCollapsed(si.name, collapsed)
		_ = sb.store.Save()
		sb.Rebuild()
		return sb, nil
	case kindEntry:
		entry := sb.findEntry(si.entryID)
		if entry != nil {
			return sb, func() tea.Msg { return LoadEntryMsg{Entry: *entry} }
		}
	}
	return sb, nil
}

func (sb Sidebar) handleCollapse() (Sidebar, tea.Cmd) {
	sel := sb.list.SelectedItem()
	if sel == nil {
		return sb, nil
	}
	si, ok := sel.(sidebarItem)
	if !ok {
		return sb, nil
	}
	if si.kind == kindFolder && !si.collapsed {
		sb.store.SetCollapsed(si.name, true)
		_ = sb.store.Save()
		sb.Rebuild()
	}
	return sb, nil
}

func (sb Sidebar) handleCreateFolder() (Sidebar, tea.Cmd) {
	name := "New Folder"
	i := 1
	for sb.folderExists(name) {
		i++
		name = fmt.Sprintf("New Folder %d", i)
	}
	_ = sb.store.CreateFolder(name)
	sb.Rebuild()
	return sb, func() tea.Msg { return SidebarUpdatedMsg{} }
}

func (sb Sidebar) folderExists(name string) bool {
	for _, f := range sb.store.Folders() {
		if f.Name == name {
			return true
		}
	}
	return false
}

// --- Rename support ---

func (sb Sidebar) startRename() (Sidebar, tea.Cmd) {
	sel := sb.list.SelectedItem()
	if sel == nil {
		return sb, nil
	}
	si, ok := sel.(sidebarItem)
	if !ok || si.kind == kindSeparator {
		return sb, nil
	}

	sb.renaming = true
	sb.renameInput.SetValue(si.name)
	sb.renameInput.CursorEnd()
	sb.resetScrollState()
	sb.syncListSize()

	if si.kind == kindFolder {
		sb.renameIsFolder = true
		sb.renameID = si.name
	} else {
		sb.renameIsFolder = false
		sb.renameID = si.entryID
	}

	return sb, sb.renameInput.Focus()
}

func (sb Sidebar) updateRename(msg tea.KeyPressMsg) (Sidebar, tea.Cmd) {
	switch msg.String() {
	case "enter":
		newName := sb.renameInput.Value()
		if newName != "" {
			if sb.renameIsFolder {
				_ = sb.store.RenameFolder(sb.renameID, newName)
			} else {
				_ = sb.store.RenameEntry(sb.renameID, newName)
			}
		}
		sb.cancelRename()
		sb.Rebuild()
		return sb, func() tea.Msg { return SidebarUpdatedMsg{} }
	case "esc":
		sb.cancelRename()
		return sb, nil
	default:
		var cmd tea.Cmd
		sb.renameInput, cmd = sb.renameInput.Update(msg)
		return sb, cmd
	}
}

func (sb *Sidebar) cancelRename() {
	sb.renaming = false
	sb.renameInput.Blur()
	sb.resetScrollState()
	sb.syncListSize()
}

func (sb Sidebar) handleDelete() (Sidebar, tea.Cmd) {
	sel := sb.list.SelectedItem()
	if sel == nil {
		return sb, nil
	}
	si, ok := sel.(sidebarItem)
	if !ok {
		return sb, nil
	}

	switch si.kind {
	case kindFolder:
		_ = sb.store.DeleteFolder(si.name)
		sb.Rebuild()
	case kindEntry:
		_ = sb.store.DeleteEntry(si.entryID)
		sb.Rebuild()
	}
	return sb, func() tea.Msg { return SidebarUpdatedMsg{} }
}

// handleMove moves an entry to the next (dir=1) or previous (dir=-1) folder.
// It auto-opens target folders and re-closes folders that were auto-opened when
// the entry leaves them. The cursor follows the moved entry.
func (sb Sidebar) handleMove(dir int) (Sidebar, tea.Cmd) {
	sel := sb.list.SelectedItem()
	if sel == nil {
		return sb, nil
	}
	si, ok := sel.(sidebarItem)
	if !ok || si.kind != kindEntry {
		return sb, nil
	}

	folders := sb.store.Folders()
	if len(folders) == 0 {
		return sb, nil
	}

	// Determine current folder
	currentFolder := si.folder
	if currentFolder == "" {
		currentFolder = unsortedDir
	}

	// Build rotation targets: each folder name, then unsorted
	targets := make([]string, 0, len(folders)+1)
	for _, f := range folders {
		targets = append(targets, f.Name)
	}
	targets = append(targets, unsortedDir)

	// Find current index in targets
	currentIdx := 0
	for i, t := range targets {
		if t == currentFolder {
			currentIdx = i
			break
		}
	}

	// Move in direction, wrapping around
	nextIdx := (currentIdx + dir + len(targets)) % len(targets)
	nextTarget := targets[nextIdx]

	if nextTarget == currentFolder {
		return sb, nil
	}

	// Initialize auto-opened tracking if needed
	if sb.autoOpened == nil {
		sb.autoOpened = make(map[string]bool)
	}

	// If leaving a folder that was auto-opened, re-collapse it
	if currentFolder != unsortedDir && sb.autoOpened[currentFolder] {
		sb.store.SetCollapsed(currentFolder, true)
		delete(sb.autoOpened, currentFolder)
		_ = sb.store.Save()
	}

	// If target is a folder and it's collapsed, auto-open it
	if nextTarget != unsortedDir && sb.store.IsCollapsed(nextTarget) {
		sb.store.SetCollapsed(nextTarget, false)
		sb.autoOpened[nextTarget] = true
		_ = sb.store.Save()
	}

	entryID := si.entryID
	_ = sb.store.MoveEntry(entryID, nextTarget)
	sb.RebuildAndFollow(entryID)
	return sb, func() tea.Msg { return SidebarUpdatedMsg{} }
}

// --- Search support ---

func (sb Sidebar) startSearch() (Sidebar, tea.Cmd) {
	sb.searching = true
	sb.searchInput.SetValue("")
	sb.resetScrollState()
	sb.syncListSize()
	return sb, sb.searchInput.Focus()
}

func (sb Sidebar) updateSearch(msg tea.KeyPressMsg) (Sidebar, tea.Cmd) {
	switch msg.String() {
	case "esc":
		sb.cancelSearch()
		sb.Rebuild()
		return sb, nil
	case "enter":
		// Confirm search — exit search mode but keep the filtered view
		// Select the currently highlighted item
		sb.cancelSearch()
		return sb, nil
	case "down", "j":
		// Navigate within filtered results
		prevIdx := sb.list.Index()
		var cmd tea.Cmd
		sb.list, cmd = sb.list.Update(msg)
		newIdx := sb.list.Index()
		if newIdx != prevIdx {
			dir := 1
			if newIdx < prevIdx {
				dir = -1
			}
			sb.skipSeparator(dir)
		}
		return sb, cmd
	case "up", "k":
		prevIdx := sb.list.Index()
		var cmd tea.Cmd
		sb.list, cmd = sb.list.Update(msg)
		newIdx := sb.list.Index()
		if newIdx != prevIdx {
			dir := -1
			sb.skipSeparator(dir)
		}
		return sb, cmd
	default:
		var cmd tea.Cmd
		sb.searchInput, cmd = sb.searchInput.Update(msg)
		sb.rebuildFiltered(sb.searchInput.Value())
		return sb, cmd
	}
}

func (sb *Sidebar) cancelSearch() {
	sb.searching = false
	sb.searchInput.Blur()
	sb.searchInput.SetValue("")
	sb.resetScrollState()
	sb.syncListSize()
}

// rebuildFiltered builds items matching the search query.
// - If a folder name matches, show the folder (collapsed display, no entries)
// - If an entry name/query matches, show it under its folder header
//   (only matching entries shown, folder is displayed as expanded)
// - Unsorted matching entries shown under the Unsorted separator
func (sb *Sidebar) rebuildFiltered(query string) {
	query = strings.ToLower(query)
	if query == "" {
		sb.Rebuild()
		return
	}

	var items []list.Item
	folders := sb.store.Folders()
	unsorted := sb.store.Unsorted()

	for _, f := range folders {
		folderMatches := strings.Contains(strings.ToLower(f.Name), query)

		// Find matching entries in this folder
		var matchingEntries []sidebarItem
		for _, e := range f.Entries {
			if strings.Contains(strings.ToLower(e.Name), query) ||
				strings.Contains(strings.ToLower(e.Query), query) {
				matchingEntries = append(matchingEntries, sidebarItem{
					kind:     kindEntry,
					name:     e.Name,
					folder:   f.Name,
					entryID:  e.ID,
					endpoint: e.Endpoint,
				})
			}
		}

		if folderMatches {
			// Show folder itself
			items = append(items, sidebarItem{
				kind:      kindFolder,
				name:      f.Name,
				collapsed: len(matchingEntries) == 0,
			})
			// Also show any matching entries
			for _, me := range matchingEntries {
				items = append(items, me)
			}
		} else if len(matchingEntries) > 0 {
			// Folder doesn't match but has matching entries — show folder as context
			items = append(items, sidebarItem{
				kind:      kindFolder,
				name:      f.Name,
				collapsed: false,
			})
			for _, me := range matchingEntries {
				items = append(items, me)
			}
		}
	}

	// Matching unsorted entries
	var unsortedMatches []sidebarItem
	for _, e := range unsorted {
		if strings.Contains(strings.ToLower(e.Name), query) ||
			strings.Contains(strings.ToLower(e.Query), query) {
			unsortedMatches = append(unsortedMatches, sidebarItem{
				kind:     kindEntry,
				name:     e.Name,
				folder:   "",
				entryID:  e.ID,
				endpoint: e.Endpoint,
			})
		}
	}

	if len(unsortedMatches) > 0 && len(folders) > 0 {
		items = append(items, sidebarItem{
			kind: kindSeparator,
			name: "Unsorted",
		})
	}
	for _, me := range unsortedMatches {
		items = append(items, me)
	}

	sb.list.SetItems(items)
	if len(items) > 0 {
		sb.list.Select(0)
		sb.skipSeparator(1)
	}
}

// commitAutoOpened clears auto-opened tracking, keeping folders in their current state.
// Called when the user performs any action other than move.
func (sb *Sidebar) commitAutoOpened() {
	sb.autoOpened = nil
}

// View renders the sidebar.
func (sb Sidebar) View() string {
	if sb.renaming {
		prompt := lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("Rename:")
		input := sb.renameInput.View()
		return prompt + "\n" + input + "\n" + sb.list.View()
	}
	if sb.searching {
		return sb.searchInput.View() + "\n" + sb.list.View()
	}
	return sb.list.View()
}

// ItemCount returns the number of items in the list.
func (sb *Sidebar) ItemCount() int {
	return len(sb.list.Items())
}
