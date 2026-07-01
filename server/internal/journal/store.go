package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"home-stream/server/internal/datafile"
	"home-stream/server/internal/id"
)

type Store struct {
	mu      sync.RWMutex
	path    string
	limit   int
	entries []Entry
}

func NewStore(path string, limit int) (*Store, error) {
	if limit <= 0 {
		limit = 200
	}
	s := &Store{
		path:  path,
		limit: limit,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Record(event Event) (Entry, error) {
	entryID, err := id.New(16)
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{
		ID:        entryID,
		Type:      event.Type,
		GuestID:   event.GuestID,
		Name:      event.Name,
		Role:      event.Role,
		IP:        event.IP,
		Detail:    event.Detail,
		Metadata:  event.Metadata,
		CreatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return Entry{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return Entry{}, err
	}
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Entry{}, err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return Entry{}, err
	}
	if err := file.Close(); err != nil {
		return Entry{}, err
	}

	s.entries = append(s.entries, entry)
	if len(s.entries) > s.limit {
		s.entries = s.entries[len(s.entries)-s.limit:]
	}
	return entry, nil
}

func (s *Store) Last(limit int) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.entries) {
		limit = len(s.entries)
	}
	result := make([]Entry, 0, limit)
	for i := len(s.entries) - limit; i < len(s.entries); i++ {
		result = append(result, s.entries[i])
	}
	return result
}

func (s *Store) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		s.entries = append(s.entries, entry)
		if len(s.entries) > s.limit {
			s.entries = s.entries[len(s.entries)-s.limit:]
		}
	}
	if err := scanner.Err(); err != nil {
		if backupErr := datafile.BackupCorrupt(s.path); backupErr != nil {
			return backupErr
		}
		s.entries = nil
		return nil
	}
	return nil
}
