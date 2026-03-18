# Bliss 2.0 Architecture

## Project Structure

```
cmd/bliss/main.go       — entry point
internal/store/         — store access, git operations
internal/context/       — resolving .bliss-context markers
internal/todo/          — todo file read/write
internal/list/          — list file read/write, sections
internal/ui/            — interactive terminal UI (check, groom)
```

## Dependencies

- `cobra` — CLI command structure
- `bubbletea` — interactive terminal UI
- `go-git` — git operations (no external git binary required)

Dependencies are kept minimal. Any addition requires a clear reason.

## Git Integration

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

## Error Handling

- Explicit error returns throughout, no panics except for truly unrecoverable states.
- User-facing errors print a clean message and exit with a non-zero code. No stack traces shown to the user.

## Testing

- **Unit tests** for packages with well-defined logic (todo parsing, list parsing, context resolution). Minimal mocking — tests touch real files in temp directories where possible.
- **Integration tests** that invoke the full CLI, set up test data, run sequences of commands, and assert on output and file state.
- Coverage is focused on the parts most likely to break when things change, not on achieving a particular percentage.
