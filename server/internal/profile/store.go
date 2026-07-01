package profile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"home-stream/server/internal/datafile"
	"home-stream/server/internal/id"
)

var ErrNotFound = errors.New("guest not found")

type Store struct {
	mu     sync.RWMutex
	path   string
	guests map[string]Guest
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		path:   path,
		guests: make(map[string]Guest),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Create(role, name, photoURL, ip string) (Guest, error) {
	guestID, err := id.New(16)
	if err != nil {
		return Guest{}, err
	}
	secret, err := id.New(32)
	if err != nil {
		return Guest{}, err
	}

	now := time.Now().UTC()
	guest := Guest{
		ID:        guestID,
		Role:      NormalizeRole(role),
		Name:      name,
		PhotoURL:  photoURL,
		Secret:    secret,
		IP:        ip,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.guests[guest.ID] = guest
	if err := s.saveLocked(); err != nil {
		return Guest{}, err
	}
	return guest, nil
}

func (s *Store) Authenticate(guestID, secret string) (Guest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	guest, ok := s.guests[guestID]
	if !ok || guest.Secret != secret {
		return Guest{}, false
	}
	guest.Role = NormalizeRole(guest.Role)
	return guest, true
}

func (s *Store) Update(guestID, name, photoURL string) (Guest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	guest, ok := s.guests[guestID]
	if !ok {
		return Guest{}, ErrNotFound
	}
	if name != "" {
		guest.Name = name
	}
	guest.PhotoURL = photoURL
	guest.UpdatedAt = time.Now().UTC()
	s.guests[guest.ID] = guest

	if err := s.saveLocked(); err != nil {
		return Guest{}, err
	}
	return guest, nil
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
	if err := json.Unmarshal(data, &s.guests); err != nil {
		if backupErr := datafile.BackupCorrupt(s.path); backupErr != nil {
			return backupErr
		}
		s.guests = make(map[string]Guest)
		return s.saveLocked()
	}
	if s.guests == nil {
		s.guests = make(map[string]Guest)
	}
	return nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.guests, "", "  ")
	if err != nil {
		return err
	}
	return datafile.WriteAtomic(s.path, data, 0o600)
}
