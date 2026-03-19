# Design: Personal Mode, bliss status, and Context-Free Todos

Date: 2026-03-19

## Overview

Three related changes that share a common theme: bliss should work as a plain todo manager out of the box, without requiring `bliss init`. "Personal mode" is the term for this context-free operation. Alongside this, `bliss status` is introduced as a situational awareness command that replaces `bliss contexts`.

## Goals

- All bliss commands work without any `.bliss-context` file present (personal mode).
- `bliss add` outside a context writes to a personal inbox, not an error.
- `bliss status` gives a at-a-glance view of the current context (or personal mode), other contexts, and git sync state.
- `bliss contexts` is removed; its functionality is absorbed into `bliss status`.
- Context init path is tracked so `bliss status` can detect stale paths.

## Non-Goals

- `bliss doctor` (future command to scan filesystem and fix stale paths).
- Personal list customization (default lists unchanged).
- `bliss init` converting personal todos into a context.
- `bliss list --all` writing a session file (positions are ambiguous across contexts).

---

## 1. File Format Extension

Add `~/.bliss2/todos/` at the store root for personal todos:

```
~/.bliss2/
  todos/                     ← NEW: personal todo files
    <todo-uuid>.md
  lists/                     ← existing: personal list files
    inbox.txt
    today.txt
    this-week.txt
    next-week.txt
    later.txt
  contexts/
    <context-uuid>/
      meta.md                ← EXTENDED: second line is the init path
      todos/
        <todo-uuid>.md
      lists/
        <list-name>.txt
```

### meta.md format (extended)

```
My Project
/Users/cs/git/my-project
```

Line 1: context name (unchanged). Line 2: absolute filesystem path recorded at `bliss init` time. `bliss status` checks whether that path still contains a `.bliss-context` file pointing to the correct UUID. If not, the path is shown as `(stale path)`.

---

## 2. Personal Mode

**Personal mode** is the state where no `.bliss-context` file is found by walking up from the current directory. In personal mode, `contextUUID` is the empty string `""` throughout the store layer.

The store already uses `""` for personal lists. This design extends the same convention to todos.

Previously, `FindContext` returning no result caused every command to error. After this change, it is not an error — it is personal mode.

### Command behavior in personal mode

| Command | Personal mode behavior |
|---|---|
| `bliss add` | Writes to `~/.bliss2/todos/`; `--list` targets personal lists |
| `bliss list` | Shows personal lists + personal inbox |
| `bliss list --all` | Shows all contexts + personal lists (no session written) |
| `bliss done` | Resolves position from session, operates on personal todos |
| `bliss move` | Resolves position from session, moves within personal lists |
| `bliss check` | Interactive view of personal todos/lists |
| `bliss groom` | Grooming of personal todos/lists |
| `bliss history` | Shows store-wide history (no context filter) |
| `bliss status` | Shows personal mode view (see Section 3) |
| `bliss init` | Creates a new context (unchanged) |

### bliss init guard

`bliss init` errors if run from the home directory (`~`):

```
cannot init a context in the home directory — use personal mode instead
```

This prevents accidentally shadowing personal mode for all subdirectories.

---

## 3. bliss status

Replaces `bliss contexts`. Non-interactive, plain text output.

### When inside a context

```
* my-project  /Users/cs/git/my-project
  inbox    3
  today    2
  later    5

  other-project   /Users/cs/git/other     inbox 1  today 0
  archived-thing  (stale path)            inbox 0

  personal  inbox 2  today 1

  store  ↑2 ahead  ↓1 behind  origin/main
```

- Current context shown prominently at top with per-list breakdown.
- Other contexts shown as compact one-line summaries with path.
- Personal todos shown as their own section.
- Git sync shown at bottom.

### When in personal mode

```
  personal mode
  inbox    5
  today    2
  this-week  1

  my-project     /Users/cs/git/my-project   inbox 3  today 2
  other-project  /Users/cs/git/other        inbox 1

  store  no remote
```

- "personal mode" header instead of a context name.
- Other contexts shown as compact summaries.
- Git sync line always shown: remote status, or `no remote` if none configured.
- Lists with zero todos are omitted from all output in `bliss status`.

### bliss contexts

Removed. `bliss status` covers all its use cases.

---

## 4. Store Layer Changes

The `store` package is extended to handle personal todos and the new metadata:

### Store auto-initialization

`store.Open()` currently errors if `~/.bliss2` does not exist. In personal mode, a first-time user running any command must not hit this error. `store.Open()` is changed to auto-create the store directory (and git init it) if it does not exist — equivalent to calling `store.Init()` transparently. This ensures personal mode works with zero setup.

### TodosDir for personal mode

The internal `TodosDir(contextUUID)` helper currently always constructs `~/.bliss2/contexts/<uuid>/todos/`. When `contextUUID` is `""` this produces a broken path. `TodosDir` must branch on empty string and return `~/.bliss2/todos/` when `contextUUID` is `""`. This mirrors the pattern already used by `WriteList`/`ReadList` for personal lists.

All higher-level todo methods follow from this fix:

- `WriteTodo("", t)` — writes to `~/.bliss2/todos/<uuid>.md`
- `ReadTodo("", uuid)` — reads from `~/.bliss2/todos/<uuid>.md`
- `DeleteTodo("", uuid)` — deletes from `~/.bliss2/todos/`
- `ListTodos("")` — lists `~/.bliss2/todos/`
- `FindTodo(uuid)` — extended to search personal todos (`~/.bliss2/todos/`) first, then all context todos
- `RemoveFromAllLists("", uuid)` — removes from personal lists

### Context metadata extended

- `WriteContextMeta(uuid, name, path string)` — stores name and init path
- `ReadContextMeta(uuid)` returns `(name, path string, err error)` — note this is a breaking signature change from the current `(string, error)`; all call sites must be updated

### New git sync method

- `GitSyncStatus()` returns ahead/behind counts and remote name, or `("", 0, 0, nil)` if no remote

All existing personal list methods (`ReadList("", name)`, `WriteList("", name, l)`, `PersonalListNames()`) are unchanged.

---

## Rationale

### Personal mode as the default

The original design required `bliss init` before any command would work. This creates unnecessary friction for a user who just wants a todo list. By making personal mode the default, bliss is immediately useful. Contexts become an opt-in "level up" rather than a prerequisite.

### Empty contextUUID convention

Using `""` for personal mode is consistent with how personal lists already work in the store. It keeps the store interface uniform — every method that takes a `contextUUID` works with `""` — and avoids a parallel set of methods for personal vs. context todos.

### Path tracking in meta.md

Recording the init path enables `bliss status` to surface stale contexts — contexts whose directory has been moved or deleted. This is useful diagnostic information. The path is advisory (bliss works fine without it), so staleness is shown as a warning rather than an error. A future `bliss doctor` command can use this to scan the filesystem and offer fixes.

### bliss status replaces bliss contexts

`bliss contexts` showed names and todo counts. `bliss status` is a superset: it adds per-list breakdown, paths, and git sync. There is no reason to keep both. Removing `bliss contexts` keeps the command surface clean.

### bliss list --all without session

Position numbers from `bliss list --all` would span multiple contexts, creating ambiguity for `bliss done` and `bliss move` (which context does position 7 belong to?). Omitting the session from `--all` keeps the session model simple and unambiguous.

### Home directory guard in bliss init

A `.bliss-context` in `~` would shadow personal mode for every subdirectory on the machine — an almost certainly unintended consequence. The guard prevents this with a clear error message pointing back to personal mode.
