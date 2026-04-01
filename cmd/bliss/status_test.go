package main

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestStatusAlignment verifies the core design invariant: list data starts at
// the same visual column (44) for every row type, so counts align vertically.
//
// We measure this by rendering rows with no counts — when counts are empty
// the rendered string is exactly the fixed-width prefix.
func TestStatusAlignment(t *testing.T) {
	const wantWidth = 45

	cases := []struct {
		name string
		row  string
	}{
		{"active context", renderContextRow(true, "api", "/some/path", nil, 10)},
		{"inactive context", renderContextRow(false, "frontend", "/other/path", nil, 10)},
		{"personal (inactive)", renderPersonalRow(false, nil, 10)},
		{"personal (active)", renderPersonalRow(true, nil, 10)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lipgloss.Width(tc.row)
			if got != wantWidth {
				t.Errorf("prefix width = %d, want %d\nrow: %q", got, wantWidth, tc.row)
			}
		})
	}
}

// TestStatusActiveIndicator verifies the ">" marker appears only on the active row.
func TestStatusActiveIndicator(t *testing.T) {
	active := renderContextRow(true, "api", "/path", nil, 10)
	inactive := renderContextRow(false, "api", "/path", nil, 10)

	if !strings.Contains(active, ">") {
		t.Error("active context row must contain '>'")
	}
	if strings.Contains(inactive, ">") {
		t.Error("inactive context row must not contain '>'")
	}

	personalActive := renderPersonalRow(true, nil, 10)
	personalInactive := renderPersonalRow(false, nil, 10)

	if !strings.Contains(personalActive, ">") {
		t.Error("active personal row must contain '>'")
	}
	if strings.Contains(personalInactive, ">") {
		t.Error("inactive personal row must not contain '>'")
	}
}

// TestStatusLabels verifies each row type carries its entity-type label.
func TestStatusLabels(t *testing.T) {
	ctx := renderContextRow(false, "api", "/path", nil, 10)
	personal := renderPersonalRow(false, nil, 10)

	if !strings.Contains(ctx, "Context:") {
		t.Error("context row must contain 'Context:'")
	}
	if !strings.Contains(personal, "Personal:") {
		t.Error("personal row must contain 'Personal:'")
	}
}

// TestStatusPathTruncation verifies long paths are truncated with an ellipsis.
func TestStatusPathTruncation(t *testing.T) {
	longPath := "/very/long/directory/path/that/exceeds/twenty/chars"
	row := renderContextRow(false, "ctx", longPath, nil, 10)

	if strings.Contains(row, longPath) {
		t.Error("long path should be truncated in the output")
	}
	if !strings.Contains(row, "…") {
		t.Error("truncated path should contain ellipsis '…'")
	}
}

// TestStatusShortenHomePath verifies the home directory is replaced with "~".
func TestStatusShortenHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	path := home + "/projects/myapp"
	got := shortenHomePath(path)
	if !strings.HasPrefix(got, "~") {
		t.Errorf("shortenHomePath(%q) = %q, want ~ prefix", path, got)
	}
	if strings.Contains(got, home) {
		t.Errorf("shortenHomePath(%q) = %q, still contains home dir", path, got)
	}
}

// TestStatusSortedCounts verifies the semantic ordering:
// today → this-week → next-week → later → custom → bugs → incoming.
func TestStatusSortedCounts(t *testing.T) {
	input := []listCount{
		{"incoming", 1},
		{"bugs", 2},
		{"my-feature", 3},
		{"later", 4},
		{"next-week", 5},
		{"this-week", 6},
		{"today", 7},
	}
	got := sortedCounts(input)
	want := []string{"today", "this-week", "next-week", "later", "my-feature", "bugs", "incoming"}
	for i, w := range want {
		if got[i].name != w {
			t.Errorf("sortedCounts[%d] = %q, want %q", i, got[i].name, w)
		}
	}
}

// TestStatusRenderCounts verifies the "name: count  name: count" format.
func TestStatusRenderCounts(t *testing.T) {
	counts := []listCount{{"today", 3}, {"bugs", 2}}
	got := renderCounts(counts)

	for _, want := range []string{"today:", "bugs:", "3", "2"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderCounts output %q missing %q", got, want)
		}
	}
}
