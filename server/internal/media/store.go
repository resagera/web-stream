package media

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"home-stream/server/internal/id"
)

var ErrInvalidImage = errors.New("invalid image data")
var ErrTooLarge = errors.New("image data is too large")

type Store struct {
	dir      string
	urlPath  string
	maxBytes int
}

func NewStore(dir, urlPath string, maxBytes int) (*Store, error) {
	if maxBytes <= 0 {
		maxBytes = 350000
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir, urlPath: strings.TrimRight(urlPath, "/"), maxBytes: maxBytes}, nil
}

func (s *Store) SavePhoto(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, s.urlPath+"/") {
		return value, nil
	}

	contentType, encoded, ok := strings.Cut(value, ",")
	if !ok || !strings.HasPrefix(contentType, "data:image/") || !strings.Contains(contentType, ";base64") {
		return "", ErrInvalidImage
	}
	if len(encoded) > s.maxBytes*2 {
		return "", ErrTooLarge
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidImage
	}
	if len(data) > s.maxBytes {
		return "", ErrTooLarge
	}

	ext := extension(contentType)
	if ext == "" {
		return "", ErrInvalidImage
	}
	if !hasImageSignature(ext, data) {
		return "", ErrInvalidImage
	}
	name, err := id.New(18)
	if err != nil {
		return "", err
	}
	filename := name + ext
	if err := os.WriteFile(filepath.Join(s.dir, filename), data, 0o600); err != nil {
		return "", err
	}
	return s.urlPath + "/" + filename, nil
}

func extension(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "data:image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(contentType, "data:image/png"):
		return ".png"
	case strings.HasPrefix(contentType, "data:image/webp"):
		return ".webp"
	default:
		return ""
	}
}

func hasImageSignature(ext string, data []byte) bool {
	switch ext {
	case ".jpg":
		return len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
	case ".png":
		return bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	case ".webp":
		return len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
	default:
		return false
	}
}
