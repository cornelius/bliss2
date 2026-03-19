# Agent Guide

## Documentation

| File | What it covers |
|------|----------------|
| SPEC.md | Store structure, file formats, design rationale |
| CLI.md | Every command, flags, interaction model, rationale |
| ARCHITECTURE.md | Package layout, dependencies, testing approach |
| README.md | Development, installation, usage quickstart |

## Build and test

Always run both before committing. No exceptions.

```sh
go test ./...
go build ./cmd/bliss
```

**Never test manually.** If something needs verifying, write a test for it. Manual runs are not repeatable and don't protect against regressions.

## Issue tracking

Todos for this project are tracked in bliss itself. The store is at `~/.bliss2/`. The context is the repo root (`.bliss-context` file).

Always run `./bliss list` first to get current position numbers before acting on any todo. Position numbers are stable within a session — running `bliss done 5` does not shift position 6 to 5. The session mapping is only replaced when `bliss list` is run again. To act on multiple todos without re-listing, use each todo's original position number from the last `bliss list` output.

UUIDs are stable across sessions and can also be used with `bliss done` and `bliss move`.

Useful commands:
```sh
./bliss list                      # see all todos with position numbers
./bliss move <n> -l <list>        # move a todo to a list
./bliss done <n>                  # complete a todo
./bliss history                   # see recent changes
```

Current lists: `next`, `groom`, inbox (virtual — todos not in any list).

## Code conventions

**Comments:** Only add them when the code isn't self-explanatory. Never restate the function name or include volatile details like paths. Focus on why, or on non-obvious behavior. See `getCreationTimes` in `internal/store/store.go` for a good example.

**Error handling:** Return clean user-facing messages. No stack traces, no raw library errors shown to users.

**Store access:** Nothing outside `internal/store` constructs paths into `~/.bliss2/` or touches store files directly. All I/O goes through `Store` methods.

**Interactive commands:** `bliss check` and `bliss groom` commit once on quit, not after each action. Non-interactive commands commit immediately after each change.

**Personal lists vs context lists:** Personal lists (`~/.bliss2/lists/`) may reference todos from any context. Use `store.FindTodo(uuid)` (not `ReadTodo`) when resolving UUIDs from personal lists.
