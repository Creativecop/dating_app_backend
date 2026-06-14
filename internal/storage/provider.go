package storage

import (
	"context"
	"os"
)

type TempFile struct {
	File *os.File
	Key  string
	Path string
}

type Provider interface {
	Name() string
	CreateTemp(ctx context.Context) (*TempFile, error)
	CommitTemp(ctx context.Context, temp *TempFile, objectKey string) error
	Delete(ctx context.Context, objectKey string) error
	Open(ctx context.Context, objectKey string) (*os.File, error)
	LocalPath(objectKey string) (string, error)
}
