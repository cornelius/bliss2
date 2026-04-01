package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDone_confirmsTitle(t *testing.T) {
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

func TestDone_byUUID(t *testing.T) {
	home, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Delete by UUID")

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

func TestDone_personalMode(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Complete me")
	bliss(t, dir, env, "list")

	out, err := bliss(t, dir, env, "done", "1")
	if err != nil {
		t.Fatalf("done in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Done:") {
		t.Errorf("done output %q missing confirmation", out)
	}
}

func TestDone_sessionStability(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "First", "-l", "today")
	bliss(t, dir, env, "add", "Second", "-l", "today")
	bliss(t, dir, env, "add", "Third", "-l", "today")
	bliss(t, dir, env, "list")

	bliss(t, dir, env, "done", "1")

	// Position 3 must still resolve to "Third" — session is not renumbered.
	out, err := bliss(t, dir, env, "done", "3")
	if err != nil {
		t.Fatalf("done 3 after done 1: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Third") {
		t.Errorf("done 3 output %q: expected 'Third', session must not renumber", out)
	}
}

func TestMove_confirmsListAndTitle(t *testing.T) {
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

func TestMove_urgent(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "First in later", "-l", "later")
	bliss(t, dir, env, "add", "Second in later", "-l", "later")
	bliss(t, dir, env, "add", "Move me urgent") // lands in incoming
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

func TestMove_personalMode(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Move me")
	bliss(t, dir, env, "list")

	out, err := bliss(t, dir, env, "move", "1", "-l", "today")
	if err != nil {
		t.Fatalf("move in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Moved to") || !strings.Contains(out, "today") {
		t.Errorf("move output %q missing confirmation", out)
	}
}
