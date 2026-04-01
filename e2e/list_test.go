package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestList_headerShowsContextAndPath(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myapi")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "myapi")
	bliss(t, proj, env, "add", "Do something")

	out, err := bliss(t, proj, env, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	for _, want := range []string{"bliss list", "Context:", "myapi", "Path:"} {
		if !strings.Contains(out, want) {
			t.Errorf("list header missing %q:\n%s", want, out)
		}
	}
}

func TestList_headerPersonalMode(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Personal task")

	out, err := bliss(t, dir, env, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	for _, want := range []string{"bliss list", "Personal"} {
		if !strings.Contains(out, want) {
			t.Errorf("personal list header missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Context:") {
		t.Errorf("personal list header must not contain 'Context:':\n%s", out)
	}
}

func TestList_headerFilteredShowsListName(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	bliss(t, proj, env, "add", "Do it this week", "-l", "this-week")

	out, err := bliss(t, proj, env, "list", "this-week")
	if err != nil {
		t.Fatalf("list this-week: %v\n%s", err, out)
	}
	for _, want := range []string{"bliss list", "Context:", "List:", "this-week", "Path:"} {
		if !strings.Contains(out, want) {
			t.Errorf("filtered list header missing %q:\n%s", want, out)
		}
	}
}

func TestList_itemNumberingFormat(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	for i := 0; i < 10; i++ {
		bliss(t, proj, env, "add", "Todo item", "-l", "today")
	}

	out, err := bliss(t, proj, env, "list", "today")
	if err != nil {
		t.Fatalf("list today: %v\n%s", err, out)
	}
	// Item 1: "  1  " (2 spaces + digit + 2 spaces)
	if !strings.Contains(out, "  1  ") {
		t.Errorf("item 1 should have '  1  ' prefix:\n%s", out)
	}
	// Item 10: " 10  " (1 space + 2 digits + 2 spaces)
	if !strings.Contains(out, " 10  ") {
		t.Errorf("item 10 should have ' 10  ' prefix:\n%s", out)
	}
}

func TestList_semanticOrder(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	bliss(t, proj, env, "add", "Bug fix", "-l", "bugs")
	bliss(t, proj, env, "add", "This week task", "-l", "this-week")
	bliss(t, proj, env, "add", "Today task", "-l", "today")
	bliss(t, proj, env, "add", "Incoming item")

	out, err := bliss(t, proj, env, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}

	lines := strings.Split(out, "\n")
	indexOf := func(s string) int {
		for i, l := range lines {
			if strings.Contains(l, s) {
				return i
			}
		}
		return -1
	}

	todayLine := indexOf("today")
	thisWeekLine := indexOf("this-week")
	bugsLine := indexOf("bugs")
	incomingLine := indexOf("incoming")

	if todayLine < 0 || thisWeekLine < 0 || bugsLine < 0 || incomingLine < 0 {
		t.Fatalf("missing expected sections:\n%s", out)
	}
	if todayLine >= thisWeekLine {
		t.Errorf("today (line %d) should appear before this-week (line %d)", todayLine, thisWeekLine)
	}
	if thisWeekLine >= bugsLine {
		t.Errorf("this-week (line %d) should appear before bugs (line %d)", thisWeekLine, bugsLine)
	}
	if bugsLine >= incomingLine {
		t.Errorf("bugs (line %d) should appear before incoming (line %d)", bugsLine, incomingLine)
	}
}

func TestList_unlistedTodosInIncoming(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	bliss(t, proj, env, "add", "Unlisted task")
	bliss(t, proj, env, "add", "Listed task", "-l", "today")

	out, err := bliss(t, proj, env, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "incoming") {
		t.Errorf("output should contain 'incoming' section:\n%s", out)
	}
	incomingIdx := strings.Index(out, "incoming")
	taskIdx := strings.Index(out, "Unlisted task")
	if taskIdx < incomingIdx {
		t.Errorf("unlisted task should appear after incoming header:\n%s", out)
	}
}

func TestList_emptyListsOmitted(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	bliss(t, proj, env, "add", "Today task", "-l", "today")

	out, err := bliss(t, proj, env, "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if strings.Contains(out, "this-week") {
		t.Errorf("empty 'this-week' list should not appear:\n%s", out)
	}
	if strings.Contains(out, "bugs") {
		t.Errorf("empty 'bugs' list should not appear:\n%s", out)
	}
}

func TestList_personalFlagInsideContext(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "myproject")
	bliss(t, proj, env, "add", "Context task", "-l", "today")
	bliss(t, home, env, "add", "Personal task", "-l", "today")

	out, err := bliss(t, proj, env, "list", "--personal")
	if err != nil {
		t.Fatalf("list --personal: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Personal task") {
		t.Errorf("output %q missing personal task", out)
	}
	if strings.Contains(out, "Context task") {
		t.Errorf("output %q must not show context task with --personal", out)
	}
	if !strings.Contains(out, "Personal") {
		t.Errorf("output %q missing 'Personal' in header", out)
	}
}

func TestList_all_spansAllContexts(t *testing.T) {
	home, env := blissEnv(t)
	proj1 := filepath.Join(home, "proj1")
	proj2 := filepath.Join(home, "proj2")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	bliss(t, proj1, env, "init", "--name", "Project One")
	bliss(t, proj2, env, "init", "--name", "Project Two")
	bliss(t, proj1, env, "add", "Todo in proj1")
	bliss(t, proj2, env, "add", "Todo in proj2")

	out, err := bliss(t, home, env, "list", "--all")
	if err != nil {
		t.Fatalf("list --all: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Todo in proj1") {
		t.Errorf("output missing proj1 todo: %s", out)
	}
	if !strings.Contains(out, "Todo in proj2") {
		t.Errorf("output missing proj2 todo: %s", out)
	}
}

func TestList_all_noPositionNumbers(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	bliss(t, proj, env, "add", "Task one", "-l", "today")
	bliss(t, proj, env, "add", "Task two", "-l", "today")

	out, err := bliss(t, proj, env, "list", "--all")
	if err != nil {
		t.Fatalf("list --all: %v\n%s", err, out)
	}
	if strings.Contains(out, "  1  ") {
		t.Errorf("--all output must not contain position numbers ('  1  '):\n%s", out)
	}
	if !strings.Contains(out, "Task one") || !strings.Contains(out, "Task two") {
		t.Errorf("--all output should contain todo titles:\n%s", out)
	}
}

func TestList_all_contextHeaders(t *testing.T) {
	home, env := blissEnv(t)
	proj1 := filepath.Join(home, "alpha")
	proj2 := filepath.Join(home, "beta")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	bliss(t, proj1, env, "init", "--name", "alpha")
	bliss(t, proj2, env, "init", "--name", "beta")
	bliss(t, proj1, env, "add", "Alpha task")
	bliss(t, proj2, env, "add", "Beta task")

	out, err := bliss(t, home, env, "list", "--all")
	if err != nil {
		t.Fatalf("list --all: %v\n%s", err, out)
	}
	for _, want := range []string{"Context:", "alpha", "beta", "Path:"} {
		if !strings.Contains(out, want) {
			t.Errorf("--all output missing %q:\n%s", want, out)
		}
	}
}

func TestList_contextFlagFromOutsideProject(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myservice")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	bliss(t, proj, env, "add", "Service task")

	// List from an unrelated directory using --context flag
	outside := t.TempDir()
	out, err := bliss(t, outside, env, "list", "--context", "myservice")
	if err != nil {
		t.Fatalf("list --context: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Service task") {
		t.Errorf("list --context output %q missing todo", out)
	}
	if !strings.Contains(out, "myservice") {
		t.Errorf("list --context output %q missing context name in header", out)
	}
}

func TestList_unknownContextOffersSync(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	// Write a .bliss-context referencing a non-existent context slug
	if err := os.WriteFile(filepath.Join(dir, ".bliss-context"), []byte("ghost-project\n"), 0644); err != nil {
		t.Fatalf("writing .bliss-context: %v", err)
	}

	// Decline sync offer — should error
	out, err := blissStdin(t, dir, env, "n\n", "list")
	if err == nil {
		t.Fatalf("expected error for unknown context, got: %s", out)
	}
	if !strings.Contains(out, "ghost-project") {
		t.Errorf("error output %q should mention the missing context name", out)
	}
}

