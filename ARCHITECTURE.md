# Bliss 2 Architecture

## Project Structure

```
cmd/bliss/main.go       — entry point
e2e/                    — e2e tests (invoke the real binary, verify user contract)
internal/store/         — store access, git operations
internal/context/       — resolving .bliss-context markers
internal/todo/          — todo file read/write
internal/list/          — list file read/write, sections
internal/ui/            — interactive terminal UI (check, groom)
```

## Client implementation

- CLI implemented in Go (module `github.com/cornelius/bliss2`, binary `bliss`)

- `cobra` — CLI command structure
- `bubbletea` — interactive terminal UI
- `go-git` — git operations (no external git binary required)

Dependencies are kept minimal. Any addition requires a clear reason.

## Git Integration

- Git as storage backend (`~/.bliss2/`)

`go-git` is used for all git operations. It is encapsulated entirely within the `store` package behind an interface:

```go
// internal/store/store.go
type Store interface {
    Commit(message string) error
}
```

Nothing outside the `store` package interacts with git directly. If the git backend is replaced in the future, only the `store` package changes.

## Store Encapsulation

The `store` package is the single owner of:

- All path construction into `~/.bliss2/`
- All file I/O on store data
- All git operations

No other package constructs paths into the store or reads/writes store files directly.

## Context Identity: Slugs

Contexts are identified by a **slug** — a lowercase, hyphen-separated string derived from the context name at `bliss init` time (e.g. `"My Project"` → `my-project`). The slug is:

- Stored in the `.bliss-context` marker file in the project directory.
- Used as the context's directory name in the store: `~/.bliss2/contexts/<slug>/`.
- The stable, human-readable identity for the context — no UUID is used.

`meta.yaml` inside each context directory holds context-wide metadata:

```yaml
created_at: 2026-01-15T10:30:00Z
```

Per-machine filesystem paths live in a sibling `paths/` directory, one file per host:

```
contexts/my-project/
  paths/
    thinkpad.yaml   # path: /home/cs/git/my-project
    macbook.yaml    # path: /Users/cs/projects/my-project
```

This is the **cross-machine context linking** mechanism. Each host writes only its own `paths/<hostname>.yaml`, so independent edits never collide under git merge.

### Cross-machine linking

The same context can exist on multiple machines. To link a second machine:

1. Run `bliss sync` to pull the store (which includes the context directory).
2. Run `bliss init --name "My Project"` (or just `bliss init` if the directory name matches) in the project directory on the new machine.
3. Because `~/.bliss2/contexts/my-project/` already exists (pulled via sync), `bliss init` detects the collision and offers to **link** rather than create. Confirming writes `paths/<hostname>.yaml` for the current machine and writes `.bliss-context`.

If two machines run `bliss init` for the same slug before syncing, no merge conflict is produced: each machine writes a distinct `paths/<hostname>.yaml`, and the only shared file (`meta.yaml`) is written with the same content on both sides. The earlier-created copy of `meta.yaml` survives the merge by virtue of being identical or near-identical; if `created_at` differs, prefer the older value.

This design means:
- Each machine knows its own local path to the project.
- Context data (todos, lists) is shared via git sync.
- Renaming or moving the project directory on one machine does not affect other machines — each manages its own `paths/<hostname>.yaml`.

## Error Handling

- Explicit error returns throughout, no panics except for truly unrecoverable states.
- User-facing errors print a clean message and exit with a non-zero code. No stack traces shown to the user.

## Testing

Two distinct layers, with different purposes. Some overlap between them is
expected and fine — they are not solving the same problem.

### E2e tests (`cmd/bliss/e2e/`)

E2e tests verify the **user contract**. They invoke the real `bliss` binary,
set up state through the CLI, and assert on observable output and exit codes.
They are implementation-agnostic: a rewrite in another language should pass
them unchanged.

**CLI.md is the specification.** Every documented command, flag, and behavior
should have a corresponding e2e test. Together the e2e suite is a machine-
readable version of the spec.

Guidelines:
- One binary build shared across all tests (TestMain pattern).
- Each test gets an isolated HOME via a temp directory — no shared state.
- Test names read as specifications: `TestSync_pushesCommitsToRemote`.
- Assertion messages state the contract, not the code path.
- Local bare git repos stand in as remotes — no network, no SSH keys.

### Unit tests (`*_test.go` alongside the package)

Unit tests are a **development tool**. Their primary purpose is fast feedback
during development: they catch regressions immediately, are cheap to run, and
are cheap to write. They test internal contracts — the behavior of individual
functions and packages in isolation.

Unit tests should exist even when they duplicate checks already covered by e2e
tests, because their purpose is different: not proving end-to-end correctness
but making it fast and easy to identify exactly what broke and where.

Prefer unit tests for:
- Corner cases and error paths that are expensive to drive from the CLI.
- Formatting, sorting, and layout logic (output alignment, semantic order).
- Internal contracts between packages.
- Anything where a unit test covers in 5 lines what an e2e test needs 30 for.

Guidelines:
- Touch real files in temp directories — minimal mocking.
- Keep them fast. If a test needs a git repo, that is acceptable; if it needs
  a remote, write an e2e test instead.

### General

- **Never test manually.** If something needs verifying, write a test.
- Coverage is focused on the parts most likely to break, not a percentage goal.

## Session File

`bliss list` writes a session file (`~/.bliss2/session.txt`) mapping position numbers to UUIDs. Position numbers shown in output are stable for the lifetime of that session — they do not change when todos are completed or moved. The session is only replaced when `bliss list` runs again.

`bliss done N` and `bliss move N` resolve position numbers through the session file. Always run `bliss list` before `bliss done` or `bliss move` so positions are up to date.

## Incoming

The incoming is a virtual view, not a stored list. `getIncomingTodos` returns all todos in the context that are not referenced by any named list. This means a todo is automatically "in the incoming" until it is added to a list.

## Sections

A list file contains one or more sections separated by `---` lines. Sections can have an optional name on the `---` line. The `list.List` type models this as `Sections []Section`.

In output:
- `bliss list` renders section separators as `      ──` (aligned with the todo title column).
- `bliss check` renders unnamed separators as `  ──` and named ones as `  ── name`.

## Interactive TUI

The TUI (bubbletea) has two models:
- `CheckModel` — used by `bliss check`. Supports navigation across todos and section headers, editing todo titles (enter), inserting sections (s), and renaming sections (enter on a section header).
- `GroomModel` — used by `bliss groom`. Focuses on reordering.

Both follow the bubbletea value-receiver pattern: `Update` returns a new model rather than mutating in place. Writes to the store happen immediately on each action; a git commit is issued on quit only if any write occurred (`dirty` flag).

### CheckItem type system

`CheckItem` is the row type for `CheckModel`. Each row is one of:
- **List-name header** (`IsListHeader: true`) — rendered as `[list-name]`, used in the all-lists view.
- **Section separator** (`IsSectionHeader: true`) — rendered as `──`; carries `SectionIdx` for rename.
- **Todo** (`Todo != nil`) — carries `ListName` and `ListContextUUID` so section insertion works from the all-lists view without needing to know which list is active.

### Section insert / rename flow

Insert (`s` key): splits the current todo's section at the todo's position, writes the updated list, and inserts a new `CheckItem` separator into the in-memory item slice. In single-list view the items are rebuilt from the list file; in all-lists view the new separator is inserted in place.

Rename (enter on a separator): loads the section name into the text input. On confirm, writes the updated name to the list file and rebuilds items via `itemsFromList`.

## `bliss add` stdin support

When called with no arguments, `bliss add` reads a title from stdin. If stdin is a terminal (character device), it prints a `Title: ` prompt first. This lets piped input work silently: `echo "buy milk" | bliss add`.
