# Bliss 2.0 CLI Specification

## General Principles

- UUIDs are never shown to the user. Todos are always identified by title and position.
- Every change to the store results in an immediate git commit.
- The current context is determined by walking up the directory tree from the working directory, using the first `.bliss-context` file found.
- Interactive mode is the primary interface for acting on todos. Non-interactive commands exist for quick use and scripting.

---

## Commands

### `bliss init`

Initializes a context in the current directory.

- Creates a `.bliss-context` marker file containing a new UUID.
- Creates the corresponding context directory in `~/.bliss/contexts/<uuid>/`.
- Derives the context name from the current directory name. Can be overridden with `--name <name>`.
- If a `.bliss-context` file is found by walking up the directory tree, the user is informed that a parent context exists. A nested context is created regardless.

```
bliss init
bliss init --name "My Project"
```

---

### `bliss add <title>`

Captures a new todo in the current context.

- The title is taken from command line arguments.
- By default the todo lands in the inbox (not added to any list).
- `--list <name>` adds the todo directly to a named list, appended to the end.
- `--urgent` places the todo at the top of the target list. Only valid in combination with `--list`.

```
bliss add Feed the penguins
bliss add Feed the penguins --list today
bliss add Feed the penguins --list today --urgent
```

---

### `bliss list [list-name]`

Displays todos in the current context.

- Without arguments: shows all todos, named lists first (in default order), inbox last.
- With a list name: shows only that list.
- `inbox` is a valid list name and shows floating todos not in any named list.
- By default the output is interactive (see Interactive Mode below).
- `--no-interactive` prints a plain text list, suitable for piping or scripting.

```
bliss list
bliss list today
bliss list inbox
bliss list --no-interactive
```

#### Interactive Mode

Navigation:
- Arrow keys to move between todos.
- `enter` to edit the title of the selected todo inline.
- `space` or `d` to complete (delete) the selected todo.
- `q` to quit.

---

### `bliss done [number]`

Completes (deletes) a todo.

- With a number: completes the todo at that position in the current list output. Position numbers are ephemeral — they reflect the current output of `bliss list`.
- Without arguments: opens interactive mode, equivalent to `bliss list`.

```
bliss done
bliss done 3
```

---

### `bliss groom [list-name]`

Interactive grooming mode for organizing todos across lists.

- Without arguments: starts with the inbox (incoming).
- With a list name: starts with that list.
- Shows one list at a time. The current list fills the view, with todos in order including section separators.
- Todos already acted on in the current session are marked as touched and not shown again, even if they appear in another list during the session.

#### Default Lists

The default personal Kanban lists are shown in this order:

1. Incoming (inbox)
2. Today
3. This Week
4. Next Week
5. Later

#### Navigation

- Arrow keys to move between todos.
- Tab / Shift-Tab to switch to the next / previous list.
- Number keys to jump directly to a list (1–5 for the default lists).

#### Acting on a Todo

- A number key moves the selected todo to the corresponding **section** of the current list.
- A list key (e.g. `t` for today, `w` for this week) moves the todo to the end of that list.
- A list key followed by a number moves the todo to the specified section of that list.
- `d` or `space` completes (deletes) the todo.
- `q` quits grooming.

When a todo is moved or completed it disappears from the current view immediately.

#### List Sections

Sections within a list are separated by `---` lines in the list file, optionally named (`--- urgent`). Sections are numbered 1–9 within each list and can be targeted when moving todos.

---

## Rationale

### Interactive mode as default for `bliss list`

The primary way to act on todos is by navigating a list. Making interactive mode the default removes the need for a separate command to enter it, while `--no-interactive` preserves scripting use cases.

### `bliss done` as a shortcut

`bliss done <number>` allows completing a todo without entering interactive mode — useful when the list is fresh in mind and the position is known. Without arguments it falls back to interactive mode, making it a consistent entry point.

### Grooming starts with inbox

The natural grooming flow is processing new, unorganized todos first, then reviewing existing lists. Starting with the inbox reflects this and encourages a GTD-style "capture first, organize later" workflow.

### Touched state is in-memory only

Grooming session state (which todos have been acted on) is not persisted to disk. Persisting it would complicate the data model and create stale state across sessions. The in-memory touched set is sufficient to prevent processing the same todo twice within a single grooming session.

### Title editing inline

Opening an external editor ($EDITOR) for title editing would be a context switch. Inline editing keeps the user in the terminal UI and the flow unbroken. Body editing is out of scope for the initial version.

### `--urgent` only with `--list`

The inbox has no sequence file — its order is determined by git commit time. Placing a todo at the top of the inbox is therefore not possible without creating an explicit inbox sequence file, which would add complexity. `--urgent` is only meaningful for named lists.
