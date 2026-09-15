package namer

import (
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Runner interface {
	Run(context.Context, string, ...string) (string, error)
}
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, bin string, args ...string) (string, error) {
	//nolint:gosec // command is the fixed git lookup used to name local sessions.
	b, err := exec.CommandContext(ctx, bin, args...).Output()
	return strings.TrimSpace(string(b)), err
}

type Namer struct{ Runner Runner }

func (n Namer) Name(ctx context.Context, path string, dirLength int) string {
	if dirLength < 1 {
		dirLength = 1
	}
	r := n.Runner
	if r == nil {
		r = ExecRunner{}
	}
	if root, err := r.Run(ctx, "git", "-C", path, "rev-parse", "--show-toplevel"); err == nil && root != "" {
		if isLinkedWorktree(ctx, r, path, root) {
			// A linked worktree is named after its checkout directory, which is
			// what Herdr itself labels a worktree workspace. Naming it after the
			// remote would give every worktree of a repository the same label.
			return filepath.Base(root)
		}
		if remote, err := r.Run(ctx, "git", "-C", path, "config", "--get", "remote.origin.url"); err == nil && remote != "" {
			return repoName(remote)
		}
		return filepath.Base(root)
	}
	return lastComponents(path, dirLength)
}

// isLinkedWorktree reports whether root is a `git worktree add` checkout
// rather than the repository's primary checkout. The common .git directory
// lives inside the primary checkout, so a root that is not its parent is a
// linked worktree.
func isLinkedWorktree(ctx context.Context, r Runner, path, root string) bool {
	common, err := r.Run(ctx, "git", "-C", path, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil || common == "" || filepath.Base(common) != ".git" {
		return false
	}
	return !sameDir(filepath.Dir(common), root)
}

func sameDir(a, b string) bool {
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return ra == rb
}
func repoName(remote string) string {
	remote = strings.TrimSuffix(remote, ".git")
	if strings.Contains(remote, ":") && !strings.Contains(remote, "://") {
		parts := strings.Split(remote, ":")
		remote = parts[len(parts)-1]
	}
	remote = strings.TrimRight(remote, "/")
	parts := strings.Split(remote, "/")
	name := parts[len(parts)-1]
	re := regexp.MustCompile(`[^A-Za-z0-9._/-]+`)
	return re.ReplaceAllString(name, "-")
}
func lastComponents(path string, n int) string {
	p := filepath.Clean(path)
	parts := strings.Split(p, string(filepath.Separator))
	out := []string{}
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return filepath.Base(p)
	}
	if n > len(out) {
		n = len(out)
	}
	return strings.Join(out[len(out)-n:], "/")
}
