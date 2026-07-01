package event

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"home-stream/server/internal/datafile"
)

type Store struct {
	mu       sync.RWMutex
	path     string
	settings Settings
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		path:     path,
		settings: DefaultSettings(),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Store) Update(title, description string) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if title != "" {
		s.settings.Title = title
	}
	s.settings.Description = description
	s.settings.UpdatedAt = time.Now().UTC()

	if err := s.saveLocked(); err != nil {
		return Settings{}, err
	}
	return s.settings, nil
}

func (s *Store) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.saveLocked()
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &s.settings); err != nil {
		if backupErr := datafile.BackupCorrupt(s.path); backupErr != nil {
			return backupErr
		}
		s.settings = DefaultSettings()
		return s.saveLocked()
	}
	if s.settings.Title == "" {
		s.settings.Title = DefaultSettings().Title
	}
	if s.settings.UpdatedAt.IsZero() {
		s.settings.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	return datafile.WriteAtomic(s.path, data, 0o600)
}
