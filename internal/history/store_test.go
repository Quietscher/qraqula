package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadEmptyDirCreatesUnsorted(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "unsorted"))
	if err != nil {
		t.Fatal("expected unsorted/ dir to be created")
	}
	if !info.IsDir() {
		t.Error("expected unsorted to be a directory")
	}
}

func TestSaveEntryAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	e := Entry{
		ID:        GenerateID(),
		Name:      "test",
		Query:     "{ users { name } }",
		Variables: "{}",
		Endpoint:  "https://example.com/graphql",
		CreatedAt: time.Now(),
	}
	if err := s.SaveEntry(e, "unsorted"); err != nil {
		t.Fatal(err)
	}

	// Reload
	s2 := NewStore(dir)
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.unsorted) != 1 {
		t.Fatalf("expected 1 unsorted entry, got %d", len(s2.unsorted))
	}
	if s2.unsorted[0].ID != e.ID {
		t.Errorf("expected ID %s, got %s", e.ID, s2.unsorted[0].ID)
	}
	if s2.unsorted[0].Query != e.Query {
		t.Errorf("expected query %q, got %q", e.Query, s2.unsorted[0].Query)
	}
}

func TestAddEntryRespectsLimit(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	// Add 26 entries
	for i := 0; i < 26; i++ {
		e := Entry{
			ID:        GenerateID(),
			Name:      "entry",
			Query:     "{ test }",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := s.AddEntry(e); err != nil {
			t.Fatal(err)
		}
	}

	all := s.AllEntries()
	if len(all) != 25 {
		t.Errorf("expected 25 entries after limit enforcement, got %d", len(all))
	}
}

func TestIsDuplicateDetectsMatch(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	e := Entry{
		ID:        GenerateID(),
		Name:      "test",
		Query:     "{ users }",
		Variables: "{}",
		Endpoint:  "https://example.com/graphql",
		CreatedAt: time.Now(),
	}
	_ = s.AddEntry(e)

	if !s.IsDuplicate("{ users }", "{}", "https://example.com/graphql") {
		t.Error("expected duplicate to be detected")
	}
	if s.IsDuplicate("{ different }", "{}", "https://example.com/graphql") {
		t.Error("expected different query not to be duplicate")
	}
}

func TestIsDuplicateEmptyStore(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	if s.IsDuplicate("{ test }", "", "") {
		t.Error("expected no duplicate in empty store")
	}
}

func TestMoveEntry(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	_ = s.CreateFolder("myFolder")
	e := Entry{
		ID:        GenerateID(),
		Name:      "test",
		Query:     "{ users }",
		CreatedAt: time.Now(),
	}
	_ = s.AddEntry(e)

	if len(s.unsorted) != 1 {
		t.Fatal("expected 1 unsorted entry")
	}

	_ = s.MoveEntry(e.ID, "myFolder")

	if len(s.unsorted) != 0 {
		t.Errorf("expected 0 unsorted after move, got %d", len(s.unsorted))
	}
	if len(s.folders[0].Entries) != 1 {
		t.Errorf("expected 1 entry in folder, got %d", len(s.folders[0].Entries))
	}

	// Verify file exists in new location
	_, err := os.Stat(filepath.Join(dir, "myFolder", e.ID+".json"))
	if err != nil {
		t.Error("expected file in new folder")
	}
	// Verify file removed from old location
	_, err = os.Stat(filepath.Join(dir, "unsorted", e.ID+".json"))
	if !os.IsNotExist(err) {
		t.Error("expected file removed from unsorted")
	}
}

func TestCreateAndDeleteFolder(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	_ = s.CreateFolder("testFolder")
	if len(s.folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(s.folders))
	}
	if s.folders[0].Name != "testFolder" {
		t.Errorf("expected folder name 'testFolder', got %q", s.folders[0].Name)
	}

	// Folder should be in meta
	if len(s.Meta.FolderOrder) != 1 || s.Meta.FolderOrder[0] != "testFolder" {
		t.Error("expected folder in meta order")
	}

	// Delete
	_ = s.DeleteFolder("testFolder")
	if len(s.folders) != 0 {
		t.Errorf("expected 0 folders after delete, got %d", len(s.folders))
	}
	if len(s.Meta.FolderOrder) != 0 {
		t.Error("expected empty folder order after delete")
	}
}

func TestDeleteFolderMovesEntriesToUnsorted(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	_ = s.CreateFolder("myFolder")
	e := Entry{
		ID:        GenerateID(),
		Name:      "test",
		Query:     "{ users }",
		CreatedAt: time.Now(),
	}
	_ = s.AddEntry(e)
	_ = s.MoveEntry(e.ID, "myFolder")

	_ = s.DeleteFolder("myFolder")
	if len(s.unsorted) != 1 {
		t.Errorf("expected 1 unsorted entry after folder delete, got %d", len(s.unsorted))
	}
}

func TestRenameFolder(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	_ = s.CreateFolder("oldName")
	_ = s.RenameFolder("oldName", "newName")

	if len(s.folders) != 1 || s.folders[0].Name != "newName" {
		t.Error("expected folder renamed to 'newName'")
	}
	if s.Meta.FolderOrder[0] != "newName" {
		t.Error("expected meta order updated")
	}

	// Verify directory was renamed
	_, err := os.Stat(filepath.Join(dir, "newName"))
	if err != nil {
		t.Error("expected newName directory to exist")
	}
}

func TestRenameEntry(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	e := Entry{
		ID:        GenerateID(),
		Name:      "original",
		Query:     "{ test }",
		CreatedAt: time.Now(),
	}
	_ = s.AddEntry(e)
	_ = s.RenameEntry(e.ID, "renamed")

	if s.unsorted[0].Name != "renamed" {
		t.Errorf("expected name 'renamed', got %q", s.unsorted[0].Name)
	}

	// Verify persisted
	s2 := NewStore(dir)
	_ = s2.Load()
	if s2.unsorted[0].Name != "renamed" {
		t.Errorf("expected persisted name 'renamed', got %q", s2.unsorted[0].Name)
	}
}

func TestEntryNameFromQuery(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"query GetUser { user { name } }", "GetUser"},
		{"{ user { name } }", "user"},
		{"mutation CreateUser { createUser { id } }", "CreateUser"},
		{"subscription OnMessage { onMessage { text } }", "OnMessage"},
		{"", "unnamed"},
		{"   ", "unnamed"},
	}
	for _, tt := range tests {
		got := EntryNameFromQuery(tt.query)
		if got != tt.want {
			t.Errorf("EntryNameFromQuery(%q) = %q, want %q", tt.query, got, tt.want)
		}
	}
}

func TestCorruptJSONSkippedOnLoad(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	// Write a valid entry
	e := Entry{
		ID:        GenerateID(),
		Name:      "valid",
		Query:     "{ test }",
		CreatedAt: time.Now(),
	}
	_ = s.SaveEntry(e, "unsorted")

	// Write corrupt JSON
	corruptPath := filepath.Join(dir, "unsorted", "corrupt.json")
	_ = os.WriteFile(corruptPath, []byte("not json{{{"), 0o644)

	// Reload
	s2 := NewStore(dir)
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.unsorted) != 1 {
		t.Errorf("expected 1 valid entry after skipping corrupt, got %d", len(s2.unsorted))
	}
}

func TestCollapsedState(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	_ = s.CreateFolder("test")

	s.SetCollapsed("test", true)
	if !s.IsCollapsed("test") {
		t.Error("expected folder to be collapsed")
	}

	s.SetCollapsed("test", false)
	if s.IsCollapsed("test") {
		t.Error("expected folder to not be collapsed")
	}

	// Double-collapse should not duplicate
	s.SetCollapsed("test", true)
	s.SetCollapsed("test", true)
	count := 0
	for _, n := range s.Meta.Collapsed {
		if n == "test" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 collapsed entry, got %d", count)
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if len(id) != 16 {
		t.Errorf("expected 16-char ID, got %d chars: %s", len(id), id)
	}
	// Each call should be unique
	id2 := GenerateID()
	if id == id2 {
		t.Error("expected unique IDs")
	}
}

func TestSaveAndLoadMeta(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	s.Meta.SidebarOpen = false
	s.Meta.FolderOrder = []string{"a", "b"}
	s.Meta.Collapsed = []string{"a"}
	_ = s.Save()

	s2 := NewStore(dir)
	_ = s2.Load()
	if s2.Meta.SidebarOpen {
		t.Error("expected sidebar closed")
	}
	if len(s2.Meta.FolderOrder) != 2 {
		t.Error("expected 2 folders in order")
	}
	if len(s2.Meta.Collapsed) != 1 {
		t.Error("expected 1 collapsed folder")
	}
}

func TestDeleteEntry(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	e := Entry{
		ID:        GenerateID(),
		Name:      "test",
		Query:     "{ test }",
		CreatedAt: time.Now(),
	}
	_ = s.AddEntry(e)

	_ = s.DeleteEntry(e.ID)
	if len(s.unsorted) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(s.unsorted))
	}

	// Verify file removed
	_, err := os.Stat(filepath.Join(dir, "unsorted", e.ID+".json"))
	if !os.IsNotExist(err) {
		t.Error("expected file to be removed")
	}
}

func TestAllEntries(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	_ = s.CreateFolder("myFolder")

	e1 := Entry{ID: GenerateID(), Name: "e1", Query: "{ a }", CreatedAt: time.Now()}
	e2 := Entry{ID: GenerateID(), Name: "e2", Query: "{ b }", CreatedAt: time.Now()}
	_ = s.AddEntry(e1)
	_ = s.AddEntry(e2)
	_ = s.MoveEntry(e2.ID, "myFolder")

	all := s.AllEntries()
	if len(all) != 2 {
		t.Errorf("expected 2 total entries, got %d", len(all))
	}
}

func TestMetaAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	_ = s.Load()

	s.Meta.SidebarOpen = false
	_ = s.Save()

	// Verify no temp file left behind
	_, err := os.Stat(filepath.Join(dir, metaFile+".tmp"))
	if err == nil {
		t.Error("expected temp file to be cleaned up after atomic write")
	}

	// Verify actual file is valid JSON
	data, err := os.ReadFile(filepath.Join(dir, metaFile))
	if err != nil {
		t.Fatal(err)
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Errorf("meta file is not valid JSON: %v", err)
	}
}
