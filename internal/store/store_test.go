package store

import (
	"bliss/internal/list"
	"bliss/internal/todo"
	"os"
	"path/filepath"
	"testing"
)

// newTestStore creates a Store backed by a temp directory for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()

	// Create directory structure
	dirs := []string{
		filepath.Join(dir, "contexts"),
		filepath.Join(dir, "lists"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}

	// Initialize git repo
	repo, err := initGitRepo(dir)
	if err != nil {
		t.Fatalf("initGitRepo: %v", err)
	}

	return &Store{path: dir, repo: repo}
}

func TestInit(t *testing.T) {
	dir := t.TempDir()

	dirs := []string{
		filepath.Join(dir, "contexts"),
		filepath.Join(dir, "lists"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("creating dir: %v", err)
		}
	}

	repo, err := initGitRepo(dir)
	if err != nil {
		t.Fatalf("initGitRepo: %v", err)
	}

	s := &Store{path: dir, repo: repo}

	// Verify dirs exist
	for _, d := range dirs {
		if _, err := os.Stat(d); err != nil {
			t.Errorf("dir %s not found: %v", d, err)
		}
	}
	// Verify git repo initialized
	if s.repo == nil {
		t.Error("repo is nil")
	}
}

func TestWriteReadTodo(t *testing.T) {
	s := newTestStore(t)
	contextUUID := "ctx-uuid-1"

	original := todo.Todo{
		UUID:  "todo-uuid-1",
		Title: "Feed the penguins",
		Body:  "Bring fish from freezer.",
	}

	if err := s.WriteTodo(contextUUID, original); err != nil {
		t.Fatalf("WriteTodo: %v", err)
	}

	got, err := s.ReadTodo(contextUUID, original.UUID)
	if err != nil {
		t.Fatalf("ReadTodo: %v", err)
	}

	if got.UUID != original.UUID {
		t.Errorf("UUID = %q, want %q", got.UUID, original.UUID)
	}
	if got.Title != original.Title {
		t.Errorf("Title = %q, want %q", got.Title, original.Title)
	}
	if got.Body != original.Body {
		t.Errorf("Body = %q, want %q", got.Body, original.Body)
	}
}

func TestDeleteTodo(t *testing.T) {
	s := newTestStore(t)
	contextUUID := "ctx-uuid-2"

	t1 := todo.Todo{UUID: "todo-a", Title: "Task A"}
	if err := s.WriteTodo(contextUUID, t1); err != nil {
		t.Fatalf("WriteTodo: %v", err)
	}

	if err := s.DeleteTodo(contextUUID, t1.UUID); err != nil {
		t.Fatalf("DeleteTodo: %v", err)
	}

	// Reading should fail
	_, err := s.ReadTodo(contextUUID, t1.UUID)
	if err == nil {
		t.Error("expected error reading deleted todo, got nil")
	}
}

func TestWriteReadList(t *testing.T) {
	s := newTestStore(t)
	contextUUID := "ctx-uuid-3"

	original := list.List{
		Sections: []list.Section{
			{Items: []string{"uuid-1", "uuid-2"}},
			{Name: "urgent", Items: []string{"uuid-3"}},
		},
	}

	if err := s.WriteList(contextUUID, "today", original); err != nil {
		t.Fatalf("WriteList: %v", err)
	}

	got, err := s.ReadList(contextUUID, "today")
	if err != nil {
		t.Fatalf("ReadList: %v", err)
	}

	if len(got.Sections) != len(original.Sections) {
		t.Fatalf("sections = %d, want %d", len(got.Sections), len(original.Sections))
	}

	allOrig := list.AllUUIDs(original)
	allGot := list.AllUUIDs(got)
	if len(allOrig) != len(allGot) {
		t.Fatalf("AllUUIDs len = %d, want %d", len(allGot), len(allOrig))
	}
	for i := range allOrig {
		if allOrig[i] != allGot[i] {
			t.Errorf("UUID[%d] = %q, want %q", i, allGot[i], allOrig[i])
		}
	}
}

func TestCommit(t *testing.T) {
	s := newTestStore(t)
	contextUUID := "ctx-commit"

	// Write a file to commit
	t1 := todo.Todo{UUID: "todo-commit-1", Title: "Commit test"}
	if err := s.WriteTodo(contextUUID, t1); err != nil {
		t.Fatalf("WriteTodo: %v", err)
	}

	if err := s.Commit("test commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify repo has at least one commit
	head, err := s.repo.Head()
	if err != nil {
		t.Fatalf("getting HEAD: %v", err)
	}
	if head == nil {
		t.Error("HEAD is nil after commit")
	}
}
