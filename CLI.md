# Bliss 2.0 CLI Specification

## General Principles

- UUIDs are never shown in output. Todos are always identified by title and position number.
- Non-interactive commands commit immediately after each change. Interactive commands (`check`, `groom`) commit once when the session ends.
- The current context is determined by walking up the directory tree from the working directory, using the first `.bliss-context` file found.
- Commands are either strictly non-interactive or strictly interactive. No command switches between modes based on arguments or flags.

---

## Commands

### Non-Interactive Commands

Non-interactive commands produce plain text output or perform a single action. They are suitable for scripting and quick use from the command line.

#### `bliss init`

Initializes a context in the current directory.

- Creates a `.bliss-context` marker file containing a new UUID.
- Creates the corresponding context directory in `~/.bliss2/contexts/<uuid>/`.
- Derives the context name from the current directory name. Can be overridden with `--name <name>`.
- If a `.bliss-context` file is found by walking up the directory tree, the user is informed that a parent context exists. A nested context is created regardless.

```
bliss init
bliss init --name "My Project"
```

#### `bliss add <title>`

Captures a new todo in the current context.

- If a title is given as arguments, they are joined with spaces. Because the title is passed through the shell, special characters (apostrophes, quotes) must be quoted: `bliss add "Fix John's bug"`.
- If no arguments are given, bliss reads the title from stdin. In an interactive terminal it prints a `Title:` prompt; when stdin is a pipe it reads one line silently. This avoids shell quoting entirely for titles with special characters.
- By default the todo lands in the inbox (not added to any list).
- `--list/-l <name>` adds the todo directly to a named list, appended to the end.
- `--urgent` places the todo at the top of the target list. Only valid in combination with `--list`.

```
bliss add "Feed the penguins"
bliss add "Fix John's bug" -l today
bliss add                              # prompts for title interactively
echo "Fix John's bug" | bliss add     # reads title from pipe
bliss add "Feed the penguins" -l today --urgent
```

#### `bliss list [list-name]`

Displays todos in the current context with position numbers.

- Without arguments: shows context lists first, then personal lists, then inbox.
- With a list name: shows only that list.
- `inbox` is a valid list name and shows floating todos not in any named list.
- Writes a session mapping (`~/.bliss2/session.txt`) of position numbers to UUIDs. This mapping is the basis for `bliss done` and `bliss move`.
- Position numbers are **stable within a session**: completing todo 5 does not shift todo 6 to position 5. The session is replaced only when `bliss list` is run again. To complete multiple todos, use each one's original position number from the last `bliss list` output.

```
bliss list
bliss list today
bliss list inbox
```

#### `bliss done <number|uuid>`

Completes (deletes) the todo at the given position number, or by UUID.

- Position numbers come from the last `bliss list` output.
- A UUID can be given directly instead of a position number, bypassing the session.
- Errors if a position number is given but no session mapping exists (i.e. `bliss list` has not been run yet).
- Removes the todo file and any references to it in list files.

```
bliss done 3
bliss done 7f3a2b1c-4d5e-6f7a-8b9c-0d1e2f3a4b5c
```

#### `bliss move <number|uuid> --list <name>`

Moves a todo to a named list.

- Accepts a position number from the last `bliss list` output, or a UUID directly.
- `--list/-l <name>` is required.
- `--urgent` places the todo at the top of the target list.
- Removes the todo from any list it currently appears in.

```
bliss move 3 -l today
bliss move 3 -l today --urgent
```

#### `bliss contexts`

Lists all contexts in the store.

- Marks the active context (resolved from the current directory).
- Shows the todo count for each context.

```
bliss contexts
```

#### `bliss history [--all]`

Shows the change history for the current context.

- Without flags: shows git log entries that touch the current context.
- `--all` shows the full store history across all contexts.

```
bliss history
bliss history --all
```

---

### Interactive Commands

Interactive commands take over the terminal and allow navigation and action via keyboard. They never print plain text output. Changes are committed as a single git commit when the session ends.

#### `bliss check [list-name]`

Interactive view of a single list for navigating and completing todos.

- Without arguments: shows all todos in the current context (context lists first, personal lists, inbox last).
- With a list name: shows only that list.
- Does not support multi-list switching or touched state — use `bliss groom` for that.

Navigation:
- Arrow keys to move between todos.
- `enter` to edit the title of the selected todo inline.
- `space` or `d` to complete (delete) the selected todo.
- `q` to quit.

```
bliss check
bliss check today
bliss check inbox
```

#### `bliss groom [list-name]`

Interactive grooming mode for organizing todos across lists.

- Without arguments: starts with the inbox.
- With a list name: starts with that list.
- Shows one list at a time. The current list fills the view with todos in order including section separators.
- Todos already acted on in the current session are marked as touched and not shown again, even if they appear in another list during the session. Touched state is in-memory only and is lost when grooming ends.

##### Default Lists

The default personal Kanban lists are shown in this order:

1. Inbox
2. Today
3. This Week
4. Next Week
5. Later

##### Navigation

- Arrow keys to move between todos.
- Tab / Shift-Tab to switch to the next / previous list.
- Number keys (1–5) to jump directly to a list.

##### Acting on a Todo

- A list key (e.g. `t` for today, `w` for this week) moves the todo to the end of that list.
- A list key followed by a number moves the todo to the specified section of that list.
- `d` or `space` completes (deletes) the todo.
- `q` quits grooming.

When a todo is moved or completed it disappears from the current view immediately.

##### List Sections

Sections within a list are separated by `---` lines in the list file, optionally named (`--- urgent`). Sections are numbered 1–9 within each list and can be targeted when moving todos.

---

## Rationale

### Strict separation of interactive and non-interactive commands

Each command is either fully interactive or fully non-interactive — there are no flags to switch modes. This makes the behavior of each command predictable and keeps the implementation of each command focused. It also makes scripting reliable: non-interactive commands always produce consistent plain text output.

An earlier design had `bliss list` default to interactive mode with a `--no-interactive` flag. This was abandoned because mixing modes in one command blurs its purpose and makes it harder to compose with other tools.

### `bliss list` and session mapping

`bliss done <number>` and `bliss move <number>` need stable position numbers between the `bliss list` call and the action. The session mapping file (`~/.bliss2/session.txt`) records the UUID at each position as shown by the last `bliss list`. This mapping is replaced only when `bliss list` is called again — not after each `done` or `move`. This means you can complete todos 3, 5, and 7 in sequence using their original position numbers without re-listing. Both commands also accept a UUID directly, which bypasses the session entirely and is more robust for scripting.

### Interactive commands commit on quit

Each action in `bliss check` or `bliss groom` writes to disk immediately so that a crash does not lose work. The git commit is deferred to when the session ends with `q`, producing a single commit per session rather than one per action. This keeps the git history meaningful.

### `bliss check` vs `bliss groom`

Both are interactive but serve different purposes. `bliss check` is for quickly navigating and completing todos in a single list — a lightweight view. `bliss groom` is for a full grooming session across multiple lists with touched state tracking. Keeping them separate keeps each command focused and avoids a complex mode-switching interface in a single command.

### Grooming starts with inbox

The natural grooming flow is processing new, unorganized todos first, then reviewing existing lists. Starting with the inbox reflects this and encourages a GTD-style "capture first, organize later" workflow.

### Touched state is in-memory only

Grooming session state (which todos have been acted on) is not persisted to disk. Persisting it would complicate the data model and create stale state across sessions. The in-memory touched set is sufficient to prevent processing the same todo twice within a single grooming session.

### Title editing inline

Opening an external editor (`$EDITOR`) for title editing would be a context switch. Inline editing keeps the user in the terminal UI and the flow unbroken. Body editing is out of scope for the initial version.

### `--urgent` only with `--list`

The inbox has no sequence file — its order is determined by git commit time. Placing a todo at the top of the inbox is therefore not possible without creating an explicit inbox sequence file, which would add complexity. `--urgent` is only meaningful for named lists.
