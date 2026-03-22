package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "bliss-test-bin-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "bliss")
	out, err := exec.Command("go", "build", "-o", binaryPath, "bliss/cmd/bliss").CombinedOutput()
	if err != nil {
		panic("build failed: " + string(out))
	}

	os.Exit(m.Run())
}

// blissEnv returns a HOME-isolated environment and the temp home directory.
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

// setupRemote creates a bare git repo and configures it as origin for the bliss
// store. Requires at least one bliss commit to exist first. Returns the bare path.
func setupRemote(t *testing.T, home string) string {
	t.Helper()
	bare := filepath.Join(home, "remote.git")
	storeDir := filepath.Join(home, ".bliss2")

	if out, err := exec.Command("git", "init", "--bare", bare).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", storeDir, "remote", "add", "origin", bare).CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, out)
	}

	// Resolve the current branch name — differs by git version and config.
	out, err := exec.Command("git", "-C", storeDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("get branch: %v", err)
	}
	branch := strings.TrimSpace(string(out))

	if out, err := exec.Command("git", "-C", storeDir, "push", "-u", "origin", branch).CombinedOutput(); err != nil {
		t.Fatalf("initial push: %v\n%s", err, out)
	}

	// Align the bare repo's HEAD to the branch we just pushed.
	// go-git uses "master" but the system git may default to "main", leaving
	// the bare repo's HEAD pointing to an unborn branch — which breaks git clone.
	if out, err := exec.Command("git", "-C", bare, "symbolic-ref", "HEAD", "refs/heads/"+branch).CombinedOutput(); err != nil {
		t.Fatalf("set bare HEAD: %v\n%s", err, out)
	}

	return bare
}

// advanceRemote clones the bare repo, adds an empty commit, and pushes it back,
// simulating another machine having pushed new commits to the remote.
func advanceRemote(t *testing.T, bare string) {
	t.Helper()
	clone := t.TempDir()

	if out, err := exec.Command("git", "clone", bare, clone).CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	gitEnv := append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.local",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.local",
	)
	cmd := exec.Command("git", "-C", clone, "commit", "--allow-empty", "-m", "remote commit")
	cmd.Env = gitEnv
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", clone, "push").CombinedOutput(); err != nil {
		t.Fatalf("git push: %v\n%s", err, out)
	}
}
