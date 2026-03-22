package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Integration tests (use the built binary) ──────────────────────────────────

// TestWorkflow_addNoList verifies "Added:" confirmation without a list.
func TestWorkflow_addNoList(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "add", "My task")
	if err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Added:") {
		t.Errorf("output %q missing 'Added:' label", out)
	}
	if !strings.Contains(out, "My task") {
		t.Errorf("output %q missing title", out)
	}
	// No brackets — new format uses plain list name.
	if strings.Contains(out, "[") {
		t.Errorf("output %q must not contain brackets", out)
	}
}

// TestWorkflow_addToList verifies "Added to listname:" confirmation.
func TestWorkflow_addToList(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "add", "My task", "-l", "today")
	if err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Added to") {
		t.Errorf("output %q missing 'Added to' phrase", out)
	}
	if !strings.Contains(out, "today") {
		t.Errorf("output %q missing list name", out)
	}
	if !strings.Contains(out, "My task") {
		t.Errorf("output %q missing title", out)
	}
	if strings.Contains(out, "[") {
		t.Errorf("output %q must not contain brackets", out)
	}
}

// TestWorkflow_done verifies "Done:" confirmation.
func TestWorkflow_done(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Finish me")
	bliss(t, dir, env, "list")

	out, err := bliss(t, dir, env, "done", "1")
	if err != nil {
		t.Fatalf("done: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Done:") {
		t.Errorf("output %q missing 'Done:' label", out)
	}
	if !strings.Contains(out, "Finish me") {
		t.Errorf("output %q missing title", out)
	}
}

// TestWorkflow_move verifies "Moved to listname:" confirmation without brackets.
func TestWorkflow_move(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Move me")
	bliss(t, dir, env, "list")

	out, err := bliss(t, dir, env, "move", "1", "-l", "later")
	if err != nil {
		t.Fatalf("move: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Moved to") {
		t.Errorf("output %q missing 'Moved to' phrase", out)
	}
	if !strings.Contains(out, "later") {
		t.Errorf("output %q missing list name", out)
	}
	if strings.Contains(out, "[") {
		t.Errorf("output %q must not contain brackets", out)
	}
}

// TestWorkflow_init verifies "Initialized:" shows name and path without UUID.
func TestWorkflow_init(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myservice")
	os.MkdirAll(proj, 0755)

	out, err := bliss(t, proj, env, "init", "--name", "myservice")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Initialized") {
		t.Errorf("output %q missing 'Initialized' label", out)
	}
	if !strings.Contains(out, "Context:") {
		t.Errorf("output %q missing 'Context:' label", out)
	}
	if !strings.Contains(out, "Path:") {
		t.Errorf("output %q missing 'Path:' label", out)
	}
	if !strings.Contains(out, "myservice") {
		t.Errorf("output %q missing context name", out)
	}
	// UUID must not appear — it's an internal detail.
	if strings.Count(out, "-") >= 4 {
		// UUIDs have 4 hyphens; a path or name will not
		t.Errorf("output %q appears to contain UUID (4+ hyphens)", out)
	}
}

// TestDone_uuid verifies bliss done accepts a UUID directly, bypassing the session.
func TestDone_uuid(t *testing.T) {
	home, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Delete by UUID")

	// Find the todo file in the personal store to get the UUID.
	matches, err := filepath.Glob(filepath.Join(home, ".bliss2", "todos", "*.md"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("could not find todo file in store: %v", err)
	}
	uuid := strings.TrimSuffix(filepath.Base(matches[0]), ".md")

	// No bliss list — UUID bypasses the session.
	out, err := bliss(t, dir, env, "done", uuid)
	if err != nil {
		t.Fatalf("done by UUID: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Done:") {
		t.Errorf("output %q missing 'Done:' confirmation", out)
	}
	if !strings.Contains(out, "Delete by UUID") {
		t.Errorf("output %q missing todo title", out)
	}
}

// TestAdd_urgent verifies --urgent places the todo at the top of the list.
func TestAdd_urgent(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Normal task", "-l", "today")
	bliss(t, dir, env, "add", "Urgent task", "-l", "today", "--urgent")

	out, err := bliss(t, dir, env, "list", "today")
	if err != nil {
		t.Fatalf("list today: %v\n%s", err, out)
	}

	urgentIdx := strings.Index(out, "Urgent task")
	normalIdx := strings.Index(out, "Normal task")
	if urgentIdx < 0 || normalIdx < 0 {
		t.Fatalf("missing tasks in output:\n%s", out)
	}
	if urgentIdx >= normalIdx {
		t.Errorf("urgent task should appear before normal task:\n%s", out)
	}
}

// TestMove_urgent verifies --urgent places the moved todo at the top of the target list.
func TestMove_urgent(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "First in later", "-l", "later")
	bliss(t, dir, env, "add", "Second in later", "-l", "later")
	bliss(t, dir, env, "add", "Move me urgent") // lands in inbox
	bliss(t, dir, env, "list")

	out, err := bliss(t, dir, env, "move", "3", "-l", "later", "--urgent")
	if err != nil {
		t.Fatalf("move --urgent: %v\n%s", err, out)
	}

	out, err = bliss(t, dir, env, "list", "later")
	if err != nil {
		t.Fatalf("list later: %v\n%s", err, out)
	}

	movedIdx := strings.Index(out, "Move me urgent")
	firstIdx := strings.Index(out, "First in later")
	if movedIdx < 0 || firstIdx < 0 {
		t.Fatalf("missing todos in output:\n%s", out)
	}
	if movedIdx >= firstIdx {
		t.Errorf("urgently moved todo should appear before existing todos:\n%s", out)
	}
}

// TestHistory_header verifies bliss history opens with a "bliss history" header.
func TestHistory_header(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "proj")
	bliss(t, proj, env, "add", "A task")

	out, err := bliss(t, proj, env, "history")
	if err != nil {
		t.Fatalf("history: %v\n%s", err, out)
	}

	if !strings.Contains(out, "bliss history") {
		t.Errorf("output %q missing 'bliss history' header", out)
	}
	if !strings.Contains(out, "Context:") {
		t.Errorf("output %q missing 'Context:' label", out)
	}
}

// TestHistory_personalHeader verifies personal mode shows "Personal" not "Context:".
func TestHistory_personalHeader(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Personal task")

	out, err := bliss(t, dir, env, "history")
	if err != nil {
		t.Fatalf("history: %v\n%s", err, out)
	}

	if !strings.Contains(out, "bliss history") {
		t.Errorf("output %q missing 'bliss history' header", out)
	}
	if !strings.Contains(out, "Personal") {
		t.Errorf("output %q missing 'Personal' in header", out)
	}
	if strings.Contains(out, "Context:") {
		t.Errorf("output %q should not contain 'Context:' in personal mode", out)
	}
}


// TestHistory_allHeader verifies bliss history --all header format.
func TestHistory_allHeader(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Something")

	out, err := bliss(t, dir, env, "history", "--all")
	if err != nil {
		t.Fatalf("history --all: %v\n%s", err, out)
	}

	if !strings.Contains(out, "bliss history --all") {
		t.Errorf("output %q missing 'bliss history --all' header", out)
	}
}

// TestHistory_contextFiltering verifies bliss history only shows entries for
// the current context, not for other contexts or personal.
func TestHistory_contextFiltering(t *testing.T) {
	home, env := blissEnv(t)
	proj1 := filepath.Join(home, "alpha")
	proj2 := filepath.Join(home, "beta")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	bliss(t, proj1, env, "init", "--name", "alpha")
	bliss(t, proj2, env, "init", "--name", "beta")
	bliss(t, proj1, env, "add", "Alpha task")
	bliss(t, proj2, env, "add", "Beta task")

	out, err := bliss(t, proj1, env, "history")
	if err != nil {
		t.Fatalf("history in alpha: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Alpha task") {
		t.Errorf("alpha history missing 'Alpha task':\n%s", out)
	}
	if strings.Contains(out, "Beta task") {
		t.Errorf("alpha history should not contain 'Beta task':\n%s", out)
	}
}

// TestHistory_personal verifies --personal shows only personal commits.
func TestHistory_personal(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "proj")
	bliss(t, proj, env, "add", "Context task")
	bliss(t, home, env, "add", "Personal task") // no context → personal

	out, err := bliss(t, proj, env, "history", "--personal")
	if err != nil {
		t.Fatalf("history --personal: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Personal task") {
		t.Errorf("--personal history missing 'Personal task':\n%s", out)
	}
	if strings.Contains(out, "Context task") {
		t.Errorf("--personal history should not contain 'Context task':\n%s", out)
	}
}

// TestHistory_allContextLabel verifies --all includes a context label column.
func TestHistory_allContextLabel(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "alpha")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "alpha")
	bliss(t, proj, env, "add", "Context task")
	bliss(t, home, env, "add", "Personal task")

	out, err := bliss(t, proj, env, "history", "--all")
	if err != nil {
		t.Fatalf("history --all: %v\n%s", err, out)
	}

	// --all should label context entries with the context name.
	if !strings.Contains(out, "alpha") {
		t.Errorf("--all output missing context label 'alpha':\n%s", out)
	}
	// --all should label personal entries.
	if !strings.Contains(out, "personal") {
		t.Errorf("--all output missing 'personal' label:\n%s", out)
	}
}

// TestHistory_isoTimestamp verifies entries use friendly ISO datetime format.
func TestHistory_isoTimestamp(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "A task")

	out, err := bliss(t, dir, env, "history")
	if err != nil {
		t.Fatalf("history: %v\n%s", err, out)
	}

	// Should contain a date in 2006-01-02 format (year-month-day).
	// Match any entry line with an ISO-style date.
	hasISO := false
	for _, line := range strings.Split(out, "\n") {
		if len(line) >= 10 && line[4] == '-' && line[7] == '-' {
			hasISO = true
			break
		}
	}
	if !hasISO {
		t.Errorf("history output has no ISO date (YYYY-MM-DD) in entries:\n%s", out)
	}
}
