// Package store is the single owner of all store I/O and git operations.
// No other package constructs store paths or reads/writes store files.
package store

import (
	"bliss/internal/list"
	"bliss/internal/todo"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Store struct {
	path string
	repo *git.Repository
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".bliss2"), nil
}

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

func Init() (*Store, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}

	dirs := []string{
		filepath.Join(path, "contexts"),
		filepath.Join(path, "lists"),
		filepath.Join(path, "todos"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	repo, err := initGitRepo(path)
	if err != nil {
		return nil, err
	}

	return &Store{path: path, repo: repo}, nil
}

func initGitRepo(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		repo, err = git.PlainInit(path, false)
		if err != nil {
			return nil, fmt.Errorf("initializing git repo: %w", err)
		}
	}
	return repo, nil
}

func (s *Store) ContextDir(uuid string) string {
	return filepath.Join(s.path, "contexts", uuid)
}

func (s *Store) TodosDir(uuid string) string {
	if uuid == "" {
		return filepath.Join(s.path, "todos")
	}
	return filepath.Join(s.path, "contexts", uuid, "todos")
}

func (s *Store) ContextListsDir(uuid string) string {
	return filepath.Join(s.path, "contexts", uuid, "lists")
}

func (s *Store) PersonalListsDir() string {
	return filepath.Join(s.path, "lists")
}

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

// ListContextUUIDs returns the UUIDs of all contexts in the store.
func (s *Store) ListContextUUIDs() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.path, "contexts"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading contexts dir: %w", err)
	}
	var uuids []string
	for _, e := range entries {
		if e.IsDir() {
			uuids = append(uuids, e.Name())
		}
	}
	return uuids, nil
}

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

func (s *Store) WriteTodo(contextUUID string, t todo.Todo) error {
	todosDir := s.TodosDir(contextUUID)
	if err := os.MkdirAll(todosDir, 0755); err != nil {
		return fmt.Errorf("creating todos dir: %w", err)
	}
	filePath := filepath.Join(todosDir, t.UUID+".md")
	return os.WriteFile(filePath, []byte(todo.Format(t)), 0644)
}

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

func (s *Store) ReadTodo(contextUUID, todoUUID string) (todo.Todo, error) {
	filePath := filepath.Join(s.TodosDir(contextUUID), todoUUID+".md")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return todo.Todo{}, fmt.Errorf("reading todo %s: %w", todoUUID, err)
	}
	t, err := todo.Parse(string(data))
	if err != nil {
		return todo.Todo{}, fmt.Errorf("parsing todo %s: %w", todoUUID, err)
	}
	t.UUID = todoUUID
	return t, nil
}

func (s *Store) DeleteTodo(contextUUID, todoUUID string) error {
	filePath := filepath.Join(s.TodosDir(contextUUID), todoUUID+".md")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("deleting todo %s: %w", todoUUID, err)
	}
	return nil
}

// ListTodos returns todos sorted by creation time (oldest first), derived from git history.
func (s *Store) ListTodos(contextUUID string) ([]todo.Todo, error) {
	todosDir := s.TodosDir(contextUUID)
	entries, err := os.ReadDir(todosDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading todos dir: %w", err)
	}

	creationTimes := s.getCreationTimes(contextUUID)

	var todos []todo.Todo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		uuid := strings.TrimSuffix(entry.Name(), ".md")
		t, err := s.ReadTodo(contextUUID, uuid)
		if err != nil {
			continue
		}
		if ct, ok := creationTimes[uuid]; ok {
			t.CreatedAt = ct
		} else {
			info, err := entry.Info()
			if err == nil {
				t.CreatedAt = info.ModTime()
			}
		}
		todos = append(todos, t)
	}

	sort.Slice(todos, func(i, j int) bool {
		return todos[i].CreatedAt.Before(todos[j].CreatedAt)
	})

	return todos, nil
}

// getCreationTimes walks git history to find the earliest commit touching each todo file.
func (s *Store) getCreationTimes(contextUUID string) map[string]time.Time {
	result := make(map[string]time.Time)

	iter, err := s.repo.Log(&git.LogOptions{Order: git.LogOrderCommitterTime})
	if err != nil {
		return result
	}
	defer iter.Close()

	var prefix string
	if contextUUID == "" {
		prefix = "todos" + string(filepath.Separator)
	} else {
		prefix = filepath.Join("contexts", contextUUID, "todos") + string(filepath.Separator)
	}

	iter.ForEach(func(c *object.Commit) error {
		files, err := c.Files()
		if err != nil {
			return nil
		}
		files.ForEach(func(f *object.File) error {
			if strings.HasPrefix(f.Name, prefix) {
				uuid := strings.TrimSuffix(filepath.Base(f.Name), ".md")
				// Iterating newest-first; keep overwriting to end up with the oldest commit.
				result[uuid] = c.Author.When
			}
			return nil
		})
		return nil
	})

	return result
}

// WriteList writes to the context lists dir, or personal lists dir when contextUUID is empty.
func (s *Store) WriteList(contextUUID, listName string, l list.List) error {
	var listsDir string
	if contextUUID == "" {
		listsDir = s.PersonalListsDir()
	} else {
		listsDir = s.ContextListsDir(contextUUID)
	}
	if err := os.MkdirAll(listsDir, 0755); err != nil {
		return fmt.Errorf("creating lists dir: %w", err)
	}
	filePath := filepath.Join(listsDir, listName+".txt")
	return os.WriteFile(filePath, []byte(list.Format(l)), 0644)
}

// ReadList reads from the context lists dir, or personal lists dir when contextUUID is empty.
// Returns an empty list if the file does not exist.
func (s *Store) ReadList(contextUUID, listName string) (list.List, error) {
	var listsDir string
	if contextUUID == "" {
		listsDir = s.PersonalListsDir()
	} else {
		listsDir = s.ContextListsDir(contextUUID)
	}
	filePath := filepath.Join(listsDir, listName+".txt")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return list.List{Sections: []list.Section{{}}}, nil
		}
		return list.List{}, fmt.Errorf("reading list %s: %w", listName, err)
	}
	return list.Parse(string(data))
}

func (s *Store) ListNames(contextUUID string) ([]string, error) {
	if contextUUID == "" {
		return listNamesInDir(s.PersonalListsDir())
	}
	return listNamesInDir(s.ContextListsDir(contextUUID))
}

func (s *Store) PersonalListNames() ([]string, error) {
	return listNamesInDir(s.PersonalListsDir())
}

func listNamesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading lists dir: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			names = append(names, strings.TrimSuffix(entry.Name(), ".txt"))
		}
	}
	return names, nil
}

func (s *Store) WriteSession(mapping map[int]string) error {
	sessionPath := filepath.Join(s.path, "session.txt")
	var sb strings.Builder
	keys := make([]int, 0, len(mapping))
	for k := range mapping {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%d %s\n", k, mapping[k]))
	}
	return os.WriteFile(sessionPath, []byte(sb.String()), 0644)
}

func (s *Store) ReadSession() (map[int]string, error) {
	sessionPath := filepath.Join(s.path, "session.txt")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no session found (run 'bliss list' first)")
		}
		return nil, fmt.Errorf("reading session: %w", err)
	}

	mapping := make(map[int]string)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		mapping[n] = parts[1]
	}
	return mapping, nil
}

// RemoveFromList operates on the personal lists dir when contextUUID is empty.
func (s *Store) RemoveFromList(contextUUID, listName, uuid string) error {
	l, err := s.ReadList(contextUUID, listName)
	if err != nil {
		return err
	}
	list.Remove(&l, uuid)
	return s.WriteList(contextUUID, listName, l)
}

func (s *Store) RemoveFromAllLists(contextUUID, uuid string) error {
	names, err := s.ListNames(contextUUID)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := s.RemoveFromList(contextUUID, name, uuid); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Commit(message string) error {
	wt, err := s.repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	if err := wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("staging changes: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return fmt.Errorf("getting status: %w", err)
	}
	if status.IsClean() {
		return nil
	}

	_, err = wt.Commit("bliss: "+message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "bliss",
			Email: "bliss@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	return nil
}

// HistoryEntry is one entry from the git log.
type HistoryEntry struct {
	Time    time.Time
	Message string
}

// ReadHistory returns git log entries, optionally filtered to a single context.
// If contextUUID is empty, all entries are returned.
func (s *Store) ReadHistory(contextUUID string) ([]HistoryEntry, error) {
	iter, err := s.repo.Log(&git.LogOptions{Order: git.LogOrderCommitterTime})
	if err != nil {
		return nil, fmt.Errorf("reading git log: %w", err)
	}
	defer iter.Close()

	var entries []HistoryEntry
	prefix := filepath.Join("contexts", contextUUID) + string(filepath.Separator)

	err = iter.ForEach(func(c *object.Commit) error {
		if contextUUID == "" {
			entries = append(entries, HistoryEntry{Time: c.Author.When, Message: c.Message})
			return nil
		}
		// Filter to commits touching this context's directory.
		files, err := c.Files()
		if err != nil {
			return nil
		}
		return files.ForEach(func(f *object.File) error {
			if strings.HasPrefix(f.Name, prefix) {
				entries = append(entries, HistoryEntry{Time: c.Author.When, Message: c.Message})
				return fmt.Errorf("stop") // sentinel to stop inner iteration
			}
			return nil
		})
	})
	// Ignore the sentinel error used to break inner iteration.
	if err != nil && err.Error() != "stop" {
		return nil, err
	}

	return entries, nil
}

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
