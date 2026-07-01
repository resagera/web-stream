package chat

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"home-stream/server/internal/datafile"
)

type Storage struct {
	path string
}

func NewStorage(path string) Storage {
	return Storage{path: path}
}

func (s Storage) LoadLast(limit int) ([]Message, error) {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []Message
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var message Message
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			continue
		}
		messages = append(messages, message)
		if len(messages) > limit {
			copy(messages, messages[1:])
			messages = messages[:limit]
		}
	}
	if err := scanner.Err(); err != nil {
		if backupErr := datafile.BackupCorrupt(s.path); backupErr != nil {
			return nil, backupErr
		}
		return nil, nil
	}
	return messages, nil
}

func (s Storage) Append(message Message) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}
