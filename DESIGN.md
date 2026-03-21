# Design

## Format for storing todos

Each todo is stored as a plain text file. The first line is the title, followed by an optional body separated by a blank line. Todo data and ordering are stored separately — reprioritizing a todo does not change the todo file itself.

Todos are organized in lists. Lists are plain text files with one todo identifier per line. Lists can have optional sections separated by `---` for micro-grouping.

See SPEC.md for the full format specification.

## Storage backend

Git is the storage backend. The store is a git repository at `~/.bliss2/`. Every change results in an immediate commit. Sync between computers is handled by git remotes.

It should be possible to aggregate todos across multiple contexts.

## Capturing todos

`bliss add <title>` captures a todo in the current context. Context is determined by a `.bliss-context` marker file in the current or any parent directory, initialized with `bliss init`.

It should also be possible to capture todos from other input channels, e.g. a mobile app, so todos can be captured on the go.

## Retrieving todos

Todos should always be retrievable on request. By default in the context where it matters. Getting a list.

There should also be a global overview. A list of lists, maybe a board.

## Prioritizing todos

It needs to be possible to easily order todos to reflect priorities.

It needs to be possible to groom todo lists, i.e. go through a list of todos and sort them in different other lists, e.g. to plan when to do them (example categories could be today, this week, next week, later).

Lists should have optional separators for micro-grouping todos.

## Completing todos

There needs to be a way to check off todos which have been done.

Done todos should vanish from the view to remove the mental load associated with them. They should still be available in history on request.
