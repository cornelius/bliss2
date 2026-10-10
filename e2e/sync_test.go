package e2e

import (
	"os"
	"path/filepath"
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

func TestSync_divergedRebasesAndPushes(t *testing.T) {
	homeA, envA := blissEnv(t)
	dirA := t.TempDir()

	bliss(t, dirA, envA, "add", "Initial task")
	bare := setupRemote(t, homeA)
	homeB, envB := cloneMachine(t, bare)
	dirB := t.TempDir()

	bliss(t, dirA, envA, "add", "Task from A")
	if out, err := bliss(t, dirA, envA, "sync"); err != nil {
		t.Fatalf("sync A: %v\n%s", err, out)
	}

	bliss(t, dirB, envB, "add", "Task from B")
	out, err := bliss(t, dirB, envB, "sync")
	if err != nil {
		t.Fatalf("sync of a diverged store must succeed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Pulled 1 commit") || !strings.Contains(out, "Pushed 1 commit") {
		t.Errorf("output %q should report both the pulled and the pushed commit", out)
	}
	if merges := git(t, filepath.Join(homeB, ".bliss2"), "rev-list", "--merges", "--count", "HEAD"); merges != "0" {
		t.Errorf("sync of a diverged store should keep history linear, found %s merge commits", merges)
	}

	if out, err := bliss(t, dirA, envA, "sync"); err != nil {
		t.Fatalf("sync A: %v\n%s", err, out)
	}
	list, _ := bliss(t, dirA, envA, "list")
	for _, title := range []string{"Initial task", "Task from A", "Task from B"} {
		if !strings.Contains(list, title) {
			t.Errorf("after both machines sync, list on A should contain %q, got:\n%s", title, list)
		}
	}
}

func TestSync_conflictingDivergenceLeavesStoreUnchanged(t *testing.T) {
	homeA, envA := blissEnv(t)
	dirA := t.TempDir()

	bliss(t, dirA, envA, "add", "Initial task")
	bare := setupRemote(t, homeA)
	homeB, envB := cloneMachine(t, bare)
	dirB := t.TempDir()
	storeB := filepath.Join(homeB, ".bliss2")

	// Both machines append to the same list file, which conflicts.
	bliss(t, dirA, envA, "add", "--list", "next", "Task from A")
	if out, err := bliss(t, dirA, envA, "sync"); err != nil {
		t.Fatalf("sync A: %v\n%s", err, out)
	}
	bliss(t, dirB, envB, "add", "--list", "next", "Task from B")
	before := git(t, storeB, "rev-parse", "HEAD")

	out, err := bliss(t, dirB, envB, "sync")
	if err == nil {
		t.Fatalf("expected error on conflicting divergence, got: %s", out)
	}
	if !strings.Contains(out, "conflict") {
		t.Errorf("error output %q should mention the conflict", out)
	}
	if after := git(t, storeB, "rev-parse", "HEAD"); after != before {
		t.Errorf("failed sync should leave local commits untouched: HEAD moved from %s to %s", before, after)
	}
	if status := git(t, storeB, "status", "--porcelain"); status != "" {
		t.Errorf("failed sync should leave a clean working tree, got:\n%s", status)
	}
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(storeB, ".git", name)); err == nil {
			t.Errorf("failed sync should not leave a rebase in progress (found .git/%s)", name)
		}
	}
}

func TestSync_divergedWithUncommittedChangesReportsGitError(t *testing.T) {
	homeA, envA := blissEnv(t)
	dirA := t.TempDir()

	bliss(t, dirA, envA, "add", "Initial task")
	bare := setupRemote(t, homeA)
	homeB, envB := cloneMachine(t, bare)
	dirB := t.TempDir()
	storeB := filepath.Join(homeB, ".bliss2")

	bliss(t, dirA, envA, "add", "Task from A")
	if out, err := bliss(t, dirA, envA, "sync"); err != nil {
		t.Fatalf("sync A: %v\n%s", err, out)
	}
	bliss(t, dirB, envB, "add", "Task from B")

	// A hand edit in the store that was never committed blocks the rebase.
	if err := os.WriteFile(filepath.Join(storeB, "bliss2-was-here"), []byte("edited\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := bliss(t, dirB, envB, "sync")
	if err == nil {
		t.Fatalf("expected error when the store has uncommitted changes, got: %s", out)
	}
	if strings.Contains(out, "conflict") {
		t.Errorf("error output %q should not claim a conflict when the rebase never started", out)
	}
	if !strings.Contains(out, "rebase failed") {
		t.Errorf("error output %q should say the rebase failed and pass on git's reason", out)
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
