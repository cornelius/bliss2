package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindContext_found(t *testing.T) {
	dir := t.TempDir()
	uuid := "7f3a2b1c-4d5e-6f7a-8b9c-0d1e2f3a4b5c"

	if err := WriteContextFile(dir, uuid); err != nil {
		t.Fatalf("WriteContextFile: %v", err)
	}

	gotUUID, gotDir, err := FindContext(dir)
	if err != nil {
		t.Fatalf("FindContext: %v", err)
	}
	if gotUUID != uuid {
		t.Errorf("UUID = %q, want %q", gotUUID, uuid)
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

	uuid := "aabbccdd-1122-3344-5566-778899aabbcc"
	if err := WriteContextFile(parentDir, uuid); err != nil {
		t.Fatalf("WriteContextFile: %v", err)
	}

	gotUUID, gotDir, err := FindContext(childDir)
	if err != nil {
		t.Fatalf("FindContext: %v", err)
	}
	if gotUUID != uuid {
		t.Errorf("UUID = %q, want %q", gotUUID, uuid)
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
