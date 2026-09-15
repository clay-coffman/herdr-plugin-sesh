package namer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNameFromGitRemote(t *testing.T) {
	d := t.TempDir()
	run(t, d, "git", "init")
	run(t, d, "git", "remote", "add", "origin", "git@github.com:fullerzz/herdr-plugin-sesh.git")
	got := Namer{}.Name(context.Background(), d, 1)
	if got != "herdr-plugin-sesh" {
		t.Fatalf("got %q", got)
	}
}
func TestNameFromDirectoryLength(t *testing.T) {
	got := Namer{}.Name(context.Background(), "/tmp/parent/child", 2)
	if got != "parent/child" {
		t.Fatalf("got %q", got)
	}
}
func TestNameFromRepoWithoutRemote(t *testing.T) {
	d := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(d, 0700); err != nil {
		t.Fatal(err)
	}
	run(t, d, "git", "init")
	got := Namer{}.Name(context.Background(), d, 1)
	if got != "repo" {
		t.Fatalf("got %q", got)
	}
}
func TestNameLinkedWorktreeUsesCheckoutDirectory(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0700); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "git", "init", "-q")
	run(t, repo, "git", "remote", "add", "origin", "git@github.com:fullerzz/herdr-plugin-sesh.git")
	run(t, repo, "git", "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init")
	linked := filepath.Join(t.TempDir(), "feat-picker-labels")
	run(t, repo, "git", "worktree", "add", "-q", linked, "-b", "feat/picker-labels")
	if got := (Namer{}).Name(context.Background(), linked, 1); got != "feat-picker-labels" {
		t.Fatalf("linked worktree: got %q", got)
	}
	if got := (Namer{}).Name(context.Background(), repo, 1); got != "herdr-plugin-sesh" {
		t.Fatalf("primary checkout: got %q", got)
	}
}
func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	//nolint:gosec // tests execute fixed git commands.
	c := exec.Command(args[0], args[1:]...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v %s", args, err, out)
	}
}
