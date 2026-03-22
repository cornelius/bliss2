package e2e

import (
	"strings"
	"testing"
)

func TestSync_noRemote(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "A task")

	out, err := bliss(t, dir, env, "sync")
	if err == nil {
		t.Fatalf("expected error when no remote configured, got: %s", out)
	}
	if !strings.Contains(out, "no remote") {
		t.Errorf("error output %q should mention 'no remote'", out)
	}
}

func TestSync_alreadyUpToDate(t *testing.T) {
	home, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "A task")
	setupRemote(t, home)

	out, err := bliss(t, dir, env, "sync")
	if err != nil {
		t.Fatalf("sync when up to date: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Already up to date") {
		t.Errorf("output %q missing 'Already up to date'", out)
	}
}

func TestSync_pushesLocalCommits(t *testing.T) {
	home, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Initial task")
	setupRemote(t, home)

	// Add a new todo — creates a commit that hasn't been pushed yet.
	bliss(t, dir, env, "add", "New task")

	out, err := bliss(t, dir, env, "sync")
	if err != nil {
		t.Fatalf("sync push: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Pushed") {
		t.Errorf("output %q missing 'Pushed'", out)
	}
	if !strings.Contains(out, "commit") {
		t.Errorf("output %q missing 'commit'", out)
	}
}

func TestSync_pullsRemoteCommits(t *testing.T) {
	home, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Initial task")
	bare := setupRemote(t, home)

	// Advance the remote without touching the local store.
	advanceRemote(t, bare)

	out, err := bliss(t, dir, env, "sync")
	if err != nil {
		t.Fatalf("sync pull: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Pulled") {
		t.Errorf("output %q missing 'Pulled'", out)
	}
	if !strings.Contains(out, "commit") {
		t.Errorf("output %q missing 'commit'", out)
	}
}

func TestSync_divergedReportsError(t *testing.T) {
	home, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Initial task")
	bare := setupRemote(t, home)

	// Advance remote and local independently to create divergence.
	advanceRemote(t, bare)
	bliss(t, dir, env, "add", "Local-only task")

	out, err := bliss(t, dir, env, "sync")
	if err == nil {
		t.Fatalf("expected error on diverged store, got: %s", out)
	}
	if !strings.Contains(out, "diverged") {
		t.Errorf("error output %q should mention 'diverged'", out)
	}
}

func TestSync_singularCommitWord(t *testing.T) {
	home, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Initial task")
	setupRemote(t, home)
	bliss(t, dir, env, "add", "One new task") // exactly 1 commit ahead

	out, err := bliss(t, dir, env, "sync")
	if err != nil {
		t.Fatalf("sync: %v\n%s", err, out)
	}
	if strings.Contains(out, "commits") {
		t.Errorf("output %q should use singular 'commit' not 'commits' for 1 commit", out)
	}
	if !strings.Contains(out, "1 commit") {
		t.Errorf("output %q should say '1 commit'", out)
	}
}
