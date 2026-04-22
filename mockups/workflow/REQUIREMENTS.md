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
- A muted action phrase that names what happened ("Added to", "Done", "Moved to", "Initialized")
- The todo title (done, move) or context name + path (init)
- The target `context/list` path (add) or list name (move)

## What must never appear
- A `bliss <command>` header banner — workflow commands do not have overview headers
- Internal-only identifiers — the context name (slug) shown in init output is the same one stored on disk; nothing else from the store should leak into command output
- Brackets around list names — use plain bold list name instead of `[today]`

## Variants

| Command | Variant | Output |
|---|---|---|
| add | incoming (no list) | `Added to context/incoming` (or `Added to incoming` in personal mode) |
| add | to named list | `Added to context/listname` (or `Added to listname` in personal mode) |
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
- **No store-internal identifiers in init output.** The user-visible context
  name (slug) is what they need to confirm, alongside the path.
- **Single line only.** These are transactional confirmations. No multi-line
  layout, no section structure.
- **`bliss add` does not echo the title.** The user just typed it (or piped it
  in). Echoing is redundant. The confirmation communicates *where* the todo
  landed, not *what* it says. `bliss list` is the place to see the title.
- **`bliss add` always names the destination as `context/list`.** Even when no
  `-l` flag is given, the output says `Added to <context>/incoming`. Incoming
  is a virtual view in the data model (no stored list file), but for output
  purposes it is treated as a pseudo-list name. This gives one uniform
  phrasing across all four cases (context × list, context × incoming,
  personal × list, personal × incoming) instead of two phrases with a
  special case.
