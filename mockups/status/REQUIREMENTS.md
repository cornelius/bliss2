# bliss status — Requirements

## Purpose

System overview. Answers the question: **where does everything stand?**
Not a workflow tool — you run this to orient yourself, not to act on todos.

## What it shows

**Contexts.**
All known contexts are listed. Each shows a count per named list.
The active context (cwd matches a .bliss-context) is marked.
Personal mode (no context in cwd) is shown like any other context, just
labelled "personal".

All contexts use the exact same format. The active context gets a marker,
not extra detail.

**Store.**
Always shown, even when there is nothing to report:
- Local store path
- Remote URL (or indication that none is configured)
- Sync state: ahead, behind, or synced

## What it does not show

- Todo titles or content
- Individual todo details
- A prompt to act on anything

## Layout constraints

- One context per line (or one compact block — no multi-line per context)
- Lists appear in a consistent, semantic order:
  today → this-week → next-week → later → [custom] → bugs → incoming
- Lists with 0 todos: omit the list, or show a dash — do not clutter
- Must read clearly without color
