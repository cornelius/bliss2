package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatus_personalMode(t *testing.T) {
	_, env := blissEnv(t)
	dir := t.TempDir()

	bliss(t, dir, env, "add", "Personal task one")
	bliss(t, dir, env, "add", "Personal task two")

	out, err := bliss(t, dir, env, "status")
	if err != nil {
		t.Fatalf("status in personal mode: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Personal:") {
		t.Errorf("output %q missing 'Personal:' label", out)
	}
	if !strings.Contains(out, "inbox") {
		t.Errorf("output %q missing inbox count", out)
	}
	if !strings.Contains(out, "no remote") {
		t.Errorf("output %q missing git sync line", out)
	}
}

func TestStatus_insideContext(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "myproject")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "My Project")
	bliss(t, proj, env, "add", "Context task")

	out, err := bliss(t, proj, env, "status")
	if err != nil {
		t.Fatalf("status inside context: %v\n%s", err, out)
	}
	if !strings.Contains(out, "My Project") {
		t.Errorf("output %q missing context name", out)
	}
	if !strings.Contains(out, "myproject") {
		t.Errorf("output %q missing context path", out)
	}
	if !strings.Contains(out, "inbox") {
		t.Errorf("output %q missing inbox", out)
	}
	if !strings.Contains(out, "no remote") {
		t.Errorf("output %q missing git sync line", out)
	}
}

func TestStatus_activeContextMarked(t *testing.T) {
	home, env := blissEnv(t)
	proj1 := filepath.Join(home, "active")
	proj2 := filepath.Join(home, "other")
	os.MkdirAll(proj1, 0755)
	os.MkdirAll(proj2, 0755)

	bliss(t, proj1, env, "init", "--name", "active")
	bliss(t, proj2, env, "init", "--name", "other")
	bliss(t, proj1, env, "add", "task one")
	bliss(t, proj2, env, "add", "task two")

	out, err := bliss(t, proj1, env, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, ">") {
		t.Errorf("output %q missing active context indicator '>'", out)
	}
}

func TestStatus_stalePathReported(t *testing.T) {
	home, env := blissEnv(t)
	proj := filepath.Join(home, "willmove")
	os.MkdirAll(proj, 0755)

	bliss(t, proj, env, "init", "--name", "Will Move")
	os.RemoveAll(proj)

	out, err := bliss(t, home, env, "status")
	if err != nil {
		t.Fatalf("status with stale: %v\n%s", err, out)
	}
	if !strings.Contains(out, "stale") {
		t.Errorf("output %q missing stale indicator", out)
	}
}
