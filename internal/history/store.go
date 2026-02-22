package history

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Entry represents a single saved query in history.
type Entry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	Variables string    `json:"variables"`
	Endpoint  string    `json:"endpoint"`
	CreatedAt time.Time `json:"createdAt"`
}

// Folder groups entries under a user-defined name.
type Folder struct {
	Name    string
	Entries []Entry // sorted newest-first by CreatedAt
}

// Meta stores sidebar UI state and folder ordering.
type Meta struct {
	FolderOrder []string `json:"folderOrder"`
	Collapsed   []string `json:"collapsed"`
	SidebarOpen bool     `json:"sidebarOpen"`
}

const (
	metaFile     = "_meta.json"
	unsortedDir  = "unsorted"
	maxEntries   = 25
	maxNameLen   = 30
)

// Store manages on-disk history storage.
type Store struct {
	dir      string
	Meta     Meta
	folders  []Folder
	unsorted []Entry
}

// NewStore creates a new Store rooted at dir.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Load reads the directory structure: scan subdirs for .json files,
// read _meta.json, populate folders + unsorted. Skips corrupt files.
func (s *Store) Load() error {
	// Ensure base directory exists
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	// Ensure unsorted/ exists
	if err := os.MkdirAll(filepath.Join(s.dir, unsortedDir), 0o755); err != nil {
		return err
	}

	// Read meta
	s.Meta = Meta{SidebarOpen: true}
	metaPath := filepath.Join(s.dir, metaFile)
	if data, err := os.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(data, &s.Meta)
	}

	// Scan directories
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	folderMap := make(map[string][]Entry)
	var dirNames []string

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		dirNames = append(dirNames, name)
		folderMap[name] = s.loadEntries(filepath.Join(s.dir, name))
	}

	// Build unsorted
	s.unsorted = folderMap[unsortedDir]
	sortEntriesNewestFirst(s.unsorted)

	// Build folders in meta order, then append any dirs not in meta
	s.folders = nil
	seen := map[string]bool{unsortedDir: true}
	for _, name := range s.Meta.FolderOrder {
		if entries, ok := folderMap[name]; ok && name != unsortedDir {
			sortEntriesNewestFirst(entries)
			s.folders = append(s.folders, Folder{Name: name, Entries: entries})
			seen[name] = true
		}
	}
	for _, name := range dirNames {
		if !seen[name] {
			entries := folderMap[name]
			sortEntriesNewestFirst(entries)
			s.folders = append(s.folders, Folder{Name: name, Entries: entries})
		}
	}

	return nil
}

func (s *Store) loadEntries(dir string) []Entry {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var result []Entry
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			continue
		}
		var e Entry
		if err := json.Unmarshal(data, &e); err != nil {
			continue // skip corrupt
		}
		result = append(result, e)
	}
	return result
}

func sortEntriesNewestFirst(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})
}

// Save writes _meta.json atomically.
func (s *Store) Save() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.Meta, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, metaFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, metaFile))
}

// SaveEntry writes an entry JSON to <dir>/<folder>/<id>.json atomically.
func (s *Store) SaveEntry(e Entry, folder string) error {
	dir := filepath.Join(s.dir, folder)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, e.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AddEntry adds an entry to unsorted, enforces 25-entry global limit, and persists.
func (s *Store) AddEntry(e Entry) error {
	s.unsorted = append([]Entry{e}, s.unsorted...)
	if err := s.SaveEntry(e, unsortedDir); err != nil {
		return err
	}

	// Enforce global limit
	all := s.AllEntries()
	if len(all) > maxEntries {
		// Find and evict the oldest entries
		sort.Slice(all, func(i, j int) bool {
			return all[i].CreatedAt.After(all[j].CreatedAt)
		})
		for _, old := range all[maxEntries:] {
			_ = s.DeleteEntry(old.ID)
		}
	}
	return nil
}

// DeleteEntry finds an entry by ID across all folders and removes its file.
func (s *Store) DeleteEntry(id string) error {
	// Check unsorted
	for i, e := range s.unsorted {
		if e.ID == id {
			s.unsorted = append(s.unsorted[:i], s.unsorted[i+1:]...)
			return os.Remove(filepath.Join(s.dir, unsortedDir, id+".json"))
		}
	}
	// Check folders
	for fi := range s.folders {
		for ei, e := range s.folders[fi].Entries {
			if e.ID == id {
				s.folders[fi].Entries = append(s.folders[fi].Entries[:ei], s.folders[fi].Entries[ei+1:]...)
				return os.Remove(filepath.Join(s.dir, s.folders[fi].Name, id+".json"))
			}
		}
	}
	return nil
}

// MoveEntry moves an entry from its current folder to the target folder.
func (s *Store) MoveEntry(id, toFolder string) error {
	// Find entry and its current folder
	var entry Entry
	var fromDir string
	found := false

	for i, e := range s.unsorted {
		if e.ID == id {
			entry = e
			fromDir = unsortedDir
			s.unsorted = append(s.unsorted[:i], s.unsorted[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		for fi := range s.folders {
			for ei, e := range s.folders[fi].Entries {
				if e.ID == id {
					entry = e
					fromDir = s.folders[fi].Name
					s.folders[fi].Entries = append(s.folders[fi].Entries[:ei], s.folders[fi].Entries[ei+1:]...)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	if !found {
		return nil
	}

	// Remove old file
	_ = os.Remove(filepath.Join(s.dir, fromDir, id+".json"))

	// Add to target
	if toFolder == unsortedDir {
		s.unsorted = append([]Entry{entry}, s.unsorted...)
	} else {
		for fi := range s.folders {
			if s.folders[fi].Name == toFolder {
				s.folders[fi].Entries = append([]Entry{entry}, s.folders[fi].Entries...)
				break
			}
		}
	}

	return s.SaveEntry(entry, toFolder)
}

// CreateFolder creates a new folder directory and updates meta.
func (s *Store) CreateFolder(name string) error {
	if err := os.MkdirAll(filepath.Join(s.dir, name), 0o755); err != nil {
		return err
	}
	s.folders = append(s.folders, Folder{Name: name})
	s.Meta.FolderOrder = append(s.Meta.FolderOrder, name)
	return s.Save()
}

// DeleteFolder removes a folder. Entries are moved to unsorted first.
func (s *Store) DeleteFolder(name string) error {
	for fi := range s.folders {
		if s.folders[fi].Name == name {
			// Move entries to unsorted
			for _, e := range s.folders[fi].Entries {
				_ = os.Remove(filepath.Join(s.dir, name, e.ID+".json"))
				s.unsorted = append([]Entry{e}, s.unsorted...)
				_ = s.SaveEntry(e, unsortedDir)
			}
			s.folders = append(s.folders[:fi], s.folders[fi+1:]...)
			break
		}
	}

	// Remove from meta
	for i, n := range s.Meta.FolderOrder {
		if n == name {
			s.Meta.FolderOrder = append(s.Meta.FolderOrder[:i], s.Meta.FolderOrder[i+1:]...)
			break
		}
	}

	// Remove collapsed state
	for i, n := range s.Meta.Collapsed {
		if n == name {
			s.Meta.Collapsed = append(s.Meta.Collapsed[:i], s.Meta.Collapsed[i+1:]...)
			break
		}
	}

	_ = os.RemoveAll(filepath.Join(s.dir, name))
	return s.Save()
}

// RenameFolder renames a folder directory and updates meta.
func (s *Store) RenameFolder(old, newName string) error {
	if err := os.Rename(filepath.Join(s.dir, old), filepath.Join(s.dir, newName)); err != nil {
		return err
	}
	for fi := range s.folders {
		if s.folders[fi].Name == old {
			s.folders[fi].Name = newName
			break
		}
	}
	for i, n := range s.Meta.FolderOrder {
		if n == old {
			s.Meta.FolderOrder[i] = newName
			break
		}
	}
	for i, n := range s.Meta.Collapsed {
		if n == old {
			s.Meta.Collapsed[i] = newName
			break
		}
	}
	return s.Save()
}

// RenameEntry loads an entry, updates its name, and saves.
func (s *Store) RenameEntry(id, newName string) error {
	// Find in unsorted
	for i, e := range s.unsorted {
		if e.ID == id {
			s.unsorted[i].Name = newName
			return s.SaveEntry(s.unsorted[i], unsortedDir)
		}
	}
	// Find in folders
	for fi := range s.folders {
		for ei, e := range s.folders[fi].Entries {
			if e.ID == id {
				s.folders[fi].Entries[ei].Name = newName
				return s.SaveEntry(s.folders[fi].Entries[ei], s.folders[fi].Name)
			}
		}
	}
	return nil
}

// AllEntries returns a flat list of all entries across folders and unsorted.
func (s *Store) AllEntries() []Entry {
	var all []Entry
	all = append(all, s.unsorted...)
	for _, f := range s.folders {
		all = append(all, f.Entries...)
	}
	return all
}

// Folders returns the user-defined folders.
func (s *Store) Folders() []Folder {
	return s.folders
}

// Unsorted returns entries in the unsorted directory.
func (s *Store) Unsorted() []Entry {
	return s.unsorted
}

// HasContent returns true if there are any entries or folders.
func (s *Store) HasContent() bool {
	if len(s.unsorted) > 0 {
		return true
	}
	for _, f := range s.folders {
		if len(f.Entries) > 0 {
			return true
		}
	}
	return len(s.folders) > 0
}

// IsDuplicate checks if the most recent entry matches the given query/variables/endpoint.
func (s *Store) IsDuplicate(query, variables, endpoint string) bool {
	all := s.AllEntries()
	if len(all) == 0 {
		return false
	}
	sortEntriesNewestFirst(all)
	newest := all[0]
	return newest.Query == query && newest.Variables == variables && newest.Endpoint == endpoint
}

// SetCollapsed updates the collapsed state for a folder.
func (s *Store) SetCollapsed(folder string, collapsed bool) {
	if collapsed {
		for _, n := range s.Meta.Collapsed {
			if n == folder {
				return
			}
		}
		s.Meta.Collapsed = append(s.Meta.Collapsed, folder)
	} else {
		for i, n := range s.Meta.Collapsed {
			if n == folder {
				s.Meta.Collapsed = append(s.Meta.Collapsed[:i], s.Meta.Collapsed[i+1:]...)
				return
			}
		}
	}
}

// IsCollapsed returns whether a folder is collapsed.
func (s *Store) IsCollapsed(folder string) bool {
	for _, n := range s.Meta.Collapsed {
		if n == folder {
			return true
		}
	}
	return false
}

// GenerateID generates a random 16-char hex ID.
func GenerateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var operationRE = regexp.MustCompile(`^\s*(?:query|mutation|subscription)\s+(\w+)`)
var firstFieldRE = regexp.MustCompile(`\{\s*(\w+)`)

// EntryNameFromQuery extracts a short name from a GraphQL query string.
func EntryNameFromQuery(query string) string {
	if query == "" {
		return "unnamed"
	}
	// Try operation name: "query GetUser { ... }" → "GetUser"
	if m := operationRE.FindStringSubmatch(query); len(m) > 1 {
		return truncate(m[1], maxNameLen)
	}
	// Try first field: "{ user { name } }" → "user"
	if m := firstFieldRE.FindStringSubmatch(query); len(m) > 1 {
		return truncate(m[1], maxNameLen)
	}
	return "unnamed"
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
