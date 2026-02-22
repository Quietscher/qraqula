package history

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()
	return s
}

func TestNewSidebarEmptyStore(t *testing.T) {
	store := testStore(t)
	sb := NewSidebar(store)

	if sb.ItemCount() != 0 {
		t.Errorf("expected 0 items for empty store, got %d", sb.ItemCount())
	}
}

func TestSidebarRebuildWithUnsortedOnly(t *testing.T) {
	store := testStore(t)

	e1 := Entry{ID: GenerateID(), Name: "GetUsers", Query: "{ users }", CreatedAt: time.Now()}
	e2 := Entry{ID: GenerateID(), Name: "GetPosts", Query: "{ posts }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.AddEntry(e2)

	sb := NewSidebar(store)
	// No folders, so no separator — just 2 entries
	if sb.ItemCount() != 2 {
		t.Errorf("expected 2 items (2 entries, no separator), got %d", sb.ItemCount())
	}
}

func TestSidebarRebuildWithFoldersAndUnsorted(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("API")
	e1 := Entry{ID: GenerateID(), Name: "GetUsers", Query: "{ users }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.MoveEntry(e1.ID, "API")

	e2 := Entry{ID: GenerateID(), Name: "GetPosts", Query: "{ posts }", CreatedAt: time.Now()}
	_ = store.AddEntry(e2)

	sb := NewSidebar(store)
	// folder + folder entry + separator + unsorted entry = 4
	if sb.ItemCount() != 4 {
		t.Errorf("expected 4 items (folder + entry + sep + unsorted), got %d", sb.ItemCount())
	}
}

func TestSidebarNoSeparatorWhenNoUnsorted(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("API")
	e1 := Entry{ID: GenerateID(), Name: "GetUsers", Query: "{ users }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.MoveEntry(e1.ID, "API")

	sb := NewSidebar(store)
	// folder + entry = 2, NO separator because no unsorted entries
	if sb.ItemCount() != 2 {
		t.Errorf("expected 2 items (folder + entry, no separator), got %d", sb.ItemCount())
	}
}

func TestSidebarFolderCollapseChangesItemCount(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("API")
	e1 := Entry{ID: GenerateID(), Name: "GetUsers", Query: "{ users }", CreatedAt: time.Now()}
	e2 := Entry{ID: GenerateID(), Name: "GetPosts", Query: "{ posts }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.AddEntry(e2)
	_ = store.MoveEntry(e1.ID, "API")
	_ = store.MoveEntry(e2.ID, "API")

	sb := NewSidebar(store)
	// folder + 2 entries = 3 (no separator, no unsorted)
	expandedCount := sb.ItemCount()
	if expandedCount != 3 {
		t.Errorf("expected 3 items expanded, got %d", expandedCount)
	}

	store.SetCollapsed("API", true)
	sb.Rebuild()
	collapsedCount := sb.ItemCount()

	// folder only = 1
	if collapsedCount != 1 {
		t.Errorf("expected 1 item when collapsed, got %d", collapsedCount)
	}
}

func TestSidebarSelectedEntry(t *testing.T) {
	store := testStore(t)

	e := Entry{ID: GenerateID(), Name: "GetUsers", Query: "{ users }", Endpoint: "http://localhost/graphql", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Only entry at index 0 (no separator since no folders)
	sb.list.Select(0)

	selected := sb.SelectedEntry()
	if selected == nil {
		t.Fatal("expected selected entry")
	}
	if selected.ID != e.ID {
		t.Errorf("expected entry ID %s, got %s", e.ID, selected.ID)
	}
}

func TestSidebarSelectedFolder(t *testing.T) {
	store := testStore(t)
	_ = store.CreateFolder("TestFolder")

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	sb.list.Select(0)

	folder := sb.SelectedFolder()
	if folder != "TestFolder" {
		t.Errorf("expected folder 'TestFolder', got %q", folder)
	}
}

func TestSidebarView(t *testing.T) {
	store := testStore(t)
	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	view := sb.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestSidebarOpenState(t *testing.T) {
	store := testStore(t)
	sb := NewSidebar(store)

	if !sb.IsOpen() {
		t.Error("expected sidebar open by default")
	}
	sb.SetOpen(false)
	if sb.IsOpen() {
		t.Error("expected sidebar closed after SetOpen(false)")
	}
}

func TestSidebarCursorSkipsSeparator(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("API")
	e1 := Entry{ID: GenerateID(), Name: "FolderEntry", Query: "{ a }", CreatedAt: time.Now()}
	e2 := Entry{ID: GenerateID(), Name: "UnsortedEntry", Query: "{ b }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.AddEntry(e2)
	_ = store.MoveEntry(e1.ID, "API")

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Items: folder(0), entry(1), separator(2), unsorted entry(3)
	if sb.ItemCount() != 4 {
		t.Fatalf("expected 4 items, got %d", sb.ItemCount())
	}

	// Select the folder entry (index 1), then navigate down
	sb.list.Select(1)
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})

	// Should skip separator (index 2) and land on unsorted entry (index 3)
	idx := sb.list.Index()
	if idx == 2 {
		t.Error("cursor should skip separator item")
	}
	if idx != 3 {
		t.Errorf("expected cursor at index 3 (unsorted entry), got %d", idx)
	}
}

func TestSidebarRenameEntry(t *testing.T) {
	store := testStore(t)

	e := Entry{ID: GenerateID(), Name: "OldName", Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)
	sb.list.Select(0)

	// Press 'r' to enter rename mode
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if !sb.renaming {
		t.Fatal("expected rename mode to be active")
	}
	if sb.renameInput.Value() != "OldName" {
		t.Errorf("expected rename input pre-filled with 'OldName', got %q", sb.renameInput.Value())
	}

	// Clear and type new name
	sb.renameInput.SetValue("NewName")

	// Press enter to confirm
	sb, _ = sb.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if sb.renaming {
		t.Error("expected rename mode to exit after enter")
	}

	// Verify entry was renamed
	if store.unsorted[0].Name != "NewName" {
		t.Errorf("expected entry renamed to 'NewName', got %q", store.unsorted[0].Name)
	}
}

func TestSidebarRenameFolder(t *testing.T) {
	store := testStore(t)
	_ = store.CreateFolder("OldFolder")

	sb := NewSidebar(store)
	sb.SetSize(40, 20)
	sb.list.Select(0)

	// Press 'r' to enter rename mode
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if !sb.renaming {
		t.Fatal("expected rename mode to be active")
	}

	sb.renameInput.SetValue("RenamedFolder")
	sb, _ = sb.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if sb.renaming {
		t.Error("expected rename mode to exit")
	}
	if store.folders[0].Name != "RenamedFolder" {
		t.Errorf("expected folder renamed to 'RenamedFolder', got %q", store.folders[0].Name)
	}
}

func TestSidebarRenameCancelWithEsc(t *testing.T) {
	store := testStore(t)

	e := Entry{ID: GenerateID(), Name: "OrigName", Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)
	sb.list.Select(0)

	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	sb.renameInput.SetValue("Changed")
	sb, _ = sb.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if sb.renaming {
		t.Error("expected rename mode to exit on esc")
	}
	if store.unsorted[0].Name != "OrigName" {
		t.Errorf("expected name unchanged after esc, got %q", store.unsorted[0].Name)
	}
}

func TestSidebarMoveCursorFollowsEntry(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("A")
	_ = store.CreateFolder("B")
	e := Entry{ID: GenerateID(), Name: "MyQuery", Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e) // goes to unsorted

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Items: folder A(0), folder B(1), separator(2), unsorted entry(3)
	if sb.ItemCount() != 4 {
		t.Fatalf("expected 4 items, got %d", sb.ItemCount())
	}

	// Select the unsorted entry
	sb.list.Select(3)
	sel := sb.list.SelectedItem().(sidebarItem)
	if sel.entryID != e.ID {
		t.Fatalf("expected selected entry %s, got %s", e.ID, sel.entryID)
	}

	// Move to folder A (m moves forward: unsorted → A wraps around)
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})

	// Cursor should now be on the entry inside folder A
	newSel := sb.list.SelectedItem()
	if newSel == nil {
		t.Fatal("expected selected item after move")
	}
	si := newSel.(sidebarItem)
	if si.entryID != e.ID {
		t.Errorf("expected cursor to follow entry %s, got %s", e.ID, si.entryID)
	}
	if si.folder != "A" {
		t.Errorf("expected entry in folder A, got %q", si.folder)
	}
}

func TestSidebarMoveAutoOpensCollapsedFolder(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("Target")
	store.SetCollapsed("Target", true)
	_ = store.Save()

	e := Entry{ID: GenerateID(), Name: "MyQuery", Query: "{ q }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Items: collapsed folder Target(0), separator(1), unsorted entry(2)
	// Select unsorted entry
	sb.list.Select(2)

	// Move to Target — should auto-open it
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})

	// Folder should now be expanded
	if store.IsCollapsed("Target") {
		t.Error("expected Target folder to be auto-opened after move")
	}

	// Cursor should be on the entry inside Target
	si := sb.list.SelectedItem().(sidebarItem)
	if si.entryID != e.ID {
		t.Errorf("expected cursor on moved entry, got %s", si.entryID)
	}
}

func TestSidebarMoveAutoClosesOnLeave(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("A")
	_ = store.CreateFolder("B")
	store.SetCollapsed("A", true)
	_ = store.Save()

	e := Entry{ID: GenerateID(), Name: "MyQuery", Query: "{ q }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Select unsorted entry (last item)
	items := sb.list.Items()
	sb.list.Select(len(items) - 1)

	// Move into folder A (auto-opens it)
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if store.IsCollapsed("A") {
		t.Fatal("expected A to be auto-opened")
	}

	// Move again out of A into B
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})

	// A should be re-collapsed since it was auto-opened
	if !store.IsCollapsed("A") {
		t.Error("expected A to be re-collapsed after entry moved out")
	}

	// Entry should now be in B
	si := sb.list.SelectedItem().(sidebarItem)
	if si.folder != "B" {
		t.Errorf("expected entry in folder B, got %q", si.folder)
	}
}

func TestSidebarMoveReverseWithShiftM(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("A")
	_ = store.CreateFolder("B")
	e := Entry{ID: GenerateID(), Name: "MyQuery", Query: "{ q }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)
	_ = store.MoveEntry(e.ID, "B")

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Find and select entry in B
	for i, item := range sb.list.Items() {
		if si, ok := item.(sidebarItem); ok && si.entryID == e.ID {
			sb.list.Select(i)
			break
		}
	}

	// Shift+M should move backward: B → A
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'M', Text: "M"})

	si := sb.list.SelectedItem().(sidebarItem)
	if si.folder != "A" {
		t.Errorf("expected entry moved to A with Shift+M, got %q", si.folder)
	}
	if si.entryID != e.ID {
		t.Errorf("expected cursor to follow entry")
	}
}

func TestSidebarSearchFindsEntryInCollapsedFolder(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("API")
	e := Entry{ID: GenerateID(), Name: "GetUsers", Query: "{ users { name } }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)
	_ = store.MoveEntry(e.ID, "API")
	store.SetCollapsed("API", true)
	_ = store.Save()

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Collapsed folder — only 1 item visible (the folder)
	if sb.ItemCount() != 1 {
		t.Fatalf("expected 1 item (collapsed folder), got %d", sb.ItemCount())
	}

	// Start search with /
	sb, _ = sb.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if !sb.searching {
		t.Fatal("expected search mode active")
	}

	// Type "get" — should find the entry inside the collapsed folder
	for _, ch := range "get" {
		sb, _ = sb.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}

	// Should show folder + matching entry = 2 items
	if sb.ItemCount() != 2 {
		t.Errorf("expected 2 items (folder + entry), got %d", sb.ItemCount())
	}

	// The entry should be selectable
	found := false
	for _, item := range sb.list.Items() {
		if si, ok := item.(sidebarItem); ok && si.entryID == e.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected matching entry in filtered results")
	}
}

func TestSidebarSearchShowsFolderWhenNameMatches(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("MyAPI")
	e := Entry{ID: GenerateID(), Name: "Unrelated", Query: "{ x }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)
	_ = store.MoveEntry(e.ID, "MyAPI")

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Start search
	sb, _ = sb.Update(tea.KeyPressMsg{Code: '/', Text: "/"})

	// Type "myapi" — folder name matches
	for _, ch := range "myapi" {
		sb, _ = sb.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}

	// Should show the folder (name matches)
	if sb.ItemCount() < 1 {
		t.Error("expected at least folder in results")
	}
	first := sb.list.Items()[0].(sidebarItem)
	if first.kind != kindFolder || first.name != "MyAPI" {
		t.Errorf("expected folder MyAPI as first result, got %+v", first)
	}
}

func TestSidebarSearchMatchesQuery(t *testing.T) {
	store := testStore(t)

	e := Entry{ID: GenerateID(), Name: "ShortName", Query: "query GetCountries { countries { name } }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	sb, _ = sb.Update(tea.KeyPressMsg{Code: '/', Text: "/"})

	// Search by query content
	for _, ch := range "countries" {
		sb, _ = sb.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}

	if sb.ItemCount() != 1 {
		t.Errorf("expected 1 matching entry, got %d", sb.ItemCount())
	}
}

func TestSidebarSearchEscRestoresNormalView(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("API")
	e1 := Entry{ID: GenerateID(), Name: "GetUsers", Query: "{ users }", CreatedAt: time.Now()}
	e2 := Entry{ID: GenerateID(), Name: "GetPosts", Query: "{ posts }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.AddEntry(e2)
	_ = store.MoveEntry(e1.ID, "API")

	sb := NewSidebar(store)
	sb.SetSize(40, 20)
	normalCount := sb.ItemCount()

	// Search to get filtered results
	sb, _ = sb.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, ch := range "users" {
		sb, _ = sb.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}

	// Esc to exit search
	sb, _ = sb.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if sb.searching {
		t.Error("expected search mode to exit on esc")
	}
	if sb.ItemCount() != normalCount {
		t.Errorf("expected normal item count %d after esc, got %d", normalCount, sb.ItemCount())
	}
}

func TestSidebarSearchOnlyShowsMatchingEntriesInFolder(t *testing.T) {
	store := testStore(t)

	_ = store.CreateFolder("API")
	e1 := Entry{ID: GenerateID(), Name: "GetUsers", Query: "{ users }", CreatedAt: time.Now()}
	e2 := Entry{ID: GenerateID(), Name: "GetPosts", Query: "{ posts }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.AddEntry(e2)
	_ = store.MoveEntry(e1.ID, "API")
	_ = store.MoveEntry(e2.ID, "API")

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	sb, _ = sb.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, ch := range "users" {
		sb, _ = sb.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}

	// Should show folder + only GetUsers, not GetPosts
	if sb.ItemCount() != 2 {
		t.Errorf("expected 2 items (folder + matching entry), got %d", sb.ItemCount())
	}

	// Verify only the matching entry is shown
	for _, item := range sb.list.Items() {
		si := item.(sidebarItem)
		if si.kind == kindEntry && si.name == "GetPosts" {
			t.Error("non-matching entry GetPosts should not appear in results")
		}
	}
}

func TestSidebarRenameViewShowsInput(t *testing.T) {
	store := testStore(t)

	e := Entry{ID: GenerateID(), Name: "TestEntry", Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)
	sb.list.Select(0)

	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	view := sb.View()
	if view == "" {
		t.Error("expected non-empty view during rename")
	}
}

func TestSidebarScrollStateShared(t *testing.T) {
	store := testStore(t)
	e := Entry{ID: GenerateID(), Name: "TestEntry", Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// scroll pointer should be shared between sidebar and delegate
	if sb.scroll == nil {
		t.Fatal("expected scroll state to be set")
	}
}

func TestSidebarScrollResetsOnRebuild(t *testing.T) {
	store := testStore(t)
	e := Entry{ID: GenerateID(), Name: "VeryLongEntryNameThatExceedsWidth", Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(25, 20) // narrow to force truncation

	// Manually set scroll offset
	sb.scroll.offset = 5
	sb.scroll.active = true

	// Rebuild should reset
	sb.Rebuild()
	if sb.scroll.offset != 0 {
		t.Errorf("expected scroll offset reset to 0, got %d", sb.scroll.offset)
	}
	if sb.scroll.active {
		t.Error("expected scroll active to be false after rebuild")
	}
}

func TestSidebarScrollResetsOnSearch(t *testing.T) {
	store := testStore(t)
	e := Entry{ID: GenerateID(), Name: "TestEntry", Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)
	sb.scroll.offset = 3
	sb.scroll.active = true

	// Start search
	sb, _ = sb.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if sb.scroll.offset != 0 {
		t.Errorf("expected scroll offset reset on search start, got %d", sb.scroll.offset)
	}
	if sb.scroll.active {
		t.Error("expected scroll inactive during search")
	}
}

func TestSidebarScrollTickAdvancesOffset(t *testing.T) {
	store := testStore(t)
	// Create an entry with a very long name to ensure it needs scrolling
	longName := "ThisIsAVeryLongEntryNameThatDefinitelyExceedsTheAvailableWidth"
	e := Entry{ID: GenerateID(), Name: longName, Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(25, 20) // narrow width

	// Set up scroll state as if the delay has passed
	sb.scroll.lastIdx = sb.list.Index()

	// Send a scroll tick
	sb, cmd := sb.Update(scrollTickMsg{})

	// Should have advanced offset
	if sb.scroll.offset != 1 {
		t.Errorf("expected scroll offset 1 after tick, got %d", sb.scroll.offset)
	}
	if !sb.scroll.active {
		t.Error("expected scroll active after tick on truncated item")
	}
	if cmd == nil {
		t.Error("expected next tick to be scheduled")
	}
}

func TestSidebarScrollTickIgnoredDuringSearch(t *testing.T) {
	store := testStore(t)
	e := Entry{ID: GenerateID(), Name: "VeryLongNameThatNeedsScrolling", Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(25, 20)
	sb.scroll.lastIdx = sb.list.Index()
	sb.searching = true

	// Send a scroll tick — should be ignored during search
	sb, cmd := sb.Update(scrollTickMsg{})
	if sb.scroll.offset != 0 {
		t.Errorf("expected scroll offset unchanged during search, got %d", sb.scroll.offset)
	}
	if cmd != nil {
		t.Error("expected no next tick during search")
	}
}

func TestSidebarManualScrollRight(t *testing.T) {
	store := testStore(t)
	longName := "ThisIsAVeryLongEntryNameThatDefinitelyExceedsTheAvailableWidth"
	e := Entry{ID: GenerateID(), Name: longName, Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(25, 20)

	// Press right arrow to scroll manually
	sb, _ = sb.Update(tea.KeyPressMsg{Code: tea.KeyRight, Text: "right"})

	if sb.scroll.offset != 1 {
		t.Errorf("expected scroll offset 1 after right arrow, got %d", sb.scroll.offset)
	}
}

func TestSidebarManualScrollLeft(t *testing.T) {
	store := testStore(t)
	longName := "ThisIsAVeryLongEntryNameThatDefinitelyExceedsTheAvailableWidth"
	e := Entry{ID: GenerateID(), Name: longName, Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(25, 20)

	// First scroll right, then left
	sb.scroll.offset = 5
	sb.scroll.active = true

	sb, _ = sb.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Text: "left"})

	if sb.scroll.offset != 4 {
		t.Errorf("expected scroll offset 4 after left arrow, got %d", sb.scroll.offset)
	}
}

func TestSidebarManualScrollLeftStopsAtZero(t *testing.T) {
	store := testStore(t)
	longName := "ThisIsAVeryLongEntryNameThatDefinitelyExceedsTheAvailableWidth"
	e := Entry{ID: GenerateID(), Name: longName, Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(25, 20)

	// Already at 0, pressing left should stay at 0
	sb, _ = sb.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Text: "left"})

	if sb.scroll.offset != 0 {
		t.Errorf("expected scroll offset to stay at 0, got %d", sb.scroll.offset)
	}
}

func TestSidebarManualScrollPausesAutoScroll(t *testing.T) {
	store := testStore(t)
	longName := "ThisIsAVeryLongEntryNameThatDefinitelyExceedsTheAvailableWidth"
	e := Entry{ID: GenerateID(), Name: longName, Query: "{ test }", CreatedAt: time.Now()}
	_ = store.AddEntry(e)

	sb := NewSidebar(store)
	sb.SetSize(25, 20)
	sb.scroll.lastIdx = sb.list.Index()

	// Manual scroll right — should pause auto-scroll
	sb, _ = sb.Update(tea.KeyPressMsg{Code: tea.KeyRight, Text: "right"})
	if !sb.scroll.paused {
		t.Error("expected scroll paused after manual scroll")
	}

	// Tick should be ignored while paused
	sb, cmd := sb.Update(scrollTickMsg{})
	if sb.scroll.offset != 1 {
		t.Errorf("expected offset unchanged at 1, got %d", sb.scroll.offset)
	}
	if cmd != nil {
		t.Error("expected no next tick while paused")
	}
}

func TestSidebarScrollPauseClearsOnNavigation(t *testing.T) {
	store := testStore(t)
	e1 := Entry{ID: GenerateID(), Name: "VeryLongFirstEntryName!!", Query: "{ a }", CreatedAt: time.Now()}
	e2 := Entry{ID: GenerateID(), Name: "Second", Query: "{ b }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.AddEntry(e2)

	sb := NewSidebar(store)
	sb.SetSize(25, 20)
	sb.scroll.paused = true
	sb.scroll.offset = 3

	// Navigate to next item — should clear pause
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if sb.scroll.paused {
		t.Error("expected pause cleared after navigation")
	}
	if sb.scroll.offset != 0 {
		t.Errorf("expected offset reset, got %d", sb.scroll.offset)
	}
}

func TestSidebarScrollResetsOnNavigation(t *testing.T) {
	store := testStore(t)
	e1 := Entry{ID: GenerateID(), Name: "First", Query: "{ a }", CreatedAt: time.Now()}
	e2 := Entry{ID: GenerateID(), Name: "Second", Query: "{ b }", CreatedAt: time.Now()}
	_ = store.AddEntry(e1)
	_ = store.AddEntry(e2)

	sb := NewSidebar(store)
	sb.SetSize(40, 20)

	// Set up scroll state
	sb.scroll.offset = 5
	sb.scroll.active = true

	// Navigate to next item
	sb, _ = sb.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})

	// Scroll should be reset
	if sb.scroll.offset != 0 {
		t.Errorf("expected scroll offset reset after navigation, got %d", sb.scroll.offset)
	}
	if sb.scroll.active {
		t.Error("expected scroll inactive after navigation")
	}
}
