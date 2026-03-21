# Bliss 2

Peace of mind with todos. Note them down when something occurs to you. Get them back to the top of your head when needed. With context. With the lowest effort possible.

Minimize brain cycles spent on todos. Free your mind to think big.

See [VISION.md](VISION.md) for the full picture.

## Prior art

* https://github.com/cornelius/bliss (focused on the UI side, but built with the same model in mind, more info at https://blog.cornelius-schumacher.de/2013/06/experimenting-with-user-interfaces-for.html)
* https://apps.kde.org/de/korganizer/ (traditional todo manager, tailored to my needs and my taste)
* https://trello.com (lists of list, excellent tool for personal Kanban)
* http://todotxt.org/ (similar philosophy)
* https://en.wikipedia.org/wiki/Getting_Things_Done (influential background)
* https://cornelius.github.io/top/ (thoughts on productivity)

## Design

### Format for storing todos

Each todo is stored as a plain text file. The first line is the title, followed by an optional body separated by a blank line. Todo data and ordering are stored separately — reprioritizing a todo does not change the todo file itself.

Todos are organized in lists. Lists are plain text files with one todo identifier per line. Lists can have optional sections separated by `---` for micro-grouping.

See SPEC.md for the full format specification.

### Storage backend

Git is the storage backend. The store is a git repository at `~/.bliss2/`. Every change results in an immediate commit. Sync between computers is handled by git remotes.

It should be possible to aggregate todos across multiple contexts.

### Capturing todos

`bliss add <title>` captures a todo in the current context. Context is determined by a `.bliss-context` marker file in the current or any parent directory, initialized with `bliss init`.

It should also be possible to capture todos from other input channels, e.g. a mobile app, so todos can be captured on the go.

### Retrieving todos

Todos should always be retrievable on request. By default in the context where it matters. Getting a list.

There should also be a global overview. A list of lists, maybe a board.

### Prioritizing todos

It needs to be possible to easily order todos to reflect priorities.

It needs to be possible to groom todo lists, i.e. go through a list of todos and sort them in different other lists, e.g. to plan when to do them (example categories could be today, this week, next week, later).

List should have optional separators for micro grouping todos.

### Completing todos

There needs to be a way to check off todos which have been done.

Done todos should vanish from the view to remove the mental load associated with them. They should still be available in history on request.

## Development

**Prerequisites:** Go 1.21+

```sh
# Run tests
go test ./...

# Build and run locally without installing
go build ./cmd/bliss
./bliss
```

## Installation

Add `~/go/bin` to your PATH once (e.g. in `~/.zshrc`):

```sh
export PATH="$PATH:$HOME/go/bin"
```

Then install:

```sh
go install ./cmd/bliss
```

## Usage

```sh
cd my-project
bliss init
bliss add "My first todo"
bliss list
bliss done 1
```

---

### Command line client

- `bliss init` — initialize a directory as a todo context
- `bliss add <title>` — capture a todo in the current context
- `bliss list` — list todos with position numbers
- `bliss done <number>` — complete a todo by position number
- `bliss check` — interactive view to navigate and complete todos
- `bliss groom` — interactive grooming across lists

See CLI.md for the full command specification.

### Technical decisions

- CLI implemented in Go (module `github.com/cornelius/bliss2`, binary `bliss`)
- Git as storage backend (`~/.bliss2/`)
- An Android app for mobile capture is planned
