package invite

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"home-stream/server/internal/datafile"
	"home-stream/server/internal/id"
	"home-stream/server/internal/profile"
)

var ErrNotFound = errors.New("invite not found")
var ErrInactive = errors.New("invite inactive or exhausted")

type Store struct {
	mu      sync.RWMutex
	path    string
	invites map[string]Invite
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		path:    path,
		invites: make(map[string]Invite),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Create(role, label string, maxUses int) (Invite, error) {
	token, err := id.New(24)
	if err != nil {
		return Invite{}, err
	}
	if maxUses < 0 {
		maxUses = 0
	}

	now := time.Now().UTC()
	invite := Invite{
		Token:     token,
		Role:      profile.NormalizeRole(role),
		Label:     label,
		Active:    true,
		MaxUses:   maxUses,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.invites[token] = invite
	if err := s.saveLocked(); err != nil {
		return Invite{}, err
	}
	return invite, nil
}

func (s *Store) List() []Invite {
	s.mu.RLock()
	defer s.mu.RUnlock()

	invites := make([]Invite, 0, len(s.invites))
	for _, invite := range s.invites {
		invites = append(invites, invite)
	}
	return invites
}

func (s *Store) Use(token string) (Invite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	invite, ok := s.invites[token]
	if !ok {
		return Invite{}, ErrNotFound
	}
	if !invite.CanUse() {
		return Invite{}, ErrInactive
	}

	invite.UsedCount++
	invite.UpdatedAt = time.Now().UTC()
	s.invites[token] = invite

	if err := s.saveLocked(); err != nil {
		return Invite{}, err
	}
	return invite, nil
}

func (s *Store) Disable(token string) (Invite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	invite, ok := s.invites[token]
	if !ok {
		return Invite{}, ErrNotFound
	}
	invite.Active = false
	invite.UpdatedAt = time.Now().UTC()
	s.invites[token] = invite

	if err := s.saveLocked(); err != nil {
		return Invite{}, err
	}
	return invite, nil
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
	if err := json.Unmarshal(data, &s.invites); err != nil {
		if backupErr := datafile.BackupCorrupt(s.path); backupErr != nil {
			return backupErr
		}
		s.invites = make(map[string]Invite)
		return s.saveLocked()
	}
	if s.invites == nil {
		s.invites = make(map[string]Invite)
	}
	return nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.invites, "", "  ")
	if err != nil {
		return err
	}
	return datafile.WriteAtomic(s.path, data, 0o600)
}
