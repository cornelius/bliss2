package main

import (
	"strings"
	"testing"
)

// Unit tests for pure list-rendering functions.
// E2e tests for bliss list are in cmd/bliss/e2e/list_test.go.

func TestListSortKey(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{"today", 0},
		{"this-week", 1},
		{"next-week", 2},
		{"later", 3},
		{"my-feature", 4},
		{"sprint-1", 4},
		{"bugs", 5},
		{"incoming", 9},
	}
	for _, tc := range cases {
		got := listSortKey(tc.name)
		if got != tc.want {
			t.Errorf("listSortKey(%q) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestSortListNames(t *testing.T) {
	input := []string{"incoming", "bugs", "sprint-1", "later", "next-week", "this-week", "today"}
	got := sortListNames(input)
	want := []string{"today", "this-week", "next-week", "later", "sprint-1", "bugs", "incoming"}
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
	const dash3 = "───"

	cases := []struct {
		pos    int
		name   string
		want2  string
		noWant string
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
