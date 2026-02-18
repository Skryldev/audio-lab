package storage

import (
	"context"
	"os"
	"path/filepath"
)

// LocalStorage implements ports.StorageProvider for local filesystem
type LocalStorage struct{}

// NewLocalStorage creates a new local storage provider
func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

// Exists checks if a file exists
func (s *LocalStorage) Exists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Size returns file size in bytes
func (s *LocalStorage) Size(_ context.Context, path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// Remove deletes a file
func (s *LocalStorage) Remove(_ context.Context, path string) error {
	return os.Remove(path)
}

// TempFile creates a temporary file and returns its path
func (s *LocalStorage) TempFile(_ context.Context, dir, pattern string) (string, error) {
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return filepath.Abs(f.Name())
}