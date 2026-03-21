package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShow_noHeader verifies bliss show has no header line.
func TestShow_noHeader(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "myproject")
	bliss(t, proj, env, "add", "A task", "-l", "today")

	out, err := bliss(t, proj, env, "show")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	if strings.Contains(out, "bliss show") {
		t.Errorf("output %q must not contain a header line", out)
	}
	if strings.Contains(out, "Context:") {
		t.Errorf("output %q must not contain 'Context:' label", out)
	}
	if strings.Contains(out, "Path:") {
		t.Errorf("output %q must not contain 'Path:' label", out)
	}
}

// TestShow_inboxOmittedWhenEmpty verifies inbox is not shown when empty.
func TestShow_inboxOmittedWhenEmpty(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	// Add to a named list (not inbox).
	bliss(t, dir, env, "add", "A task", "-l", "today")

	out, err := bliss(t, dir, env, "show")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	if strings.Contains(out, "inbox") {
		t.Errorf("output %q must not show inbox when empty", out)
	}
}

// TestShow_inboxShownWhenNonEmpty verifies inbox appears when it has items.
func TestShow_inboxShownWhenNonEmpty(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	// Add to inbox (no -l flag).
	bliss(t, dir, env, "add", "Inbox task")

	out, err := bliss(t, dir, env, "show")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	if !strings.Contains(out, "inbox") {
		t.Errorf("output %q must show inbox when non-empty", out)
	}
	if !strings.Contains(out, "Inbox task") {
		t.Errorf("output %q missing inbox task title", out)
	}
}

// TestShow_noPersonalWhenInContext verifies personal todos are not shown inside a context.
func TestShow_noPersonalWhenInContext(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "myproject")
	bliss(t, proj, env, "add", "Context task", "-l", "today")
	// Add a personal todo from outside the context.
	bliss(t, home, env, "add", "Personal task", "-l", "today")

	out, err := bliss(t, proj, env, "show")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	if strings.Contains(out, "Personal task") {
		t.Errorf("output %q must not show personal todos inside a context", out)
	}
	if !strings.Contains(out, "Context task") {
		t.Errorf("output %q missing context task", out)
	}
}

// TestShow_positionNumbers verifies todos have position numbers.
func TestShow_positionNumbers(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "First task", "-l", "today")
	bliss(t, dir, env, "add", "Second task", "-l", "today")

	out, err := bliss(t, dir, env, "show")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	if !strings.Contains(out, "1") {
		t.Errorf("output %q missing position number 1", out)
	}
	if !strings.Contains(out, "2") {
		t.Errorf("output %q missing position number 2", out)
	}
}

// TestShow_doneWorksAfterShow verifies session mapping is written so bliss done works.
func TestShow_doneWorksAfterShow(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Complete me", "-l", "today")
	bliss(t, dir, env, "show")

	out, err := bliss(t, dir, env, "done", "1")
	if err != nil {
		t.Fatalf("done after show: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Done:") {
		t.Errorf("done output %q missing 'Done:' confirmation", out)
	}
}

// TestShow_empty verifies the empty state message.
func TestShow_empty(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "show")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	if !strings.Contains(out, "All done. Nothing left to do.") {
		t.Errorf("output %q missing empty state message", out)
	}
}

// TestShow_filteredList verifies bliss show <list-name> shows only that list
// with no list name header and no indent.
func TestShow_filteredList(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Today task", "-l", "today")
	bliss(t, dir, env, "add", "Later task", "-l", "later")

	out, err := bliss(t, dir, env, "show", "today")
	if err != nil {
		t.Fatalf("show today: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Today task") {
		t.Errorf("output %q missing 'Today task'", out)
	}
	if strings.Contains(out, "Later task") {
		t.Errorf("output %q must not show 'Later task' when filtered to today", out)
	}
	// No list name shown in filtered mode.
	if strings.Contains(out, "today") {
		t.Errorf("output %q must not show list name in filtered mode", out)
	}
	// No header.
	if strings.Contains(out, "List:") {
		t.Errorf("output %q must not contain 'List:' label", out)
	}
	// Number starts at column 0 — line should start with "1  ".
	if !strings.HasPrefix(strings.TrimLeft(out, "\n"), "1  ") {
		t.Errorf("output %q: filtered items must start at column 0", out)
	}
}
