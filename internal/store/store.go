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

	"os/exec"

	"gopkg.in/yaml.v3"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Store struct {
	path string
	repo *git.Repository
}

func storePath() (string, error) {
	if p := os.Getenv("BLISS_STORE"); p != "" {
		return p, nil
	}
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
		if err == git.ErrRepositoryNotExists {
			return nil, fmt.Errorf("%s exists but is not a bliss2 store — please remove it to start fresh", path)
		}
		return nil, fmt.Errorf("opening store git repo at %s: %w", path, err)
	}

	if _, err := os.Stat(filepath.Join(path, "bliss2-was-here")); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s exists but is not a bliss2 store — please remove it to start fresh", path)
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

	markerPath := filepath.Join(path, "bliss2-was-here")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		if err := os.WriteFile(markerPath, []byte("1\n"), 0644); err != nil {
			return nil, fmt.Errorf("writing store marker: %w", err)
		}
	}

	if err := ensureGitignore(path); err != nil {
		return nil, err
	}

	repo, err := initGitRepo(path)
	if err != nil {
		return nil, err
	}

	return &Store{path: path, repo: repo}, nil
}

// ensureGitignore writes ~/.bliss2/.gitignore if it does not already exist.
// session.txt is machine-local state and must not be version-controlled.
func ensureGitignore(storePath string) error {
	p := filepath.Join(storePath, ".gitignore")
	if _, err := os.Stat(p); err == nil {
		return nil // already exists
	}
	return os.WriteFile(p, []byte("session.txt\n"), 0644)
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

func (s *Store) Path() string { return s.path }

func (s *Store) ContextDir(contextName string) string {
	return filepath.Join(s.path, "contexts", contextName)
}

func (s *Store) TodosDir(contextName string) string {
	if contextName == "" {
		return filepath.Join(s.path, "todos")
	}
	return filepath.Join(s.path, "contexts", contextName, "todos")
}

func (s *Store) ContextListsDir(contextName string) string {
	return filepath.Join(s.path, "contexts", contextName, "lists")
}

func (s *Store) PersonalListsDir() string {
	return filepath.Join(s.path, "lists")
}

type contextMeta struct {
	CreatedAt time.Time         `yaml:"created_at,omitempty"`
	Paths     map[string]string `yaml:"paths,omitempty"`
}

// ReadContextMeta returns the local path for the current machine for the given
// context name. The path is empty if this machine has never linked the context.
func (s *Store) ReadContextMeta(contextName string) (path string, err error) {
	data, err := os.ReadFile(filepath.Join(s.ContextDir(contextName), "meta.yaml"))
	if err != nil {
		return "", err
	}
	var m contextMeta
	if err := yaml.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("parsing meta.yaml: %w", err)
	}
	host, _ := os.Hostname()
	return m.Paths[host], nil
}

// ListContextNames returns the names (slugs) of all contexts in the store.
func (s *Store) ListContextNames() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.path, "contexts"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading contexts dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ContextExists reports whether a context directory exists in the store.
func (s *Store) ContextExists(contextName string) bool {
	_, err := os.Stat(s.ContextDir(contextName))
	return err == nil
}

// WriteContextMeta writes or updates meta.yaml for the given context name.
// It preserves other machines' path entries and the original created_at timestamp.
func (s *Store) WriteContextMeta(contextName, path string) error {
	dir := s.ContextDir(contextName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating context dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "todos"), 0755); err != nil {
		return fmt.Errorf("creating todos dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "lists"), 0755); err != nil {
		return fmt.Errorf("creating lists dir: %w", err)
	}

	metaPath := filepath.Join(dir, "meta.yaml")

	// Read existing meta to preserve other hosts' paths and created_at.
	var m contextMeta
	if data, err := os.ReadFile(metaPath); err == nil {
		yaml.Unmarshal(data, &m)
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	if m.Paths == nil {
		m.Paths = make(map[string]string)
	}
	host, _ := os.Hostname()
	m.Paths[host] = path

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling meta.yaml: %w", err)
	}
	return os.WriteFile(metaPath, data, 0644)
}

func (s *Store) WriteTodo(contextName string, t todo.Todo) error {
	todosDir := s.TodosDir(contextName)
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

	names, err := s.ListContextNames()
	if err != nil {
		return todo.Todo{}, err
	}
	for _, contextName := range names {
		t, err := s.ReadTodo(contextName, todoUUID)
		if err == nil {
			return t, nil
		}
	}
	return todo.Todo{}, fmt.Errorf("todo %s not found", todoUUID)
}

func (s *Store) ReadTodo(contextName, todoUUID string) (todo.Todo, error) {
	filePath := filepath.Join(s.TodosDir(contextName), todoUUID+".md")
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

func (s *Store) DeleteTodo(contextName, todoUUID string) error {
	filePath := filepath.Join(s.TodosDir(contextName), todoUUID+".md")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("deleting todo %s: %w", todoUUID, err)
	}
	return nil
}

// ListTodos returns todos sorted by creation time (oldest first), derived from git history.
func (s *Store) ListTodos(contextName string) ([]todo.Todo, error) {
	todosDir := s.TodosDir(contextName)
	entries, err := os.ReadDir(todosDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading todos dir: %w", err)
	}

	creationTimes := s.getCreationTimes(contextName)

	var todos []todo.Todo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		todoUUID := strings.TrimSuffix(entry.Name(), ".md")
		t, err := s.ReadTodo(contextName, todoUUID)
		if err != nil {
			continue
		}
		if ct, ok := creationTimes[todoUUID]; ok {
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
func (s *Store) getCreationTimes(contextName string) map[string]time.Time {
	result := make(map[string]time.Time)

	iter, err := s.repo.Log(&git.LogOptions{Order: git.LogOrderCommitterTime})
	if err != nil {
		return result
	}
	defer iter.Close()

	var prefix string
	if contextName == "" {
		prefix = "todos" + string(filepath.Separator)
	} else {
		prefix = filepath.Join("contexts", contextName, "todos") + string(filepath.Separator)
	}

	iter.ForEach(func(c *object.Commit) error {
		files, err := c.Files()
		if err != nil {
			return nil
		}
		files.ForEach(func(f *object.File) error {
			if strings.HasPrefix(f.Name, prefix) {
				todoUUID := strings.TrimSuffix(filepath.Base(f.Name), ".md")
				// Iterating newest-first; keep overwriting to end up with the oldest commit.
				result[todoUUID] = c.Author.When
			}
			return nil
		})
		return nil
	})

	return result
}

// WriteList writes to the context lists dir, or personal lists dir when contextName is empty.
func (s *Store) WriteList(contextName, listName string, l list.List) error {
	var listsDir string
	if contextName == "" {
		listsDir = s.PersonalListsDir()
	} else {
		listsDir = s.ContextListsDir(contextName)
	}
	if err := os.MkdirAll(listsDir, 0755); err != nil {
		return fmt.Errorf("creating lists dir: %w", err)
	}
	filePath := filepath.Join(listsDir, listName+".txt")
	return os.WriteFile(filePath, []byte(list.Format(l)), 0644)
}

// ReadList reads from the context lists dir, or personal lists dir when contextName is empty.
// Returns an empty list if the file does not exist.
func (s *Store) ReadList(contextName, listName string) (list.List, error) {
	var listsDir string
	if contextName == "" {
		listsDir = s.PersonalListsDir()
	} else {
		listsDir = s.ContextListsDir(contextName)
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

func (s *Store) ListNames(contextName string) ([]string, error) {
	if contextName == "" {
		return listNamesInDir(s.PersonalListsDir())
	}
	return listNamesInDir(s.ContextListsDir(contextName))
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

// RemoveFromList operates on the personal lists dir when contextName is empty.
func (s *Store) RemoveFromList(contextName, listName, uuid string) error {
	l, err := s.ReadList(contextName, listName)
	if err != nil {
		return err
	}
	list.Remove(&l, uuid)
	return s.WriteList(contextName, listName, l)
}

func (s *Store) RemoveFromAllLists(contextName, uuid string) error {
	names, err := s.ListNames(contextName)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := s.RemoveFromList(contextName, name, uuid); err != nil {
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
// ContextName identifies which context the commit belongs to, derived from the
// store paths it touched: contexts/<name>/… → that context, todos/… → personal
// (ContextName == "" and Personal == true), anything else → neither.
type HistoryEntry struct {
	Time        time.Time
	Message     string
	ContextName string // non-empty when commit touches a specific context
	Personal    bool   // true when commit touches personal todos/lists
}

const personalTodosPrefix = "todos" + string(filepath.Separator)
const contextsPrefix = "contexts" + string(filepath.Separator)

// ReadHistory returns all git log entries with context attribution derived
// from the store paths each commit touched. Callers filter by ContextUUID or
// Personal as needed.
func (s *Store) ReadHistory() ([]HistoryEntry, error) {
	iter, err := s.repo.Log(&git.LogOptions{Order: git.LogOrderCommitterTime})
	if err != nil {
		return nil, fmt.Errorf("reading git log: %w", err)
	}
	defer iter.Close()

	var entries []HistoryEntry

	err = iter.ForEach(func(c *object.Commit) error {
		entry := HistoryEntry{Time: c.Author.When, Message: c.Message}

		// Use Stats() (changed files only) rather than Files() (whole tree),
		// so attribution reflects what the commit actually touched.
		stats, err := c.Stats()
		if err != nil {
			entries = append(entries, entry)
			return nil
		}

		for _, stat := range stats {
			if strings.HasPrefix(stat.Name, contextsPrefix) {
				rest := stat.Name[len(contextsPrefix):]
				if idx := strings.Index(rest, string(filepath.Separator)); idx > 0 {
					entry.ContextName = rest[:idx]
					break
				}
			} else if strings.HasPrefix(stat.Name, personalTodosPrefix) {
				entry.Personal = true
				break
			}
		}

		entries = append(entries, entry)
		return nil
	})
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
	cfg := remotes[0].Config()
	remoteName := cfg.Name
	if len(cfg.URLs) > 0 {
		remote = cfg.URLs[0]
	} else {
		remote = remoteName
	}

	headRef, err := s.repo.Head()
	if err != nil {
		return remote, 0, 0, nil
	}

	remoteRefName := plumbing.NewRemoteReferenceName(remoteName, headRef.Name().Short())
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

// Sync fetches from the remote, pulls if behind, and pushes if ahead.
// Returns the number of commits pushed and pulled.
// Errors if no remote is configured or if the branches have diverged.
func (s *Store) Sync() (pushed, pulled int, err error) {
	remotes, err := s.repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return 0, 0, fmt.Errorf("no remote configured — add one with git in %s", s.path)
	}

	if out, ferr := exec.Command("git", "-C", s.path, "fetch").CombinedOutput(); ferr != nil {
		return 0, 0, fmt.Errorf("fetch failed: %s", strings.TrimSpace(string(out)))
	}

	// Count ahead/behind using git directly after the fetch.
	// go-git caches remote refs and does not see changes made by external git commands.
	behindOut, err := exec.Command("git", "-C", s.path, "rev-list", "HEAD..@{u}", "--count").Output()
	if err != nil {
		// No upstream tracking branch set (never pushed) — treat as in sync.
		return 0, 0, nil
	}
	aheadOut, err := exec.Command("git", "-C", s.path, "rev-list", "@{u}..HEAD", "--count").Output()
	if err != nil {
		return 0, 0, nil
	}
	behind, _ := strconv.Atoi(strings.TrimSpace(string(behindOut)))
	ahead, _ := strconv.Atoi(strings.TrimSpace(string(aheadOut)))

	if ahead > 0 && behind > 0 {
		return 0, 0, fmt.Errorf("store has diverged from remote — resolve manually with git in %s", s.path)
	}

	if behind > 0 {
		if out, perr := exec.Command("git", "-C", s.path, "pull", "--ff-only").CombinedOutput(); perr != nil {
			return 0, 0, fmt.Errorf("pull failed: %s", strings.TrimSpace(string(out)))
		}
		return 0, behind, nil
	}

	if ahead > 0 {
		if out, perr := exec.Command("git", "-C", s.path, "push").CombinedOutput(); perr != nil {
			return 0, 0, fmt.Errorf("push failed: %s", strings.TrimSpace(string(out)))
		}
		return ahead, 0, nil
	}

	return 0, 0, nil
}
