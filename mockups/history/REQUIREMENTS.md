# bliss history — Requirements

## Category
Overview command. The user pauses to read recent activity.
See PRINCIPLES.md § "Command categories" for the governing philosophy.

## Purpose
Answers: "what have I done recently in this context?"
Secondary: audit trail for moves, completions, additions.

## What must always appear
- A `bliss history` header line (overview command)
- Context name and path in the header (or "Personal", or "--all")
- Timestamp (date + time) for each entry, muted
- A clean, human-readable description of what happened

## What must never appear
- Raw git commit message strings ("add todo", "complete", "move ... to ...")
  — these are internal implementation language, not user-facing copy
- Internal entries the user did not explicitly trigger ("add list sections",
  the bliss2-was-here marker commit)
- An unbounded list — default to the 20 most recent entries

## Variants

| Invocation | Header |
|---|---|
| `bliss history` | `bliss history  Context: name  Path: ~/path` |
| `bliss history` (personal mode) | `bliss history  Personal` |
| `bliss history --all` | `bliss history --all` |

## Entry format
`  Mon 02  15:04  Human-readable action description`

Date and time are muted. Description is plain text, capitalized, uses
natural language:
- `add todo "X"` → `Added "X"`
- `add todo "X"` + list context → `Added "X"` (list not in commit message)
- `complete "X"` → `Completed "X"`
- `move "X" to listname` → `Moved "X" → listname`
- `init context name (uuid)` → `Initialized context name`
- `add list sections` → skip (internal, not user-triggered)
- bliss2-was-here, meta.yaml init → skip

## Limit
Default: 20 most recent entries. If more exist, show a muted note:
`  (showing 20 most recent)`

## Decisions made
- **Flat list for --all.** Per-context grouping in --all mode is complex
  (ReadHistory("") returns a single interleaved stream). Keep it flat.
  This may be revisited once history semantics mature.
- **No relative dates ("today", "yesterday").** Adds implementation
  complexity for marginal UX gain. Use absolute date consistently.
- **→ arrow for moves.** More compact and visual than "to". Consistent with
  the symbols-where-they-help principle.
- **Quotes around titles.** Makes titles visually distinct from action words
  and list names in the same line.
