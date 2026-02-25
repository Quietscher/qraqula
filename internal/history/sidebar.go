package history

import (
	"fmt"
	"strings"
	"time"

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

type sectionKind int

const (
	sectionFolders sectionKind = iota
	sectionRecent
)

// Sidebar is the Bubble Tea model for the history sidebar.
type Sidebar struct {
	store  *Store
	width  int
	height int
	open   bool

	// Two-section item storage
	folderItems []sidebarItem
	recentItems []sidebarItem

	// Navigation
	activeSection sectionKind
	folderCursor  int
	folderScroll  int
	recentCursor  int
	recentScroll  int

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

	// Marquee scroll state
	scroll *scrollState
}

// NewSidebar creates a sidebar with the given store.
func NewSidebar(store *Store) Sidebar {
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
		open:        store.Meta.SidebarOpen,
		renameInput: ti,
		searchInput: si,
		scroll:      &scrollState{lastIdx: -1},
	}
	sb.Rebuild()
	return sb
}

// SetSize sets the viewport dimensions.
func (sb *Sidebar) SetSize(w, h int) {
	sb.width = w
	sb.height = h
	sb.renameInput.SetWidth(w)
	sb.searchInput.SetWidth(w)
}

// SetOpen sets sidebar visibility.
func (sb *Sidebar) SetOpen(open bool) { sb.open = open }

// IsOpen returns whether the sidebar is visible.
func (sb *Sidebar) IsOpen() bool { return sb.open }

// Focus is a no-op for now.
func (sb *Sidebar) Focus() {}

// Blur exits filter mode and rename mode when the sidebar loses focus.
func (sb *Sidebar) Blur() {
	sb.cancelSearch()
	sb.cancelRename()
	sb.commitAutoOpened()
}

// ItemCount returns the total number of items across both sections.
func (sb *Sidebar) ItemCount() int {
	return len(sb.folderItems) + len(sb.recentItems)
}

// Rebuild rebuilds both sections from store data, preserving cursor position.
func (sb *Sidebar) Rebuild() {
	sb.rebuildSections()
	sb.clampCursors()
	sb.ensureFolderVisible()
	sb.ensureRecentVisible()
	sb.resetScrollState()
}

// RebuildAndFollow rebuilds and selects the entry with the given ID.
func (sb *Sidebar) RebuildAndFollow(entryID string) {
	sb.rebuildSections()
	// Find entry in sections
	for i, item := range sb.folderItems {
		if item.entryID == entryID {
			sb.activeSection = sectionFolders
			sb.folderCursor = i
			sb.ensureFolderVisible()
			sb.resetScrollState()
			return
		}
	}
	for i, item := range sb.recentItems {
		if item.entryID == entryID {
			sb.activeSection = sectionRecent
			sb.recentCursor = i
			sb.ensureRecentVisible()
			sb.resetScrollState()
			return
		}
	}
	sb.clampCursors()
	sb.resetScrollState()
}

func (sb *Sidebar) rebuildSections() {
	sb.folderItems = nil
	sb.recentItems = nil

	folders := sb.store.Folders()
	unsorted := sb.store.Unsorted()

	for _, f := range folders {
		collapsed := sb.store.IsCollapsed(f.Name)
		sb.folderItems = append(sb.folderItems, sidebarItem{
			kind:      kindFolder,
			name:      f.Name,
			collapsed: collapsed,
		})
		if !collapsed {
			for _, e := range f.Entries {
				sb.folderItems = append(sb.folderItems, sidebarItem{
					kind:     kindEntry,
					name:     e.Name,
					folder:   f.Name,
					entryID:  e.ID,
					endpoint: e.Endpoint,
				})
			}
		}
	}

	for _, e := range unsorted {
		sb.recentItems = append(sb.recentItems, sidebarItem{
			kind:     kindEntry,
			name:     e.Name,
			folder:   "",
			entryID:  e.ID,
			endpoint: e.Endpoint,
		})
	}

	// If one section is empty, switch to the other
	if len(sb.folderItems) == 0 && len(sb.recentItems) > 0 {
		sb.activeSection = sectionRecent
	} else if len(sb.recentItems) == 0 && len(sb.folderItems) > 0 {
		sb.activeSection = sectionFolders
	}
}

func (sb *Sidebar) clampCursors() {
	if sb.folderCursor >= len(sb.folderItems) {
		sb.folderCursor = len(sb.folderItems) - 1
	}
	if sb.folderCursor < 0 {
		sb.folderCursor = 0
	}
	if sb.recentCursor >= len(sb.recentItems) {
		sb.recentCursor = len(sb.recentItems) - 1
	}
	if sb.recentCursor < 0 {
		sb.recentCursor = 0
	}
}

// selectedItem returns the currently selected item, or nil.
func (sb *Sidebar) selectedItem() *sidebarItem {
	switch sb.activeSection {
	case sectionFolders:
		if sb.folderCursor >= 0 && sb.folderCursor < len(sb.folderItems) {
			return &sb.folderItems[sb.folderCursor]
		}
	case sectionRecent:
		if sb.recentCursor >= 0 && sb.recentCursor < len(sb.recentItems) {
			return &sb.recentItems[sb.recentCursor]
		}
	}
	return nil
}

// SelectedEntry returns the currently selected entry, or nil if none.
func (sb *Sidebar) SelectedEntry() *Entry {
	si := sb.selectedItem()
	if si == nil || si.kind != kindEntry {
		return nil
	}
	return sb.findEntry(si.entryID)
}

// SelectedFolder returns the folder name if a folder is selected.
func (sb *Sidebar) SelectedFolder() string {
	si := sb.selectedItem()
	if si == nil || si.kind != kindFolder {
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

// --- Section height calculation ---

func (sb Sidebar) sectionHeights() (foldersH, recentH int) {
	h := sb.height
	if sb.renaming {
		h -= 2
	}
	if sb.searching {
		h--
	}
	if h < 2 {
		h = 2
	}

	hasFolders := len(sb.folderItems) > 0
	hasRecent := len(sb.recentItems) > 0

	if hasFolders && hasRecent {
		sepH := 1
		available := h - sepH
		if available < 2 {
			available = 2
		}

		fCount := len(sb.folderItems)
		rCount := len(sb.recentItems)

		if fCount+rCount <= available {
			// Everything fits — give each section its item count
			foldersH = fCount
			recentH = rCount
			extra := available - foldersH - recentH
			if fCount >= rCount {
				foldersH += extra
			} else {
				recentH += extra
			}
		} else {
			// Need scrolling — allocate proportionally
			total := fCount + rCount
			foldersH = available * fCount / total
			if foldersH < 1 {
				foldersH = 1
			}
			recentH = available - foldersH
			if recentH < 1 {
				recentH = 1
				foldersH = available - 1
			}
		}
		return
	}
	if hasFolders {
		return h, 0
	}
	return 0, h
}

// --- Scroll / cursor management ---

func (sb *Sidebar) ensureFolderVisible() {
	fH, _ := sb.sectionHeights()
	ensureVisible(&sb.folderCursor, &sb.folderScroll, fH, len(sb.folderItems))
}

func (sb *Sidebar) ensureRecentVisible() {
	_, rH := sb.sectionHeights()
	ensureVisible(&sb.recentCursor, &sb.recentScroll, rH, len(sb.recentItems))
}

func ensureVisible(cursor, scroll *int, height, total int) {
	if height <= 0 {
		return
	}
	if *cursor < *scroll {
		*scroll = *cursor
	}
	if *cursor >= *scroll+height {
		*scroll = *cursor - height + 1
	}
	// Clamp scroll
	maxScroll := total - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if *scroll > maxScroll {
		*scroll = maxScroll
	}
	if *scroll < 0 {
		*scroll = 0
	}
}

func (sb *Sidebar) moveDown() {
	switch sb.activeSection {
	case sectionFolders:
		if sb.folderCursor < len(sb.folderItems)-1 {
			sb.folderCursor++
			sb.ensureFolderVisible()
		} else if len(sb.recentItems) > 0 {
			sb.activeSection = sectionRecent
			sb.recentCursor = 0
			sb.ensureRecentVisible()
		}
	case sectionRecent:
		if sb.recentCursor < len(sb.recentItems)-1 {
			sb.recentCursor++
			sb.ensureRecentVisible()
		}
	}
}

func (sb *Sidebar) moveUp() {
	switch sb.activeSection {
	case sectionRecent:
		if sb.recentCursor > 0 {
			sb.recentCursor--
			sb.ensureRecentVisible()
		} else if len(sb.folderItems) > 0 {
			sb.activeSection = sectionFolders
			sb.folderCursor = len(sb.folderItems) - 1
			sb.ensureFolderVisible()
		}
	case sectionFolders:
		if sb.folderCursor > 0 {
			sb.folderCursor--
			sb.ensureFolderVisible()
		}
	}
}

// --- Marquee scroll ---

func (sb *Sidebar) nameMaxForItem(si sidebarItem) int {
	width := sb.width
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

func (sb *Sidebar) resetScrollState() {
	if sb.scroll != nil {
		sb.scroll.offset = 0
		sb.scroll.active = false
		sb.scroll.paused = false
	}
}

func (sb *Sidebar) resetAndScheduleScroll() tea.Cmd {
	sb.resetScrollState()
	if sb.scroll != nil {
		sb.scroll.lastIdx = sb.cursorID()
	}
	return tea.Tick(scrollDelay, func(time.Time) tea.Msg { return scrollTickMsg{} })
}

// cursorID returns a unique int identifying the current selection for stale tick detection.
func (sb *Sidebar) cursorID() int {
	if sb.activeSection == sectionRecent {
		return len(sb.folderItems) + sb.recentCursor
	}
	return sb.folderCursor
}

func (sb Sidebar) handleScrollTick() (Sidebar, tea.Cmd) {
	if sb.scroll == nil || sb.searching || sb.renaming || sb.scroll.paused {
		return sb, nil
	}
	if sb.scroll.lastIdx != sb.cursorID() {
		return sb, nil
	}
	si := sb.selectedItem()
	if si == nil {
		return sb, nil
	}

	nameMax := sb.nameMaxForItem(*si)
	if !needsScroll(si.name, nameMax) {
		sb.scroll.active = false
		return sb, nil
	}

	sb.scroll.active = true
	runes := []rune(si.name)
	sb.scroll.offset++
	if sb.scroll.offset >= len(runes) {
		sb.scroll.offset = 0
		return sb, tea.Tick(scrollDelay, func(time.Time) tea.Msg { return scrollTickMsg{} })
	}
	return sb, tea.Tick(scrollInterval, func(time.Time) tea.Msg { return scrollTickMsg{} })
}

func (sb *Sidebar) handleManualScroll(key string) bool {
	if sb.scroll == nil {
		return false
	}
	si := sb.selectedItem()
	if si == nil {
		return false
	}

	nameMax := sb.nameMaxForItem(*si)
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

// --- Update ---

func (sb Sidebar) Update(msg tea.Msg) (Sidebar, tea.Cmd) {
	switch msg := msg.(type) {
	case scrollTickMsg:
		return sb.handleScrollTick()
	case tea.KeyPressMsg:
		if sb.renaming {
			return sb.updateRename(msg)
		}
		if sb.searching {
			return sb.updateSearch(msg)
		}

		switch msg.String() {
		case "m":
			return sb.handleMove(1)
		case "M":
			return sb.handleMove(-1)
		default:
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
		case "j", "down":
			sb.moveDown()
			return sb, sb.resetAndScheduleScroll()
		case "k", "up":
			sb.moveUp()
			return sb, sb.resetAndScheduleScroll()
		}
	}
	return sb, nil
}

func (sb Sidebar) handleSelect() (Sidebar, tea.Cmd) {
	si := sb.selectedItem()
	if si == nil {
		return sb, nil
	}

	switch si.kind {
	case kindFolder:
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
	si := sb.selectedItem()
	if si == nil {
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
	si := sb.selectedItem()
	if si == nil {
		return sb, nil
	}

	sb.renaming = true
	sb.renameInput.SetValue(si.name)
	sb.renameInput.CursorEnd()
	sb.resetScrollState()

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
}

// --- Delete ---

func (sb Sidebar) handleDelete() (Sidebar, tea.Cmd) {
	si := sb.selectedItem()
	if si == nil {
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

// --- Move ---

func (sb Sidebar) handleMove(dir int) (Sidebar, tea.Cmd) {
	si := sb.selectedItem()
	if si == nil || si.kind != kindEntry {
		return sb, nil
	}

	folders := sb.store.Folders()
	if len(folders) == 0 {
		return sb, nil
	}

	currentFolder := si.folder
	if currentFolder == "" {
		currentFolder = unsortedDir
	}

	targets := make([]string, 0, len(folders)+1)
	for _, f := range folders {
		targets = append(targets, f.Name)
	}
	targets = append(targets, unsortedDir)

	currentIdx := 0
	for i, t := range targets {
		if t == currentFolder {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + dir + len(targets)) % len(targets)
	nextTarget := targets[nextIdx]

	if nextTarget == currentFolder {
		return sb, nil
	}

	if sb.autoOpened == nil {
		sb.autoOpened = make(map[string]bool)
	}

	if currentFolder != unsortedDir && sb.autoOpened[currentFolder] {
		sb.store.SetCollapsed(currentFolder, true)
		delete(sb.autoOpened, currentFolder)
		_ = sb.store.Save()
	}

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
	return sb, sb.searchInput.Focus()
}

func (sb Sidebar) updateSearch(msg tea.KeyPressMsg) (Sidebar, tea.Cmd) {
	switch msg.String() {
	case "esc":
		sb.cancelSearch()
		sb.Rebuild()
		return sb, nil
	case "enter":
		sb.cancelSearch()
		return sb, nil
	case "down", "j":
		sb.moveDown()
		return sb, nil
	case "up", "k":
		sb.moveUp()
		return sb, nil
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
}

func (sb *Sidebar) rebuildFiltered(query string) {
	query = strings.ToLower(query)
	if query == "" {
		sb.Rebuild()
		return
	}

	sb.folderItems = nil
	sb.recentItems = nil

	folders := sb.store.Folders()
	unsorted := sb.store.Unsorted()

	for _, f := range folders {
		folderMatches := strings.Contains(strings.ToLower(f.Name), query)

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
			sb.folderItems = append(sb.folderItems, sidebarItem{
				kind:      kindFolder,
				name:      f.Name,
				collapsed: len(matchingEntries) == 0,
			})
			sb.folderItems = append(sb.folderItems, matchingEntries...)
		} else if len(matchingEntries) > 0 {
			sb.folderItems = append(sb.folderItems, sidebarItem{
				kind:      kindFolder,
				name:      f.Name,
				collapsed: false,
			})
			sb.folderItems = append(sb.folderItems, matchingEntries...)
		}
	}

	for _, e := range unsorted {
		if strings.Contains(strings.ToLower(e.Name), query) ||
			strings.Contains(strings.ToLower(e.Query), query) {
			sb.recentItems = append(sb.recentItems, sidebarItem{
				kind:     kindEntry,
				name:     e.Name,
				folder:   "",
				entryID:  e.ID,
				endpoint: e.Endpoint,
			})
		}
	}

	// Reset cursors for filtered view
	sb.folderCursor = 0
	sb.recentCursor = 0
	sb.folderScroll = 0
	sb.recentScroll = 0

	if len(sb.folderItems) > 0 {
		sb.activeSection = sectionFolders
	} else if len(sb.recentItems) > 0 {
		sb.activeSection = sectionRecent
	}
}

func (sb *Sidebar) commitAutoOpened() {
	sb.autoOpened = nil
}

// --- View ---

func (sb Sidebar) View() string {
	var parts []string

	if sb.renaming {
		prompt := lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("Rename:")
		input := sb.renameInput.View()
		parts = append(parts, prompt, input)
	}
	if sb.searching {
		parts = append(parts, sb.searchInput.View())
	}

	fH, rH := sb.sectionHeights()

	if fH > 0 && len(sb.folderItems) > 0 {
		parts = append(parts, sb.renderSection(sb.folderItems, sb.folderCursor, sb.folderScroll, fH, sb.activeSection == sectionFolders))
	} else if fH > 0 && len(sb.recentItems) > 0 {
		// No folder items — give space to recent
		rH += fH
		fH = 0
	}

	if fH > 0 && rH > 0 && len(sb.folderItems) > 0 && len(sb.recentItems) > 0 {
		parts = append(parts, sb.renderSectionSep())
	}

	if rH > 0 && len(sb.recentItems) > 0 {
		parts = append(parts, sb.renderSection(sb.recentItems, sb.recentCursor, sb.recentScroll, rH, sb.activeSection == sectionRecent))
	} else if rH > 0 && len(sb.folderItems) > 0 && fH == 0 {
		// Folders took full space already
	}

	// Handle empty state
	if len(sb.folderItems) == 0 && len(sb.recentItems) == 0 {
		empty := lipgloss.NewStyle().Faint(true).Padding(0, 1).Render("No history yet.")
		parts = append(parts, empty)
	}

	return strings.Join(parts, "\n")
}

func (sb Sidebar) renderSection(items []sidebarItem, cursor, scroll, height int, active bool) string {
	if height <= 0 {
		return ""
	}

	width := sb.width
	if width < 4 {
		width = 4
	}

	var lines []string
	end := scroll + height
	if end > len(items) {
		end = len(items)
	}
	for i := scroll; i < end; i++ {
		isSelected := active && i == cursor
		scrollOffset := 0
		if isSelected && sb.scroll != nil {
			scrollOffset = sb.scroll.offset
		}
		line := sb.renderItem(items[i], isSelected, width, scrollOffset)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (sb Sidebar) renderItem(si sidebarItem, selected bool, width, scrollOffset int) string {
	var line string
	switch si.kind {
	case kindFolder:
		line = renderFolderLine(si, selected, width, scrollOffset)
	case kindEntry:
		line = renderEntryLine(si, selected, width, scrollOffset)
	}
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line)
}

func (sb Sidebar) renderSectionSep() string {
	width := sb.width
	if width < 4 {
		width = 4
	}
	label := " Recent "
	labelWidth := len(label)
	remaining := width - labelWidth
	if remaining < 2 {
		return lipgloss.NewStyle().MaxWidth(width).Render(hSepLabel.Render("Recent"))
	}
	left := remaining / 3
	right := remaining - left
	line := hSepLine.Render(strings.Repeat("-", left)) +
		hSepLabel.Render(label) +
		hSepLine.Render(strings.Repeat("-", right))
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

// --- Test helpers ---

// SelectInSection selects an item in a specific section (for testing).
func (sb *Sidebar) SelectInSection(section sectionKind, index int) {
	sb.activeSection = section
	switch section {
	case sectionFolders:
		sb.folderCursor = index
	case sectionRecent:
		sb.recentCursor = index
	}
}
