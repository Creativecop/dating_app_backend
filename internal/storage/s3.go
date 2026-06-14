package storage

import (
	"context"
	"fmt"
	"os"
)

type S3Provider struct{}

func NewS3Provider() *S3Provider {
	return &S3Provider{}
}

func (p *S3Provider) Name() string {
	return "s3"
}

func (p *S3Provider) CreateTemp(context.Context) (*TempFile, error) {
	return nil, fmt.Errorf("s3 storage is not implemented in Phase 4")
}

func (p *S3Provider) CommitTemp(context.Context, *TempFile, string) error {
	return fmt.Errorf("s3 storage is not implemented in Phase 4")
}

func (p *S3Provider) Delete(context.Context, string) error {
	return fmt.Errorf("s3 storage is not implemented in Phase 4")
}

func (p *S3Provider) Open(context.Context, string) (*os.File, error) {
	return nil, fmt.Errorf("s3 storage is not implemented in Phase 4")
}

func (p *S3Provider) LocalPath(string) (string, error) {
	return "", fmt.Errorf("s3 storage is not implemented in Phase 4")
}
