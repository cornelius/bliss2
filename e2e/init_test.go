package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_outputShowsNameAndPath(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myservice")
	os.MkdirAll(proj, 0755)

	out, err := bliss(t, proj, env, "init", "--name", "myservice")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Initialized") {
		t.Errorf("output %q missing 'Initialized' label", out)
	}
	if !strings.Contains(out, "Context:") {
		t.Errorf("output %q missing 'Context:' label", out)
	}
	if !strings.Contains(out, "Path:") {
		t.Errorf("output %q missing 'Path:' label", out)
	}
	if !strings.Contains(out, "myservice") {
		t.Errorf("output %q missing context name", out)
	}
	// UUID must not appear in output.
	if strings.Count(out, "-") >= 4 {
		t.Errorf("output %q appears to contain UUID (4+ hyphens)", out)
	}
}

func TestInit_alreadyInitialized(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	if _, err := bliss(t, proj, env, "init"); err != nil {
		t.Fatalf("first init: %v", err)
	}

	out, err := bliss(t, proj, env, "init")
	if err == nil {
		t.Fatalf("second init should have failed, got: %s", out)
	}
	if !strings.Contains(out, "already") {
		t.Errorf("error output %q should mention 'already'", out)
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

func TestInit_pathStoredAndShownInStatus(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	if _, err := bliss(t, proj, env, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := bliss(t, proj, env, "add", "a task"); err != nil {
		t.Fatalf("add: %v", err)
	}

	out, err := bliss(t, proj, env, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, "myproject") {
		t.Errorf("status output %q does not contain project path", out)
	}
}

func TestInit_contextsCommandRemoved(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	out, err := bliss(t, dir, env, "contexts")
	if err == nil {
		t.Errorf("expected error for removed 'contexts' command, got: %s", out)
	}
}

func TestInit_defaultsToDirectoryName(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "my-service")
	os.MkdirAll(proj, 0755)

	out, err := bliss(t, proj, env, "init")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	// Slug derived from directory name "my-service"
	if !strings.Contains(out, "my-service") {
		t.Errorf("output %q missing slug derived from directory name", out)
	}

	// .bliss-context must contain the slug, not a UUID
	data, err := os.ReadFile(filepath.Join(proj, ".bliss-context"))
	if err != nil {
		t.Fatalf("reading .bliss-context: %v", err)
	}
	slug := strings.TrimSpace(string(data))
	if slug != "my-service" {
		t.Errorf(".bliss-context = %q, want %q", slug, "my-service")
	}
}

func TestInit_slugifiesName(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "proj")
	os.MkdirAll(proj, 0755)

	out, err := bliss(t, proj, env, "init", "--name", "My Project")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if !strings.Contains(out, "my-project") {
		t.Errorf("output %q missing slugified name 'my-project'", out)
	}

	data, err := os.ReadFile(filepath.Join(proj, ".bliss-context"))
	if err != nil {
		t.Fatalf("reading .bliss-context: %v", err)
	}
	slug := strings.TrimSpace(string(data))
	if slug != "my-project" {
		t.Errorf(".bliss-context = %q, want %q", slug, "my-project")
	}
}

func TestInit_linksExistingContext(t *testing.T) {
	home, env := blissEnv(t)

	// First machine: initialize context
	proj1 := filepath.Join(home, "proj1")
	os.MkdirAll(proj1, 0755)
	if _, err := bliss(t, proj1, env, "init", "--name", "shared"); err != nil {
		t.Fatalf("first init: %v", err)
	}

	// Second directory with same slug: should offer to link
	proj2 := filepath.Join(home, "proj2")
	os.MkdirAll(proj2, 0755)
	out, err := blissStdin(t, proj2, env, "y\n", "init", "--name", "shared")
	if err != nil {
		t.Fatalf("second init (link): %v\n%s", err, out)
	}
	if !strings.Contains(out, "Linked") {
		t.Errorf("output %q missing 'Linked' confirmation", out)
	}

	// Both directories must point to the same context slug
	data1, _ := os.ReadFile(filepath.Join(proj1, ".bliss-context"))
	data2, _ := os.ReadFile(filepath.Join(proj2, ".bliss-context"))
	if strings.TrimSpace(string(data1)) != strings.TrimSpace(string(data2)) {
		t.Errorf(".bliss-context mismatch: proj1=%q proj2=%q", strings.TrimSpace(string(data1)), strings.TrimSpace(string(data2)))
	}
}
