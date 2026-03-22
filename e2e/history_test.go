package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistory_headerInsideContext(t *testing.T) {
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

func TestHistory_headerPersonalMode(t *testing.T) {
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
		t.Errorf("output %q must not contain 'Context:' in personal mode", out)
	}
}

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

func TestHistory_onlyShowsCurrentContext(t *testing.T) {
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
		t.Errorf("alpha history must not contain 'Beta task':\n%s", out)
	}
}

func TestHistory_personalFlagShowsOnlyPersonal(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "proj")
	bliss(t, proj, env, "add", "Context task")
	bliss(t, home, env, "add", "Personal task")

	out, err := bliss(t, proj, env, "history", "--personal")
	if err != nil {
		t.Fatalf("history --personal: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Personal task") {
		t.Errorf("--personal history missing 'Personal task':\n%s", out)
	}
	if strings.Contains(out, "Context task") {
		t.Errorf("--personal history must not contain 'Context task':\n%s", out)
	}
}

func TestHistory_allIncludesContextLabels(t *testing.T) {
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
	if !strings.Contains(out, "alpha") {
		t.Errorf("--all output missing context label 'alpha':\n%s", out)
	}
	if !strings.Contains(out, "personal") {
		t.Errorf("--all output missing 'personal' label:\n%s", out)
	}
}

func TestHistory_isoTimestamp(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "A task")

	out, err := bliss(t, dir, env, "history")
	if err != nil {
		t.Fatalf("history: %v\n%s", err, out)
	}
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
