# bliss list — Output Requirements

## Purpose

Answers: "what is in my current context?"

This is the full, unfiltered view of a scope. It is insight-oriented, not
action-oriented. The user runs it to understand where things stand — to scan
across lists, notice what is stacking up, and decide what to act on next.
Position numbers are a side-effect that enables follow-up commands (`done`,
`move`), not the primary goal.

Contrast with `bliss show` (planned): that command answers "what do I work
on right now?" and applies filters (defer times, scenes). `bliss list` applies
no filters — if a todo exists, it appears.

## Variants

| Invocation | Scope |
|---|---|
| `bliss list` (inside context) | Context lists only, incoming included |
| `bliss list` (outside context) | Personal lists only |
| `bliss list <name>` | Single named list only |
| `bliss list incoming` | Floating todos not in any named list |
| `bliss list --personal` | Personal lists only, regardless of context |
| `bliss list --personal <name>` | Single personal list |
| `bliss list --all` | All contexts + personal, no position numbers |

## Always show

- Which context (or personal scope) the output is scoped to — the output must
  be self-contained and readable in scrollback without knowing the cwd.
- All lists in the current scope, in semantic order:
  today → this-week → next-week → later → custom lists → bugs → incoming.
- Items within each list in list-file order, preserving section structure.
- Position numbers on every todo item (except `--all` which spans contexts).

## Never show

- Todos from outside the current scope (personal todos do not appear in a
  context view unless `--personal` is passed).
- Empty lists.
- UUIDs.

## Resolved decisions

**Deferred todos:** Shown in `bliss list` (unfiltered), but marked with a
visual indicator (e.g. `⏸`) and their defer time. They are hidden in
`bliss show`.

**`--all` structure:** Same list grouping as the default view — todos grouped
by list within each context, same semantic order. Not a flat dump.

**Custom list order:** Deferred — requires broader design of default lists and
list collections. Currently falls back to alphabetical within custom lists.

## Open questions

**Header in filtered view (`bliss list today`):**
The list name is already in the command. Two options:
- Show nothing — zero distraction, trust the user knows what they ran.
- Show just the scope identifier (context name or "personal") without
  repeating the list name — self-contained in scrollback, consistent with
  `bliss status`.

**Header in `--personal` view (inside a context):**
Same tension: workflow focus says show nothing; consistency with `bliss status`
says label the scope. To be resolved alongside the filtered view question —
they should behave the same way.

## Constraints

- Position numbers must be stable within a session: running `done 3` after
  `list` must always refer to the same todo, regardless of earlier completions.
- Section dividers (`── name`) within a list must be preserved and visible.
- Output must work in plain text without color.
