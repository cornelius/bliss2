package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Unit tests (pure functions, no store needed) ───────────────────────────────

// TestListSortKey verifies the semantic priority mapping.
func TestListSortKey(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{"today", 0},
		{"this-week", 1},
		{"next-week", 2},
		{"later", 3},
		{"my-feature", 4}, // custom
		{"sprint-1", 4},   // custom
		{"bugs", 5},
		{"inbox", 9},
	}
	for _, tc := range cases {
		got := listSortKey(tc.name)
		if got != tc.want {
			t.Errorf("listSortKey(%q) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestSortListNames verifies full semantic ordering with a mix of standard and custom names.
func TestSortListNames(t *testing.T) {
	input := []string{"inbox", "bugs", "sprint-1", "later", "next-week", "this-week", "today"}
	got := sortListNames(input)
	want := []string{"today", "this-week", "next-week", "later", "sprint-1", "bugs", "inbox"}
	if len(got) != len(want) {
		t.Fatalf("sortListNames returned %d items, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("sortListNames[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestListSectionDelim_formats verifies delimiter alignment with the number field.
//
// 1-digit zone (pos < 10):  "  ──"  (2 spaces + 2 em-dashes)
// 2-digit zone (pos >= 10): " ───"  (1 space  + 3 em-dashes)
func TestListSectionDelim_formats(t *testing.T) {
	const dash3 = "───" // 3 em-dashes — only appears in 2-digit zone

	cases := []struct {
		pos    int
		name   string
		want2  string // must contain
		noWant string // must not contain
	}{
		{1, "backlog", "  ──", dash3},
		{9, "backlog", "  ──", dash3},
		{10, "nice to fix", " ───", ""},
		{99, "nice to fix", " ───", ""},
	}

	for _, tc := range cases {
		got := listSectionDelim(tc.pos, tc.name)
		if !strings.Contains(got, tc.want2) {
			t.Errorf("listSectionDelim(%d, %q) = %q, want prefix %q", tc.pos, tc.name, got, tc.want2)
		}
		if tc.noWant != "" && strings.Contains(got, tc.noWant) {
			t.Errorf("listSectionDelim(%d, %q) = %q, should not contain %q", tc.pos, tc.name, got, tc.noWant)
		}
		if !strings.Contains(got, tc.name) {
			t.Errorf("listSectionDelim(%d, %q) = %q, missing section name", tc.pos, tc.name, got)
		}
	}
}

// TestListSectionDelim_unnamed verifies unnamed delimiters (empty name) have no trailing text.
func TestListSectionDelim_unnamed(t *testing.T) {
	got1 := listSectionDelim(1, "")
	if strings.Contains(got1, " ───") {
		t.Errorf("unnamed 1-digit delim %q should not contain 3-dash prefix", got1)
	}
	if !strings.Contains(got1, "──") {
		t.Errorf("unnamed 1-digit delim %q should contain em-dashes", got1)
	}

	got2 := listSectionDelim(10, "")
	if !strings.Contains(got2, "───") {
		t.Errorf("unnamed 2-digit delim %q should contain 3 em-dashes", got2)
	}
}

// ── Integration tests (use the built binary) ──────────────────────────────────

// TestList_headerContext verifies the header line for context mode.
func TestList_headerContext(t *testing.T) {
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

// TestList_headerPersonal verifies the header line in personal mode.
func TestList_headerPersonal(t *testing.T) {
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
		t.Errorf("personal list header should not contain 'Context:':\n%s", out)
	}
}

// TestList_headerFiltered verifies the "List:" label appears when filtering by list name.
func TestList_headerFiltered(t *testing.T) {
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

// TestList_itemNumbering verifies the %3d  format: single-digit items get "  N  "
// prefix and double-digit items get " NN  " prefix.
func TestList_itemNumbering(t *testing.T) {
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
	// Item 1 should NOT use the 2-digit format
	if strings.Contains(out, " 1  ") && !strings.Contains(out, "  1  ") {
		t.Errorf("item 1 should use 1-digit format '  1  ', not ' 1  ':\n%s", out)
	}
}

// TestList_semanticOrder verifies lists appear in semantic order:
// today → this-week → bugs → inbox.
func TestList_semanticOrder(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	bliss(t, proj, env, "add", "Bug fix", "-l", "bugs")
	bliss(t, proj, env, "add", "This week task", "-l", "this-week")
	bliss(t, proj, env, "add", "Today task", "-l", "today")
	bliss(t, proj, env, "add", "Inbox item") // no list → inbox

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
	inboxLine := indexOf("inbox")

	if todayLine < 0 || thisWeekLine < 0 || bugsLine < 0 || inboxLine < 0 {
		t.Fatalf("missing expected sections:\n%s", out)
	}
	if todayLine >= thisWeekLine {
		t.Errorf("today (line %d) should appear before this-week (line %d)", todayLine, thisWeekLine)
	}
	if thisWeekLine >= bugsLine {
		t.Errorf("this-week (line %d) should appear before bugs (line %d)", thisWeekLine, bugsLine)
	}
	if bugsLine >= inboxLine {
		t.Errorf("bugs (line %d) should appear before inbox (line %d)", bugsLine, inboxLine)
	}
}

// TestList_inboxSection verifies todos without a list appear under "inbox".
func TestList_inboxSection(t *testing.T) {
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

	if !strings.Contains(out, "inbox") {
		t.Errorf("output should contain 'inbox' section:\n%s", out)
	}
	if !strings.Contains(out, "Unlisted task") {
		t.Errorf("unlisted task should appear in output:\n%s", out)
	}

	// Verify the unlisted task appears after "inbox" header.
	inboxIdx := strings.Index(out, "inbox")
	taskIdx := strings.Index(out, "Unlisted task")
	if taskIdx < inboxIdx {
		t.Errorf("unlisted task should appear after inbox header:\n%s", out)
	}
}

// TestList_emptyListsOmitted verifies lists with zero todos are not shown.
func TestList_emptyListsOmitted(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init")
	bliss(t, proj, env, "add", "Today task", "-l", "today")
	// "this-week" and "bugs" have no todos.

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

// TestList_personalFlagInsideContext verifies --personal shows only personal
// todos and excludes context todos when run from inside a context.
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

// TestList_all_noNumbers verifies --all output has no position numbers.
func TestList_all_noNumbers(t *testing.T) {
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

	// In --all mode items are indented "  title" with no number.
	// Position numbers would look like "  1  " or " 10  ".
	if strings.Contains(out, "  1  ") {
		t.Errorf("--all output should not contain position numbers ('  1  '):\n%s", out)
	}
	if !strings.Contains(out, "Task one") || !strings.Contains(out, "Task two") {
		t.Errorf("--all output should contain todo titles:\n%s", out)
	}
}

// TestList_all_contextHeaders verifies --all shows "Context:" headers for each context.
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
