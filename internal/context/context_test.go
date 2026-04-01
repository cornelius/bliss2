package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindContext_found(t *testing.T) {
	dir := t.TempDir()
	name := "my-project"

	if err := WriteContextFile(dir, name); err != nil {
		t.Fatalf("WriteContextFile: %v", err)
	}

	gotName, gotDir, err := FindContext(dir)
	if err != nil {
		t.Fatalf("FindContext: %v", err)
	}
	if gotName != name {
		t.Errorf("name = %q, want %q", gotName, name)
	}
	if gotDir != dir {
		t.Errorf("dir = %q, want %q", gotDir, dir)
	}
}

func TestFindContext_walkUp(t *testing.T) {
	parentDir := t.TempDir()
	childDir := filepath.Join(parentDir, "child", "grandchild")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	name := "parent-project"
	if err := WriteContextFile(parentDir, name); err != nil {
		t.Fatalf("WriteContextFile: %v", err)
	}

	gotName, gotDir, err := FindContext(childDir)
	if err != nil {
		t.Fatalf("FindContext: %v", err)
	}
	if gotName != name {
		t.Errorf("name = %q, want %q", gotName, name)
	}
	if gotDir != parentDir {
		t.Errorf("dir = %q, want %q", gotDir, parentDir)
	}
}

func TestFindContext_notFound(t *testing.T) {
	dir := t.TempDir()

	_, _, err := FindContext(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
