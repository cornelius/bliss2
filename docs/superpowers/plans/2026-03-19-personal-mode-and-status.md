# Personal Mode and bliss status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make all bliss commands work without `bliss init` (personal mode), add `bliss status` to replace `bliss contexts`, and extend the store to support personal todos at `~/.bliss2/todos/`.

**Architecture:** The store's empty-string contextUUID convention (already used for personal lists) is extended to todos. `store.Open()` auto-inits on first use. `bliss status` is a new non-interactive command that shows context breakdown, other contexts with paths, and git sync state. All CLI commands ignore `FindContext` errors and fall back to personal mode.

**Tech Stack:** Go, cobra, bubbletea, lipgloss, go-git

---

## File Structure

**Modified:**
- `internal/store/store.go` — `TodosDir`, `getCreationTimes`, `ListNames`, `Init`, `Open`, `WriteContextMeta`, `ReadContextMeta`, `FindTodo`, new `GitSyncStatus`
- `internal/store/store_test.go` — new tests for personal mode store ops, context meta, git sync
- `cmd/bliss/main.go` — personal mode fallback in all commands, `bliss list --all`, `bliss init` home guard + path, new `statusCmd`, remove `contextsCmd`
- `cmd/bliss/main_test.go` — new tests for personal mode CLI, `--all`, `status`

---

## Task 1: Store — personal todos directory

Extend `TodosDir`, `getCreationTimes`, `ListNames`, and `Init` to support `contextUUID=""` for personal mode.

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/store/store_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run "TestWriteReadTodo_personal|TestListTodos_personal|TestDeleteTodo_personal|TestListNames_personal|TestRemoveFromAllLists_personal"
```

Expected: FAIL (personal todos dir not handled)

- [ ] **Step 3: Update `newTestStore` to create the `todos/` dir**

In `internal/store/store_test.go`, update `newTestStore`:

```go
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()

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

	repo, err := initGitRepo(dir)
	if err != nil {
		t.Fatalf("initGitRepo: %v", err)
	}

	return &Store{path: dir, repo: repo}
}
```

- [ ] **Step 4: Update `TodosDir` to branch on empty contextUUID**

In `internal/store/store.go`, replace `TodosDir`:

```go
func (s *Store) TodosDir(uuid string) string {
	if uuid == "" {
		return filepath.Join(s.path, "todos")
	}
	return filepath.Join(s.path, "contexts", uuid, "todos")
}
```

- [ ] **Step 5: Update `getCreationTimes` to branch on empty contextUUID**

In `internal/store/store.go`, replace the `prefix` assignment in `getCreationTimes`:

```go
var prefix string
if contextUUID == "" {
	prefix = "todos" + string(filepath.Separator)
} else {
	prefix = filepath.Join("contexts", contextUUID, "todos") + string(filepath.Separator)
}
```

- [ ] **Step 6: Update `ListNames` to branch on empty contextUUID**

In `internal/store/store.go`, replace `ListNames`:

```go
func (s *Store) ListNames(contextUUID string) ([]string, error) {
	if contextUUID == "" {
		return listNamesInDir(s.PersonalListsDir())
	}
	return listNamesInDir(s.ContextListsDir(contextUUID))
}
```

- [ ] **Step 7: Update `Init` to create `todos/` dir at store root**

In `internal/store/store.go`, update the `dirs` slice in `Init`:

```go
dirs := []string{
	filepath.Join(path, "contexts"),
	filepath.Join(path, "lists"),
	filepath.Join(path, "todos"),
}
```

- [ ] **Step 8: Run tests to verify they pass**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run "TestWriteReadTodo_personal|TestListTodos_personal|TestDeleteTodo_personal|TestListNames_personal|TestRemoveFromAllLists_personal"
```

Expected: PASS

- [ ] **Step 9: Run all store tests**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v
```

Expected: all PASS

- [ ] **Step 10: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): support personal mode (empty contextUUID) for todos and list names"
```

---

## Task 2: Store — Open auto-init

`store.Open()` currently errors if the store directory doesn't exist. In personal mode, the first command a user runs must work without calling `bliss init` first.

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/store/store_test.go`:

```go
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
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run TestOpen_autoInit
```

Expected: FAIL — "store not found"

- [ ] **Step 3: Update `Open` to auto-init**

In `internal/store/store.go`, replace the `Open` function:

```go
func Open() (*Store, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Init()
	}

	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening store git repo at %s: %w", path, err)
	}

	return &Store{path: path, repo: repo}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run TestOpen_autoInit
```

Expected: PASS

- [ ] **Step 5: Run all store tests**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): auto-init store on first Open so personal mode works without bliss init"
```

---

## Task 3: Store — extended context metadata (name + path)

`meta.md` currently stores only a context name. Extend it to store the filesystem path on line 2. This enables `bliss status` to show and validate context paths.

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/store/store_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run "TestWriteReadContextMeta_withPath|TestReadContextMeta_noPath"
```

Expected: FAIL (signature mismatch)

- [ ] **Step 3: Update `WriteContextMeta` signature and implementation**

In `internal/store/store.go`, replace `WriteContextMeta`:

```go
func (s *Store) WriteContextMeta(uuid, name, path string) error {
	dir := s.ContextDir(uuid)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating context dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "todos"), 0755); err != nil {
		return fmt.Errorf("creating todos dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "lists"), 0755); err != nil {
		return fmt.Errorf("creating lists dir: %w", err)
	}

	metaPath := filepath.Join(dir, "meta.md")
	content := name + "\n" + path + "\n"
	return os.WriteFile(metaPath, []byte(content), 0644)
}
```

- [ ] **Step 4: Update `ReadContextMeta` signature and implementation**

In `internal/store/store.go`, replace `ReadContextMeta`:

```go
func (s *Store) ReadContextMeta(uuid string) (name, path string, err error) {
	data, err := os.ReadFile(filepath.Join(s.ContextDir(uuid), "meta.md"))
	if err != nil {
		return "", "", err
	}
	lines := strings.SplitN(strings.TrimRight(string(data), "\n"), "\n", 2)
	name = strings.TrimPrefix(lines[0], "# ") // handle old "# name" format
	if len(lines) > 1 {
		path = lines[1]
	}
	return name, path, nil
}
```

- [ ] **Step 5: Fix the one call site in `main.go`**

In `cmd/bliss/main.go`, `initCmd` calls `s.WriteContextMeta(contextUUID, name)`. Update it to pass `cwd` as the third argument:

```go
if err := s.WriteContextMeta(contextUUID, name, cwd); err != nil {
```

In `contextsCmd`, `s.ReadContextMeta(uuid)` currently unpacks `(name, err)`. Update it to:

```go
name, _, err := s.ReadContextMeta(uuid)
```

- [ ] **Step 6: Run tests to verify they pass**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run "TestWriteReadContextMeta_withPath|TestReadContextMeta_noPath"
```

Expected: PASS

- [ ] **Step 7: Run all tests**

```
cd /Users/cs/git/bliss2 && go test ./...
```

Expected: all PASS

- [ ] **Step 8: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go cmd/bliss/main.go
git commit -m "feat(store): store and read context init path in meta.md"
```

---

## Task 4: Store — FindTodo searches personal todos

`FindTodo` currently only searches context todos. Extend it to check personal todos first.

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/store/store_test.go`:

```go
func TestFindTodo_personal(t *testing.T) {
	s := newTestStore(t)

	t1 := todo.Todo{UUID: "find-personal-1", Title: "Personal item"}
	if err := s.WriteTodo("", t1); err != nil {
		t.Fatalf("WriteTodo personal: %v", err)
	}

	got, err := s.FindTodo(t1.UUID)
	if err != nil {
		t.Fatalf("FindTodo: %v", err)
	}
	if got.Title != t1.Title {
		t.Errorf("Title = %q, want %q", got.Title, t1.Title)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run TestFindTodo_personal
```

Expected: FAIL — "todo not found in any context"

- [ ] **Step 3: Update `FindTodo`**

In `internal/store/store.go`, replace `FindTodo`:

```go
// FindTodo searches personal todos first, then all context todos.
// Used when reading personal lists, which may reference todos from any context.
func (s *Store) FindTodo(todoUUID string) (todo.Todo, error) {
	// Check personal todos first.
	if t, err := s.ReadTodo("", todoUUID); err == nil {
		return t, nil
	}

	uuids, err := s.ListContextUUIDs()
	if err != nil {
		return todo.Todo{}, err
	}
	for _, contextUUID := range uuids {
		t, err := s.ReadTodo(contextUUID, todoUUID)
		if err == nil {
			return t, nil
		}
	}
	return todo.Todo{}, fmt.Errorf("todo %s not found", todoUUID)
}
```

- [ ] **Step 4: Run test to verify it passes**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run TestFindTodo_personal
```

Expected: PASS

- [ ] **Step 5: Run all store tests**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): FindTodo searches personal todos before context todos"
```

---

## Task 5: Store — GitSyncStatus

New method returning remote name and ahead/behind commit counts.

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/store/store_test.go`:

```go
func TestGitSyncStatus_noRemote(t *testing.T) {
	s := newTestStore(t)

	remote, ahead, behind, err := s.GitSyncStatus()
	if err != nil {
		t.Fatalf("GitSyncStatus: %v", err)
	}
	if remote != "" {
		t.Errorf("remote = %q, want empty", remote)
	}
	if ahead != 0 || behind != 0 {
		t.Errorf("ahead=%d behind=%d, want 0 0", ahead, behind)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run TestGitSyncStatus_noRemote
```

Expected: FAIL — method does not exist

- [ ] **Step 3: Add `GitSyncStatus` to `store.go`**

Add the import `"github.com/go-git/go-git/v5/plumbing"` to the import block, then add at the end of the file:

```go
// GitSyncStatus returns the remote name and how many commits the local branch
// is ahead of and behind its remote tracking branch.
// Returns ("", 0, 0, nil) if the store has no remote configured.
func (s *Store) GitSyncStatus() (remote string, ahead, behind int, err error) {
	remotes, err := s.repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return "", 0, 0, nil
	}
	remote = remotes[0].Config().Name

	headRef, err := s.repo.Head()
	if err != nil {
		return remote, 0, 0, nil
	}

	remoteRefName := plumbing.NewRemoteReferenceName(remote, headRef.Name().Short())
	remoteRef, err := s.repo.Reference(remoteRefName, true)
	if err != nil {
		// Remote ref doesn't exist (never pushed).
		return remote, 0, 0, nil
	}

	if headRef.Hash() == remoteRef.Hash() {
		return remote, 0, 0, nil
	}

	// Walk local and remote histories to find divergence.
	localSet := make(map[plumbing.Hash]bool)
	localIter, err := s.repo.Log(&git.LogOptions{From: headRef.Hash()})
	if err == nil {
		localIter.ForEach(func(c *object.Commit) error {
			localSet[c.Hash] = true
			return nil
		})
	}

	remoteSet := make(map[plumbing.Hash]bool)
	remoteIter, err := s.repo.Log(&git.LogOptions{From: remoteRef.Hash()})
	if err == nil {
		remoteIter.ForEach(func(c *object.Commit) error {
			remoteSet[c.Hash] = true
			return nil
		})
	}

	for h := range localSet {
		if !remoteSet[h] {
			ahead++
		}
	}
	for h := range remoteSet {
		if !localSet[h] {
			behind++
		}
	}

	return remote, ahead, behind, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v -run TestGitSyncStatus_noRemote
```

Expected: PASS

- [ ] **Step 5: Run all store tests**

```
cd /Users/cs/git/bliss2 && go test ./internal/store/ -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): add GitSyncStatus returning remote ahead/behind counts"
```

---

## Task 6: CLI — personal mode for all commands

Remove the hard error when no context is found. All commands fall back to personal mode (`contextUUID=""`).

**Files:**
- Modify: `cmd/bliss/main.go`
- Modify: `cmd/bliss/main_test.go`

- [ ] **Step 1: Write failing tests**

Add to `cmd/bliss/main_test.go`:

```go
func TestPersonalMode_addAndList(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir() // no bliss init

	out, err := bliss(t, dir, env, "add", "Buy oat milk")
	if err != nil {
		t.Fatalf("add in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Buy oat milk") {
		t.Errorf("add output %q missing title", out)
	}

	out, err = bliss(t, dir, env, "list")
	if err != nil {
		t.Fatalf("list in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Buy oat milk") {
		t.Errorf("list output %q missing title", out)
	}
}

func TestPersonalMode_addToList(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "add", "Urgent task", "-l", "today")
	if err != nil {
		t.Fatalf("add to list in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "[today]") {
		t.Errorf("output %q missing list name", out)
	}

	out, err = bliss(t, dir, env, "list", "today")
	if err != nil {
		t.Fatalf("list today in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Urgent task") {
		t.Errorf("list today output %q missing title", out)
	}
}

func TestPersonalMode_done(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Complete me")
	bliss(t, dir, env, "list")

	out, err := bliss(t, dir, env, "done", "1")
	if err != nil {
		t.Fatalf("done in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Done:") {
		t.Errorf("done output %q missing confirmation", out)
	}
}

func TestPersonalMode_move(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Move me")
	bliss(t, dir, env, "list")

	out, err := bliss(t, dir, env, "move", "1", "-l", "today")
	if err != nil {
		t.Fatalf("move in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Moved to [today]") {
		t.Errorf("move output %q missing confirmation", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run "TestPersonalMode"
```

Expected: FAIL — "no .bliss-context found"

- [ ] **Step 3: Update all commands to ignore `FindContext` error**

In `cmd/bliss/main.go`, change every command that does:
```go
contextUUID, _, err := blisscontext.FindContext(cwd)
if err != nil {
    return err
}
```
to:
```go
contextUUID, _, _ := blisscontext.FindContext(cwd)
```

Commands to update: `addCmd`, `listCmd`, `doneCmd`, `moveCmd`, `checkCmd`, `groomCmd`, `historyCmd`.

Note: `historyCmd` without `--all` calls `FindContext` and returns the error — it must be fixed. `ReadHistory` already handles empty contextUUID (returns full history), so the only change needed is ignoring the `FindContext` error.

`contextsCmd` does NOT need this change — it will be removed in Task 9.

- [ ] **Step 4: Run tests to verify they pass**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run "TestPersonalMode"
```

Expected: PASS

- [ ] **Step 5: Run all CLI tests**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v
```

Expected: all PASS (existing tests still call `bliss init`, which is fine)

- [ ] **Step 6: Commit**

```bash
git add cmd/bliss/main.go cmd/bliss/main_test.go
git commit -m "feat: personal mode — all commands work without bliss init"
```

---

## Task 7: CLI — bliss list --all

Add `--all` flag to `bliss list` that shows todos from all contexts plus personal, without writing a session file.

**Files:**
- Modify: `cmd/bliss/main.go`
- Modify: `cmd/bliss/main_test.go`

- [ ] **Step 1: Write failing test**

Add to `cmd/bliss/main_test.go`:

```go
func TestList_all(t *testing.T) {
	home, env := blissEnv(t)

	// Create two project dirs with contexts.
	proj1 := filepath.Join(home, "proj1")
	proj2 := filepath.Join(home, "proj2")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	bliss(t, proj1, env, "init", "--name", "Project One")
	bliss(t, proj2, env, "init", "--name", "Project Two")
	bliss(t, proj1, env, "add", "Todo in proj1")
	bliss(t, proj2, env, "add", "Todo in proj2")

	// Run --all from a neutral directory (home).
	out, err := bliss(t, home, env, "list", "--all")
	if err != nil {
		t.Fatalf("list --all: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Todo in proj1") {
		t.Errorf("output missing proj1 todo: %s", out)
	}
	if !strings.Contains(out, "Todo in proj2") {
		t.Errorf("output missing proj2 todo: %s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run TestList_all
```

Expected: FAIL — unknown flag `--all`

- [ ] **Step 3: Add `--all` to `listCmd`**

In `cmd/bliss/main.go`, update `listCmd` to accept an `all` bool flag. When `all` is true:
- Iterate all context UUIDs via `s.ListContextUUIDs()`
- For each, read meta for name, show todos under a `[name]` header
- Show personal lists at the end
- Do NOT call `s.WriteSession(session)` — return after printing

```go
func listCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "list [list-name]",
		Short: "Show todos with position numbers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			contextUUID, _, _ := blisscontext.FindContext(cwd)

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			// --all: show every context plus personal, no session.
			if all {
				return listAll(s)
			}

			// existing logic below (unchanged except FindContext error removed above)
			// ...
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show todos from all contexts")
	return cmd
}
```

Add a `listAll` helper function in `cmd/bliss/main.go`:

```go
func listAll(s *store.Store) error {
	contextUUIDs, err := s.ListContextUUIDs()
	if err != nil {
		return err
	}

	first := true
	printHeader := func(header string) {
		if !first {
			fmt.Println()
		}
		first = false
		fmt.Println(styleListHeader.Render("  " + header))
	}

	for _, uuid := range contextUUIDs {
		name, _, _ := s.ReadContextMeta(uuid)
		todos, err := s.ListTodos(uuid)
		if err != nil || len(todos) == 0 {
			continue
		}
		printHeader(name)
		for _, t := range todos {
			fmt.Printf("    %s\n", t.Title)
		}
	}

	// Personal todos.
	personalTodos, err := s.ListTodos("")
	if err == nil && len(personalTodos) > 0 {
		printHeader("personal")
		for _, t := range personalTodos {
			fmt.Printf("    %s\n", t.Title)
		}
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run TestList_all
```

Expected: PASS

- [ ] **Step 5: Run all CLI tests**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/bliss/main.go cmd/bliss/main_test.go
git commit -m "feat: add bliss list --all to show todos across all contexts"
```

---

## Task 8: CLI — bliss init home dir guard + path tracking

Guard against `bliss init` in the home directory. `initCmd` already passes `cwd` to `WriteContextMeta` (fixed in Task 3 Step 5); verify it and add the guard.

**Files:**
- Modify: `cmd/bliss/main.go`
- Modify: `cmd/bliss/main_test.go`

- [ ] **Step 1: Write failing test**

Add to `cmd/bliss/main_test.go`:

```go
func TestInit_homeDirectoryGuard(t *testing.T) {
	home, env := blissEnv(t)

	out, err := bliss(t, home, env, "init")
	if err == nil {
		t.Fatalf("expected error running bliss init in home dir, got: %s", out)
	}
	if !strings.Contains(out, "home directory") {
		t.Errorf("error output %q does not mention home directory", out)
	}
}

func TestInit_storesPath(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	if _, err := bliss(t, proj, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	// status should show the path
	out, err := bliss(t, proj, env, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, proj) {
		t.Errorf("status output %q does not contain project path %q", out, proj)
	}
}
```

Note: `TestInit_storesPath` depends on `bliss status` existing (Task 9). Add it to the file now but it will only be run in Task 9 Step 5.

- [ ] **Step 2: Run home guard test to verify it fails**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run TestInit_homeDirectoryGuard
```

Expected: FAIL — init succeeds when it shouldn't

- [ ] **Step 3: Add home directory guard to `initCmd`**

In `cmd/bliss/main.go`, add at the start of `initCmd`'s `RunE`, after getting `cwd`:

```go
home, err := os.UserHomeDir()
if err != nil {
    return fmt.Errorf("getting home directory: %w", err)
}
if cwd == home {
    return fmt.Errorf("cannot init a context in the home directory — use personal mode instead")
}
```

- [ ] **Step 4: Run test to verify it passes**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run TestInit_homeDirectoryGuard
```

Expected: PASS

- [ ] **Step 5: Run only the home guard test (not storesPath — that needs Task 9)**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run TestInit_homeDirectoryGuard
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/bliss/main.go cmd/bliss/main_test.go
git commit -m "feat: guard bliss init against home directory"
```

---

## Task 9: CLI — bliss status, remove bliss contexts

Add `statusCmd`, wire it into `rootCmd`, and remove `contextsCmd`.

**Files:**
- Modify: `cmd/bliss/main.go`
- Modify: `cmd/bliss/main_test.go`

- [ ] **Step 1: Write failing tests**

Add to `cmd/bliss/main_test.go`:

```go
func TestStatus_personalMode(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Personal task one")
	bliss(t, dir, env, "add", "Personal task two")

	out, err := bliss(t, dir, env, "status")
	if err != nil {
		t.Fatalf("status in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "personal mode") {
		t.Errorf("output %q missing 'personal mode'", out)
	}
	if !strings.Contains(out, "inbox") {
		t.Errorf("output %q missing inbox count", out)
	}
	if !strings.Contains(out, "no remote") {
		t.Errorf("output %q missing git sync line", out)
	}
}

func TestStatus_insideContext(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "My Project")
	bliss(t, proj, env, "add", "Context task")

	out, err := bliss(t, proj, env, "status")
	if err != nil {
		t.Fatalf("status inside context: %v\n%s", err, out)
	}
	if !strings.Contains(out, "My Project") {
		t.Errorf("output %q missing context name", out)
	}
	if !strings.Contains(out, proj) {
		t.Errorf("output %q missing context path", out)
	}
	if !strings.Contains(out, "inbox") {
		t.Errorf("output %q missing inbox", out)
	}
	if !strings.Contains(out, "no remote") {
		t.Errorf("output %q missing git sync line", out)
	}
}

func TestStatus_stalePath(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "willmove")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "Will Move")

	// Remove the project dir to simulate a moved/deleted context.
	os.RemoveAll(proj)

	// Run status from home (personal mode) — should show stale path.
	out, err := bliss(t, home, env, "status")
	if err != nil {
		t.Fatalf("status with stale: %v\n%s", err, out)
	}
	if !strings.Contains(out, "stale") {
		t.Errorf("output %q missing stale indicator", out)
	}
}

func TestContexts_removed(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "contexts")
	// Should error — command no longer exists.
	if err == nil {
		t.Errorf("expected error for removed 'contexts' command, got: %s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run "TestStatus|TestContexts_removed"
```

Expected: FAIL — `status` command does not exist, `contexts` command still exists

- [ ] **Step 3: Add `statusCmd` to `main.go`**

Add this function to `cmd/bliss/main.go`:

```go
func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show context status and git sync",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			activeUUID, _, _ := blisscontext.FindContext(cwd)

			s, err := store.Open()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}

			// --- Current context or personal mode ---
			if activeUUID == "" {
				fmt.Println(styleActive.Render("  personal mode"))
			} else {
				name, path, _ := s.ReadContextMeta(activeUUID)
				fmt.Printf("%s  %s\n", styleActive.Render("* "+name), styleMuted.Render(path))
			}

			// Per-list breakdown for current context / personal mode.
			counts := statusListCounts(s, activeUUID)
			for _, lc := range counts {
				fmt.Printf("  %-12s %d\n", lc.name, lc.count)
			}

			// --- Other contexts ---
			contextUUIDs, err := s.ListContextUUIDs()
			if err != nil {
				return err
			}
			printedOtherHeader := false
			for _, uuid := range contextUUIDs {
				if uuid == activeUUID {
					continue
				}
				if !printedOtherHeader {
					fmt.Println()
					printedOtherHeader = true
				}
				name, path, _ := s.ReadContextMeta(uuid)
				pathDisplay := styleMuted.Render(path)
				if path != "" && !isContextPathFresh(path, uuid) {
					pathDisplay = styleMuted.Render("(stale path)")
				}
				compact := compactListSummary(s, uuid)
				fmt.Printf("  %-24s  %s  %s\n", name, pathDisplay, styleMuted.Render(compact))
			}

			// --- Personal section (when inside a context) ---
			if activeUUID != "" {
				personalCounts := statusListCounts(s, "")
				if len(personalCounts) > 0 {
					fmt.Println()
					fmt.Println(styleMuted.Render("  personal"))
					parts := []string{}
					for _, lc := range personalCounts {
						parts = append(parts, fmt.Sprintf("%s %d", lc.name, lc.count))
					}
					fmt.Printf("  %s\n", strings.Join(parts, "  "))
				}
			}

			// --- Git sync ---
			fmt.Println()
			remote, ahead, behind, _ := s.GitSyncStatus()
			if remote == "" {
				fmt.Println(styleMuted.Render("  store  no remote"))
			} else if ahead == 0 && behind == 0 {
				fmt.Printf("%s\n", styleMuted.Render(fmt.Sprintf("  store  synced  (%s)", remote)))
			} else {
				var parts []string
				if ahead > 0 {
					parts = append(parts, fmt.Sprintf("↑%d ahead", ahead))
				}
				if behind > 0 {
					parts = append(parts, fmt.Sprintf("↓%d behind", behind))
				}
				fmt.Printf("%s\n", styleMuted.Render(fmt.Sprintf("  store  %s  %s", strings.Join(parts, "  "), remote)))
			}

			return nil
		},
	}
}

type listCount struct {
	name  string
	count int
}

// statusListCounts returns per-list todo counts for a context (or personal mode if "").
// Lists with zero todos are omitted.
func statusListCounts(s *store.Store, contextUUID string) []listCount {
	var names []string
	if contextUUID == "" {
		names, _ = s.PersonalListNames()
	} else {
		names, _ = s.ListNames(contextUUID)
	}

	var counts []listCount
	for _, name := range names {
		l, err := s.ReadList(contextUUID, name)
		if err != nil {
			continue
		}
		n := len(list.AllUUIDs(l))
		if n > 0 {
			counts = append(counts, listCount{name, n})
		}
	}

	inboxCount := statusInboxCount(s, contextUUID)
	if inboxCount > 0 {
		counts = append(counts, listCount{"inbox", inboxCount})
	}

	return counts
}

func statusInboxCount(s *store.Store, contextUUID string) int {
	todos, err := getInboxTodos(s, contextUUID)
	if err != nil {
		return 0
	}
	return len(todos)
}

func compactListSummary(s *store.Store, contextUUID string) string {
	counts := statusListCounts(s, contextUUID)
	parts := make([]string, 0, len(counts))
	for _, lc := range counts {
		parts = append(parts, fmt.Sprintf("%s %d", lc.name, lc.count))
	}
	return strings.Join(parts, "  ")
}

// isContextPathFresh checks whether a path still contains a .bliss-context pointing to uuid.
func isContextPathFresh(path, uuid string) bool {
	data, err := os.ReadFile(filepath.Join(path, ".bliss-context"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == uuid
}
```

- [ ] **Step 4: Wire `statusCmd` into `rootCmd` and remove `contextsCmd`**

In `cmd/bliss/main.go`, update `rootCmd`:

```go
root.AddCommand(
    initCmd(),
    addCmd(),
    listCmd(),
    doneCmd(),
    moveCmd(),
    checkCmd(),
    groomCmd(),
    statusCmd(),  // replaces contextsCmd
    historyCmd(),
)
```

Delete the entire `contextsCmd()` function.

- [ ] **Step 5: Run tests to verify they pass**

```
cd /Users/cs/git/bliss2 && go test ./cmd/bliss/ -v -run "TestStatus|TestContexts_removed|TestInit_storesPath"
```

Expected: PASS

- [ ] **Step 6: Run all tests**

```
cd /Users/cs/git/bliss2 && go test ./...
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/bliss/main.go cmd/bliss/main_test.go
git commit -m "feat: add bliss status, remove bliss contexts"
```

---

## Final verification

- [ ] **Run full test suite**

```
cd /Users/cs/git/bliss2 && go test ./...
```

Expected: all PASS, no compilation errors

- [ ] **Build binary and smoke test personal mode**

```bash
cd /Users/cs/git/bliss2 && go build -o /tmp/bliss-test ./cmd/bliss/
mkdir -p /tmp/bliss-smoke && HOME=/tmp/bliss-smoke /tmp/bliss-test add "smoke test todo"
HOME=/tmp/bliss-smoke /tmp/bliss-test list
HOME=/tmp/bliss-smoke /tmp/bliss-test status
```

Expected: todo added and listed without any `bliss init`, status shows personal mode

- [ ] **Final commit message for the feature branch (if applicable)**

All individual commits are already made. No squash needed.
