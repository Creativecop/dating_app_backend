package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalProviderRejectsPathTraversal(t *testing.T) {
	provider, err := NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.LocalPath("../secret.txt"); err == nil {
		t.Fatal("expected traversal key to fail")
	}
	if _, err := provider.LocalPath("/absolute/path"); err == nil {
		t.Fatal("expected absolute key to fail")
	}
}

func TestLocalProviderTempAndCommitStayUnderRoot(t *testing.T) {
	root := t.TempDir()
	provider, err := NewLocalProvider(root)
	if err != nil {
		t.Fatalf("provider: %v", err)
	}

	temp, err := provider.CreateTemp(context.Background())
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if _, err := temp.File.WriteString("hello"); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := provider.CommitTemp(context.Background(), temp, "users/user/profile/photos/media/original.jpg"); err != nil {
		t.Fatalf("commit temp: %v", err)
	}

	finalPath, err := provider.LocalPath("users/user/profile/photos/media/original.jpg")
	if err != nil {
		t.Fatalf("local path: %v", err)
	}
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("expected final file: %v", err)
	}
	if filepath.VolumeName(temp.Path) != filepath.VolumeName(finalPath) {
		t.Fatal("expected temp and final path on same volume")
	}
}
