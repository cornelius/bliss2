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
