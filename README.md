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

See [DESIGN.md](DESIGN.md) for the design.

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
