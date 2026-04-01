# bliss add / done / move / init — Requirements

## Category
Workflow commands. The user performs an action; the output confirms it happened.
See PRINCIPLES.md § "Command categories" for the governing philosophy.

## Purpose
Each command answers a different question:
- `bliss add` — "Did my todo get saved?"
- `bliss done` — "Did the right todo get marked complete?"
- `bliss move` — "Was it moved to the right list?"
- `bliss init` — "Is this directory now a bliss context?"

## What must always appear
- A muted action phrase that names what happened ("Added", "Done", "Moved to", "Initialized")
- The todo title (add, done, move) or context name + path (init)
- The target list name when relevant (add to list, move)

## What must never appear
- A `bliss <command>` header banner — workflow commands do not have overview headers
- The context UUID (internal detail, never user-facing)
- Brackets around list names — use plain bold list name instead of `[today]`

## Variants

| Command | Variant | Output |
|---|---|---|
| add | incoming (no list) | `Added: title` |
| add | to named list | `Added to listname: title` |
| done | — | `Done: title` |
| move | — | `Moved to listname: title` |
| init | — | `Initialized  Context: name  Path: ~/path` |

## Styling
- Action phrase → stMuted
- List name (add to, move) → stBold
- Title → plain (no color, no bold — it is content, not metadata)
- Path in init → stPath (same as all other path displays)
- `Context:` and `Path:` labels in init → stMuted (shared vocabulary with overview commands)

## Decisions made
- **No brackets on list names.** `[today]` → `today` (bold). Brackets were a
  placeholder convention; the bold weight is sufficient to distinguish the list
  name from the surrounding text.
- **UUID hidden from init output.** The UUID is an implementation detail.
  The name and path are what the user needs to confirm.
- **Single line only.** These are transactional confirmations. No multi-line
  layout, no section structure.
