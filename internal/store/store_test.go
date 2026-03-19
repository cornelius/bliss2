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
		filepath.Join(dir, "todos"),
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
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", orig)

	s, err := Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if s == nil {
		t.Fatal("Init returned nil store")
	}
	if s.repo == nil {
		t.Error("Init returned store with nil repo")
	}

	storePath := filepath.Join(dir, ".bliss2")
	for _, sub := range []string{"contexts", "lists", "todos"} {
		if _, err := os.Stat(filepath.Join(storePath, sub)); err != nil {
			t.Errorf("dir %q not created: %v", sub, err)
		}
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

func TestWriteReadTodo_personal(t *testing.T) {
	s := newTestStore(t)

	original := todo.Todo{UUID: "personal-todo-1", Title: "Buy oat milk"}
	if err := s.WriteTodo("", original); err != nil {
		t.Fatalf("WriteTodo personal: %v", err)
	}

	got, err := s.ReadTodo("", original.UUID)
	if err != nil {
		t.Fatalf("ReadTodo personal: %v", err)
	}
	if got.Title != original.Title {
		t.Errorf("Title = %q, want %q", got.Title, original.Title)
	}
}

func TestListTodos_personal(t *testing.T) {
	s := newTestStore(t)

	t1 := todo.Todo{UUID: "p-todo-1", Title: "Alpha"}
	t2 := todo.Todo{UUID: "p-todo-2", Title: "Beta"}
	s.WriteTodo("", t1)
	s.WriteTodo("", t2)

	todos, err := s.ListTodos("")
	if err != nil {
		t.Fatalf("ListTodos personal: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("len = %d, want 2", len(todos))
	}
}

func TestDeleteTodo_personal(t *testing.T) {
	s := newTestStore(t)
	t1 := todo.Todo{UUID: "p-del-1", Title: "Delete me"}
	s.WriteTodo("", t1)

	if err := s.DeleteTodo("", t1.UUID); err != nil {
		t.Fatalf("DeleteTodo personal: %v", err)
	}
	if _, err := s.ReadTodo("", t1.UUID); err == nil {
		t.Error("expected error reading deleted personal todo")
	}
}

func TestListNames_personal(t *testing.T) {
	s := newTestStore(t)
	l := list.List{Sections: []list.Section{{Items: []string{"uuid-1"}}}}
	s.WriteList("", "today", l)

	names, err := s.ListNames("")
	if err != nil {
		t.Fatalf("ListNames personal: %v", err)
	}
	if len(names) != 1 || names[0] != "today" {
		t.Errorf("names = %v, want [today]", names)
	}
}

func TestRemoveFromAllLists_personal(t *testing.T) {
	s := newTestStore(t)
	l := list.List{Sections: []list.Section{{Items: []string{"uuid-a", "uuid-b"}}}}
	s.WriteList("", "today", l)

	if err := s.RemoveFromAllLists("", "uuid-a"); err != nil {
		t.Fatalf("RemoveFromAllLists personal: %v", err)
	}

	got, _ := s.ReadList("", "today")
	uuids := list.AllUUIDs(got)
	for _, u := range uuids {
		if u == "uuid-a" {
			t.Error("uuid-a still present after removal")
		}
	}
}

func TestOpen_autoInit(t *testing.T) {
	// Point HOME at an empty temp dir — store doesn't exist yet.
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", orig)

	s, err := Open()
	if err != nil {
		t.Fatalf("Open on fresh HOME: %v", err)
	}
	if s == nil {
		t.Fatal("got nil store")
	}

	// Store directory must now exist.
	storePath := filepath.Join(dir, ".bliss2")
	if _, err := os.Stat(storePath); err != nil {
		t.Errorf("store dir not created: %v", err)
	}

	// Verify todos/ dir exists to confirm Init() was called.
	todosDir := filepath.Join(storePath, "todos")
	if _, err := os.Stat(todosDir); err != nil {
		t.Errorf("todos dir not created by Init: %v", err)
	}
}

func TestWriteReadContextMeta_withPath(t *testing.T) {
	s := newTestStore(t)
	uuid := "ctx-meta-test"

	if err := s.WriteContextMeta(uuid, "My Project", "/home/user/my-project"); err != nil {
		t.Fatalf("WriteContextMeta: %v", err)
	}

	name, path, err := s.ReadContextMeta(uuid)
	if err != nil {
		t.Fatalf("ReadContextMeta: %v", err)
	}
	if name != "My Project" {
		t.Errorf("name = %q, want %q", name, "My Project")
	}
	if path != "/home/user/my-project" {
		t.Errorf("path = %q, want %q", path, "/home/user/my-project")
	}
}

func TestReadContextMeta_noPath(t *testing.T) {
	// Old format: name only, no path line. Should return name and empty path.
	s := newTestStore(t)
	uuid := "ctx-old-format"

	dir := s.ContextDir(uuid)
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "meta.md"), []byte("# Old Name\n"), 0644)

	name, path, err := s.ReadContextMeta(uuid)
	if err != nil {
		t.Fatalf("ReadContextMeta: %v", err)
	}
	if name != "Old Name" {
		t.Errorf("name = %q, want %q", name, "Old Name")
	}
	if path != "" {
		t.Errorf("path = %q, want empty", path)
	}
}
