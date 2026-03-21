# bliss history — Requirements

## Category
Overview command. The user pauses to read recent activity.
See PRINCIPLES.md § "Command categories" for the governing philosophy.

## Purpose
Answers: "what have I done recently in this context?"
Secondary: audit trail for moves, completions, additions.

## What must always appear
- A `bliss history` header line with scope info and current date
- Timestamp (ISO date + time, muted) for each entry
- Commit message with "bliss: " prefix stripped if present

## What must never appear
- UUIDs

## Variants

| Invocation | Header |
|---|---|
| `bliss history` | `bliss history  Context: name  Path: ~/path  Date` |
| `bliss history` (personal mode) | `bliss history  Personal  Date` |
| `bliss history --personal` | `bliss history  Personal  Date` |
| `bliss history --all` | `bliss history --all  Date` |

## Entry format
`2006-01-02 15:04  raw commit message`

ISO date and time are muted. Messages are shown with the "bliss: " prefix
stripped if present. Newest first. No limit on number of entries.

In `--all` mode a context label column is added between the timestamp and
the message; width equals the longest context name (or "personal"), padded.

## Decisions made
- **Flat list for --all.** Per-context grouping in --all mode is complex
  (ReadHistory("") returns a single interleaved stream). Keep it flat.
  This may be revisited once history semantics mature.
- **No relative dates ("today", "yesterday").** Adds implementation
  complexity for marginal UX gain. Use absolute date consistently.
- **Quotes around titles.** Makes titles visually distinct from action words
  and list names in the same line.
