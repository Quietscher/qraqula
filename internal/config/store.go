package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFile = "config.json"

// Store manages on-disk config storage.
type Store struct {
	dir    string
	Config Config
}

// NewStore creates a Store rooted at dir.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Load reads config.json from disk. If the file doesn't exist, Config stays zero-valued.
func (s *Store) Load() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.dir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.Config)
}

// Save writes config.json atomically (write .tmp then rename).
func (s *Store) Save() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.Config, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, configFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
