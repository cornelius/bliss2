package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary into a temp dir once for all tests.
	tmp, err := os.MkdirTemp("", "bliss-test-bin-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "bliss")
	out, err := exec.Command("go", "build", "-o", binaryPath, ".").CombinedOutput()
	if err != nil {
		panic("build failed: " + string(out))
	}

	os.Exit(m.Run())
}

// blissEnv returns an environment where HOME points to a temp directory,
// isolating the test store from the real ~/.bliss2.
func blissEnv(t *testing.T) (home string, env []string) {
	t.Helper()
	home = t.TempDir()
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "HOME=") {
			env = append(env, e)
		}
	}
	env = append(env, "HOME="+home)
	return home, env
}

// bliss runs the bliss binary with the given args in dir, using env.
func bliss(t *testing.T, dir string, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

// blissStdin runs the bliss binary with stdin piped from the given string.
func blissStdin(t *testing.T, dir string, env []string, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

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

func TestAdd_showsListInOutput(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	if _, err := bliss(t, dir, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	out, err := bliss(t, dir, env, "add", "Feed the penguins", "-l", "today")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out, "[today]") {
		t.Errorf("output %q should mention target list", out)
	}
	if !strings.Contains(out, "Feed the penguins") {
		t.Errorf("output %q should contain title", out)
	}
}

func TestAdd_noListInOutputWhenInbox(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	if _, err := bliss(t, dir, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	out, err := bliss(t, dir, env, "add", "Feed the penguins")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if strings.Contains(out, "[") {
		t.Errorf("output %q should not mention a list when added to inbox", out)
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
	if !strings.Contains(out, "[today]") {
		t.Errorf("output %q should mention target list", out)
	}
	if !strings.Contains(out, title) {
		t.Errorf("output %q does not contain title", out)
	}
}

func TestPersonalMode_addAndList(t *testing.T) {
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

func TestPersonalMode_addToList(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "add", "Urgent task", "-l", "today")
	if err != nil {
		t.Fatalf("add to list in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "[today]") {
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

func TestPersonalMode_done(t *testing.T) {
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

func TestPersonalMode_move(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Move me")
	bliss(t, dir, env, "list")

	out, err := bliss(t, dir, env, "move", "1", "-l", "today")
	if err != nil {
		t.Fatalf("move in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Moved to [today]") {
		t.Errorf("move output %q missing confirmation", out)
	}
}

func TestList_all(t *testing.T) {
	home, env := blissEnv(t)

	// Create two project dirs with contexts.
	proj1 := filepath.Join(home, "proj1")
	proj2 := filepath.Join(home, "proj2")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	bliss(t, proj1, env, "init", "--name", "Project One")
	bliss(t, proj2, env, "init", "--name", "Project Two")
	bliss(t, proj1, env, "add", "Todo in proj1")
	bliss(t, proj2, env, "add", "Todo in proj2")

	// Run --all from a neutral directory (home).
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

func TestInit_homeDirectoryGuard(t *testing.T) {
	home, env := blissEnv(t)

	out, err := bliss(t, home, env, "init")
	if err == nil {
		t.Fatalf("expected error running bliss init in home dir, got: %s", out)
	}
	if !strings.Contains(out, "home directory") {
		t.Errorf("error output %q does not mention home directory", out)
	}
}

func TestInit_storesPath(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	if _, err := bliss(t, proj, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	// status should show the path
	out, err := bliss(t, proj, env, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, proj) {
		t.Errorf("status output %q does not contain project path %q", out, proj)
	}
}
