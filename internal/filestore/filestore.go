package filestore

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Store persists uploaded files to a local directory.
type Store struct {
	dir       string
	urlPrefix string
}

// New creates a Store backed by dir, creating the directory if it does not exist.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("filestore: mkdir %s: %w", dir, err)
	}
	return &Store{dir: dir, urlPrefix: "/uploads"}, nil
}

// Save writes data to a new file with the given extension and returns its URL path.
func (s *Store) Save(data []byte, ext string) (string, error) {
	name := uuid.NewString() + ext
	path := filepath.Join(s.dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("filestore: write: %w", err)
	}
	return s.urlPrefix + "/" + name, nil
}
