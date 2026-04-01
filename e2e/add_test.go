package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdd_titleWithApostrophe(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	if _, err := bliss(t, dir, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	out, err := bliss(t, dir, env, "add", "Fix John's bug")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out, "Fix John's bug") {
		t.Errorf("output %q does not contain title", out)
	}

	out, err = bliss(t, dir, env, "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "Fix John's bug") {
		t.Errorf("list output %q does not contain title", out)
	}
}

func TestAdd_titleWithDoubleQuotes(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	if _, err := bliss(t, dir, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	title := `He said "hello"`
	out, err := bliss(t, dir, env, "add", title)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out, title) {
		t.Errorf("output %q does not contain title", out)
	}

	out, err = bliss(t, dir, env, "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, title) {
		t.Errorf("list output %q does not contain title", out)
	}
}

func TestAdd_toIncoming(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "add", "Incoming task")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out, "Added:") {
		t.Errorf("output %q missing 'Added:' label", out)
	}
	if !strings.Contains(out, "Incoming task") {
		t.Errorf("output %q missing title", out)
	}
	if strings.Contains(out, "[") {
		t.Errorf("output %q must not contain brackets", out)
	}
}

func TestAdd_toNamedList(t *testing.T) {
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

func TestAdd_stdinTitle(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	if _, err := bliss(t, dir, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	title := "Fix John's bug"
	out, err := blissStdin(t, dir, env, title+"\n", "add")
	if err != nil {
		t.Fatalf("add via stdin: %v", err)
	}
	if !strings.Contains(out, title) {
		t.Errorf("output %q does not contain title", out)
	}

	out, err = bliss(t, dir, env, "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, title) {
		t.Errorf("list output %q does not contain title", out)
	}
}

func TestAdd_stdinTitleWithList(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	if _, err := bliss(t, dir, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	title := `He said "hello"`
	out, err := blissStdin(t, dir, env, title+"\n", "add", "-l", "today")
	if err != nil {
		t.Fatalf("add via stdin: %v", err)
	}
	if !strings.Contains(out, "today") {
		t.Errorf("output %q should mention target list", out)
	}
	if !strings.Contains(out, title) {
		t.Errorf("output %q does not contain title", out)
	}
}

func TestAdd_personalMode(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir() // no bliss init

	out, err := bliss(t, dir, env, "add", "Buy oat milk")
	if err != nil {
		t.Fatalf("add in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Buy oat milk") {
		t.Errorf("add output %q missing title", out)
	}

	out, err = bliss(t, dir, env, "list")
	if err != nil {
		t.Fatalf("list in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Buy oat milk") {
		t.Errorf("list output %q missing title", out)
	}
}

func TestAdd_personalModeToList(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "add", "Urgent task", "-l", "today")
	if err != nil {
		t.Fatalf("add to list in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "today") {
		t.Errorf("output %q missing list name", out)
	}

	out, err = bliss(t, dir, env, "list", "today")
	if err != nil {
		t.Fatalf("list today in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Urgent task") {
		t.Errorf("list today output %q missing title", out)
	}
}

func TestAdd_contextFlagAddsToNamedContext(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myservice")
	os.MkdirAll(proj, 0755)

	if _, err := bliss(t, proj, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Add to context from an unrelated directory using --context flag
	outside := t.TempDir()
	out, err := bliss(t, outside, env, "add", "--context", "myservice", "Task from outside")
	if err != nil {
		t.Fatalf("add --context: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Task from outside") {
		t.Errorf("add output %q missing title", out)
	}

	// List from within the project directory must show the todo
	out, err = bliss(t, proj, env, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Task from outside") {
		t.Errorf("list output %q missing todo added via --context", out)
	}
}

func TestAdd_contextFlagOverridesCWD(t *testing.T) {
	home, env := blissEnv(t)
	proj1 := filepath.Join(home, "alpha")
	proj2 := filepath.Join(home, "beta")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	bliss(t, proj1, env, "init", "--name", "alpha")
	bliss(t, proj2, env, "init", "--name", "beta")

	// Inside alpha, add to beta via --context
	out, err := bliss(t, proj1, env, "add", "--context", "beta", "Beta task")
	if err != nil {
		t.Fatalf("add --context beta from inside alpha: %v\n%s", err, out)
	}

	// beta context must have the todo
	out, err = bliss(t, proj2, env, "list")
	if err != nil {
		t.Fatalf("list beta: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Beta task") {
		t.Errorf("beta list %q missing 'Beta task'", out)
	}

	// alpha context must not have the todo
	out, err = bliss(t, proj1, env, "list")
	if err != nil {
		t.Fatalf("list alpha: %v\n%s", err, out)
	}
	if strings.Contains(out, "Beta task") {
		t.Errorf("alpha list %q must not contain 'Beta task'", out)
	}
}
