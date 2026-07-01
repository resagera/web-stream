package passwords

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"home-stream/server/internal/datafile"
	"home-stream/server/internal/profile"
)

type Defaults struct {
	Guest       string
	Broadcaster string
	Admin       string
}

type Settings struct {
	Guest       string    `json:"guest_password"`
	Broadcaster string    `json:"broadcaster_password,omitempty"`
	Admin       string    `json:"admin_password,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PublicSettings struct {
	GuestConfigured       bool      `json:"guest_configured"`
	BroadcasterConfigured bool      `json:"broadcaster_configured"`
	AdminConfigured       bool      `json:"admin_configured"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type Update struct {
	Guest       *string
	Broadcaster *string
	Admin       *string
}

type Store struct {
	mu       sync.RWMutex
	path     string
	settings Settings
}

func NewStore(path string, defaults Defaults) (*Store, error) {
	s := &Store{
		path: path,
		settings: Settings{
			Guest:       defaults.Guest,
			Broadcaster: defaults.Broadcaster,
			Admin:       defaults.Admin,
			UpdatedAt:   time.Now().UTC(),
		},
	}
	if s.settings.Guest == "" {
		s.settings.Guest = "change-me"
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Public() PublicSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return public(s.settings)
}

func (s *Store) Matches(role, password string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch profile.NormalizeRole(role) {
	case profile.RoleGuest:
		return password == s.settings.Guest
	case profile.RoleBroadcaster:
		return s.settings.Broadcaster != "" && password == s.settings.Broadcaster
	case profile.RoleAdmin:
		return s.settings.Admin != "" && password == s.settings.Admin
	default:
		return false
	}
}

func (s *Store) Update(update Update) (PublicSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	if update.Guest != nil {
		s.settings.Guest = *update.Guest
		changed = true
	}
	if update.Broadcaster != nil {
		s.settings.Broadcaster = *update.Broadcaster
		changed = true
	}
	if update.Admin != nil {
		s.settings.Admin = *update.Admin
		changed = true
	}
	if changed {
		s.settings.UpdatedAt = time.Now().UTC()
	}
	if err := s.saveLocked(); err != nil {
		return PublicSettings{}, err
	}
	return public(s.settings), nil
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
		s.settings.UpdatedAt = time.Now().UTC()
		return s.saveLocked()
	}
	if s.settings.Guest == "" {
		s.settings.Guest = "change-me"
	}
	if s.settings.UpdatedAt.IsZero() {
		s.settings.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	return datafile.WriteAtomic(s.path, data, 0o600)
}

func public(settings Settings) PublicSettings {
	return PublicSettings{
		GuestConfigured:       settings.Guest != "",
		BroadcasterConfigured: settings.Broadcaster != "",
		AdminConfigured:       settings.Admin != "",
		UpdatedAt:             settings.UpdatedAt,
	}
}
