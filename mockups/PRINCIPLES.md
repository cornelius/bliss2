# Design Principles

## General

**Self-contained.**
Output should be understandable without knowing where it came from or what
command produced it. A reader seeing it in a scrollback buffer or a paste
should be able to make sense of it cold.

**Label everything: values and entity types.**
Every piece of data has a visible label. If a number appears, the reader
must know what it counts. But labeling values is not enough — the type of
every entity must also be labeled in the output itself. A context must be
labeled "context". A list must be identifiable as a list. The label is not
optional and cannot be implied by position or prior knowledge of bliss.

**Explicit hierarchy in the output.**
The data has a clear structure: a context is a project directory; a list is
a named collection of todos inside a context. These must appear as distinct
labeled entities in the output. A context name and a list name must never
occupy the same visual role or appear interchangeable.

**Lists are per-context and vary.**
There is no fixed set of list names. Each context can have completely
different lists, or none at all. The layout must work regardless of what
lists exist or how many.

**Symbols where they help.**
Unicode symbols (→ ● ✓ ↑ ↓ · ─ >) are welcome when they aid visual
arrangement or convey meaning at a glance. Decoration for its own sake is
not.

**Color is sparse and purposeful.**
Almost everything is uncolored or grey. Color appears only where it carries
meaning:

- One brand accent (cyan) on the word "bliss" — gives the tool a consistent
  identity across all output.
- Grey to stress or de-stress text: lighter grey for labels and secondary
  information, darker grey for paths and very low-priority elements.
- Status signals only in status output: green for good (synced, done), red
  for bad (behind, error), amber for warn (ahead, stale). Nowhere else.

Never use color to make output look decorated, colorful, or lively. If
removing color makes the output equally readable, the color was wrong.

**Text, not boxes.**
No multi-line box drawings that depend on vertical character alignment to
stay readable. Output must survive copy/paste, plain-text logging, and
narrow terminals. Single horizontal rules (────) are fine.

**Hierarchy through spacing and indentation.**
Visual weight comes from position and whitespace. Output must read well even
without color.

**No wasted lines.**
Two consecutive blank lines is a bug. Spacing is intentional: one blank line
separates logical sections; no blank line means items belong together.

**Prompt-friendly.**
Output flows naturally between a shell prompt above and a shell prompt below.

**Scannability.**
The most important information is findable in one glance. Structure guides
the eye; the reader should never have to search.
