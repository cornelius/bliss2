# Bliss 2.0

Peace of mind with todos. Note them down when something occurs to you. Get them back to the top of your head when needed. With context. With the lowest effort possible.

## Concept

Maintain a database of todos as plain text files which can be version controlled and shared easily. Readable by humans and machines in the same way. Provide clients to work with these lists. Provide a protocol to store and sync it on a server and a distributed way. Store them with context and enough meta information to make sense of them forever. Have an intelligent way to resurface them when you are ready to work on them. Have a convenient way to manage them, prioritize.

Should serve as personal Kanban. As todo list for a project. As a reminder for chores. As brain dump for ideas.

Underlying principle: Minimize brain cycles spent on todos. Free your mind to think big. Have all the details ready when needed, only when needed.

## Prior art

* https://github.com/cornelius/bliss (focused on the UI side, but built with the same model in mind, more info at https://blog.cornelius-schumacher.de/2013/06/experimenting-with-user-interfaces-for.html)
* https://apps.kde.org/de/korganizer/ (traditional todo manager, tailored to my needs and my taste)
* https://trello.com (lists of list, excellent tool for personal Kanban)
* http://todotxt.org/ (similar philosophy)
* https://en.wikipedia.org/wiki/Getting_Things_Done (influential background)

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

There also should be a global overview. A list of lists, maybe a board. A board could also be handy in a project context.

There also should be a way to let todos popup in the right context and right moment. Maybe as some kind of notification, but I don't want to see that as annoying thing that you have to silence, but as the magic help which gives you the support you just need. This would require some (artificial) intelligence. It also would need to keep environmental factors in mind, such as time or physical location, e.g. bubbling up the shopping list, when you are in the shop. This could be modeled as some sort of triggers. It could also be self-learning, so that you could give feedback, if a todo was surfaced at the right time and then the system would take this to improve for the next time. This might be much more than retrieving todos and warrant an own section.

Basic principle: Clients, format, and storage should be flexible and workable. Robust for manual and automatic handling. They should serve as building blocks and basic elements which can be orchestrated into bigger systems and extended by features in a very modular way.

### Prioritizing todos

It needs to be possible to easily order todos to reflect priorities.

It needs to be possible to groom todo lists, i.e. go through a list of todos and sort them in different other lists, e.g. to plan when to do them (example categories could be today, this week, next week, later).

List should have optional separators for micro grouping todos.

### Completing todos

There needs to be a way to check off todos which have been done.

Done todos should vanish from the view visible all the time to remove mental load associated with them. They should be still available in the history on request, so they can be tracked, processed, and checked, when necessary.

Completing a todo should be fund and satisfying experience. Like taking a piece of paper, crumbling it to a ball and throwing it into the bin. Or like popping a balloon. Or like exploding something. It should feel great to complete a todo.

It also should be possible to remove todos. Not sure if it matters to make a difference between completing and removing a todo. Maybe not. And removing a todo can also be satisfying.

## Development

**Prerequisites:** Go 1.21+

```sh
# Run tests
go test ./...

# Build
go build -o bliss ./cmd/bliss

# Install to $GOPATH/bin
go install ./cmd/bliss
```

**Quick start:**

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

### Extensions

Let's call them blisslets.
