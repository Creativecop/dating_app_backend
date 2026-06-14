package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type LocalProvider struct {
	root string
}

func NewLocalProvider(root string) (*LocalProvider, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("local storage root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(absRoot, "tmp"), 0o755); err != nil {
		return nil, err
	}
	return &LocalProvider{root: absRoot}, nil
}

func (p *LocalProvider) Name() string {
	return "local"
}

func (p *LocalProvider) CreateTemp(_ context.Context) (*TempFile, error) {
	key := "tmp/" + uuid.NewString() + ".upload"
	path, err := p.LocalPath(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &TempFile{File: file, Key: key, Path: path}, nil
}

func (p *LocalProvider) CommitTemp(_ context.Context, temp *TempFile, objectKey string) error {
	if temp == nil || temp.Path == "" {
		return fmt.Errorf("temp file is required")
	}
	if temp.File != nil {
		_ = temp.File.Close()
	}
	finalPath, err := p.LocalPath(objectKey)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return err
	}
	return os.Rename(temp.Path, finalPath)
}

func (p *LocalProvider) Delete(_ context.Context, objectKey string) error {
	path, err := p.LocalPath(objectKey)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (p *LocalProvider) Open(_ context.Context, objectKey string) (*os.File, error) {
	path, err := p.LocalPath(objectKey)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (p *LocalProvider) LocalPath(objectKey string) (string, error) {
	if filepath.IsAbs(objectKey) {
		return "", fmt.Errorf("invalid object key")
	}
	cleanKey := filepath.Clean(strings.TrimPrefix(objectKey, "/"))
	if cleanKey == "." || strings.HasPrefix(cleanKey, "..") || filepath.IsAbs(cleanKey) {
		return "", fmt.Errorf("invalid object key")
	}
	path := filepath.Join(p.root, cleanKey)
	rel, err := filepath.Rel(p.root, path)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("object key escapes storage root")
	}
	return path, nil
}
