# bliss show — Requirements

## What question does it answer?

"What do I work on right now?" — focus mode. Shows only what is relevant to
the current context. No cross-domain mixing.

## When and why someone runs it

After opening a terminal. First thing in the morning. Between tasks. When you
want to know what's next without distraction.

## Relationship to bliss list

`bliss list` is the complete, unfiltered reference view. `bliss show` is the
focused, filtered view for action.

Both follow the same scoping rule: inside a context → context todos; outside →
personal todos. The difference is in filtering and default visibility:

| Behavior                        | bliss list | bliss show |
|---------------------------------|------------|------------|
| Inbox shown by default          | yes        | no (omitted unless non-empty) |
| Deferred todos shown            | yes        | no (future) |
| Scene filtering (personal)      | no         | yes (future) |

## Must always appear

- Named lists in semantic order, labeled with list names
- Position numbers on every todo (same session mapping as `bliss list`)
- "(no todos)" if there is truly nothing to show

## Must never appear

- UUIDs
- Empty named lists
- Inbox unless it contains items
- Brackets around list names
- Todos from a different domain (no personal in context mode, no context in personal mode)

## Variants

1. `bliss show` — inside a context: context lists only (inbox hidden unless non-empty)
2. `bliss show` — outside any context (personal mode): personal lists only (inbox hidden unless non-empty)
3. `bliss show <list-name>` — shows only the named list (context if in context, else personal)

## Decisions

- **No `--personal` flag.** `bliss show` is scoped to the current context — that
  is the point. Cross-domain access belongs to `bliss list --personal`. Adding
  `--personal` to `bliss show` would undermine the focus-mode intent.

## Future filtering (not implemented yet)

- Deferred todos (`defer` front matter field in future) are hidden from `bliss show`
  but visible in `bliss list`.
- When a scene is active, personal todos without a matching scene tag are hidden
  from `bliss show`.
