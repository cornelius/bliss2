# Bliss 2.0 File Format Specification

See ARCHITECTURE.md for implementation decisions (language, dependencies, project structure, testing approach).

## Storage Backend

The store is a git repository. Git is not one of several possible backends — it is a core part of the design. Timestamps, history, and the integrity of the audit trail all depend on git. Every change to the store results in an immediate commit.

Sync between machines is handled by git remotes in the usual way.

## Store Structure

All data lives in a single directory in the user's home:

```
~/.bliss/
  contexts/
    <context-uuid>/
      meta.md
      todos/
        <todo-uuid>.md
      lists/
        <list-name>.txt
  lists/
    <list-name>.txt
```

- `contexts/` contains one subdirectory per context, named by UUID.
- `meta.md` stores the human-readable name of the context and any other context-level metadata.
- `todos/` contains one file per todo, named by UUID.
- `lists/` inside a context contains shared, context-specific list files.
- `lists/` at the store root contains personal, cross-context list files.

## Context Markers

A directory is associated with a context by placing a `.bliss-context` file in it:

```
~/my-project/.bliss-context
```

The file contains a single UUID identifying the context in the store:

```
7f3a2b1c-4d5e-6f7a-8b9c-0d1e2f3a4b5c
```

The CLI finds the context by walking up the directory tree from the current working directory, using the first `.bliss-context` file found. This mirrors the behavior of git.

## Todo File Format

Each todo is stored as a plain text Markdown file named by its UUID:

```
<todo-uuid>.md
```

The first line is the title. An optional body follows after a blank line:

```
Feed the penguins

Make sure to bring the fish from the freezer first.
Check with the zookeeper about portion sizes.
```

No metadata is stored in the file. Creation and modification timestamps are derived from git history.

## List Files

A list file is a plain text file containing an ordered sequence of todo UUIDs, one per line:

```
a1b2c3d4-e5f6-7890-abcd-ef1234567890
7f3a2b1c-4d5e-6f7a-8b9c-0d1e2f3a4b5c
```

List files use the `.txt` extension and are named by the list name (e.g. `backlog.txt`, `today.txt`).

There are two kinds of lists:

- **Context lists** — live inside a context directory. Shared within a team when the store is version controlled collaboratively.
- **Personal lists** — live at the store root. Cross-context, private to the user.

## List Sections

A list file can be divided into sections using a separator line:

```
a1b2c3d4-e5f6-7890-abcd-ef1234567890
---
7f3a2b1c-4d5e-6f7a-8b9c-0d1e2f3a4b5c
```

A separator is a line containing only `---`, optionally followed by a section name:

```
--- urgent
```

Sections allow micro-grouping within a list without creating separate list files. A list can have 1 to 5 sections; up to 9 is the hard limit. Sections have no semantic meaning to the data model — they are a presentation and prioritization aid.

The default personal lists follow the personal Kanban framework: `incoming`, `today`, `this-week`, `next-week`, `later`. These can be customized.

## Inbox

Todos that are not referenced by any list file are considered to be in the inbox. There is no explicit inbox file. The inbox is a virtual view — all todos in a context not appearing in any list. Inbox todos are ordered by creation time, derived from git history.

## Ordering and Prioritization

The order of UUIDs in a list file defines the priority of todos. The first entry is the highest priority.

When a todo is added to a list it is appended to the end by default. A `--urgent` flag places it at the top instead.

Reprioritizing a todo only modifies the list file — the todo file itself is never touched. Reprioritizing one todo does not affect any other todo's data.

## Completing Todos

Completing a todo deletes its file from `todos/`. Any references to its UUID in list files are also removed. Git history preserves the full content and lifecycle of the todo.

## UUIDs

All identifiers — todos and contexts — are UUIDs, generated at creation time and globally unique. UUIDs are never exposed to the user. The CLI always presents human-readable names and titles.

## Timestamps

Creation and modification timestamps are not stored in files. They are derived from git history. Every change to the store results in an immediate commit, so git timestamps accurately reflect real creation and modification times.

---

## Rationale

### One file per todo

Storing each todo as a separate file means that any operation on one todo — editing, completing, reprioritizing — produces a minimal, focused diff. Unrelated todos are untouched. This makes git history clean and meaningful, merge conflicts rare and localized, and manual inspection straightforward.

The expected scale of hundreds to low thousands of files is well within the comfortable range for both filesystems and git.

### Separation of todo data and ordering

Ordering and grouping are stored in list files, not in todo files. This reflects the principle that reprioritizing a todo does not change the todo itself. When one todo moves to the top of a list, only the list file changes — no other todo is affected. This keeps diffs minimal and semantically correct.

The same principle applies to grooming: moving a todo between lists changes one or two list files and nothing else.

### Central store with context markers

An alternative design would place a `.bliss/` directory inside each project directory, co-located with the project. The central store approach was chosen instead because:

- All data lives in one place, making backup and version control simple.
- Project directories are not cluttered with todo data.
- Todos are completely decoupled from the project filesystem structure — renaming or moving a project directory does not affect the store.

The `.bliss-context` marker file is the only artifact in a project directory. It contains a UUID, so renaming or moving the directory never breaks the link to the store.

### Git as storage backend

Git was chosen as the storage backend because it provides version history, sync via remotes, and timestamps for free — without any custom infrastructure. The decision to make every change an immediate commit means git timestamps are accurate and authoritative, removing the need to store created_at and updated_at in files. This keeps todo files minimal and human-writable.

Alternative backends (WebDAV, custom server) were considered in the initial design but ruled out. The plain file format remains backend-agnostic in principle, but the CLI is built around git.

### Inbox as a virtual view

An explicit inbox list file would need to be updated on every todo creation, adding noise to diffs. Defining the inbox as all todos not referenced by any named list requires no bookkeeping and produces no extra diff noise when a todo is created. The ordering of inbox todos by git commit time is a natural and automatic consequence of the storage model.

### Completing as deletion

Marking a todo as done by setting a flag in the file would leave done todos in the active dataset, requiring all queries to filter them out. Deletion removes them from the active view immediately and cleanly. Git history preserves the full record, so nothing is lost. The distinction between completing and removing a todo is intentionally not made — both are deletions.

### UUIDs never shown to users

UUIDs are an implementation detail for stable cross-context references. Exposing them would add cognitive load with no benefit. The CLI always translates to human-readable names and titles. Interaction with specific todos happens through selection in interactive mode, not by typing identifiers.
