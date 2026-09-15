package herdr

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type recRunner struct{ calls [][]string }

func (r *recRunner) Run(_ context.Context, bin string, args ...string) ([]byte, []byte, error) {
	r.calls = append(r.calls, append([]string{bin}, args...))
	return []byte(`{"id":"ws1","label":"api","cwd":"/tmp/api"}`), nil, nil
}
func TestCLIClientConstructsWorkspaceCreate(t *testing.T) {
	rr := &recRunner{}
	c := &CLIClient{Bin: "/bin/herdr", Runner: rr}
	_, err := c.WorkspaceCreate(context.Background(), WorkspaceCreateRequest{CWD: "/tmp/api", Label: "api", Focus: true})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"/bin/herdr", "workspace", "create", "--cwd", "/tmp/api", "--label", "api"},
		{"/bin/herdr", "workspace", "focus", "ws1"},
	}
	if !reflect.DeepEqual(rr.calls, want) {
		t.Fatalf("got %#v want %#v", rr.calls, want)
	}
}

func TestCLIClientConstructsWorkspaceCreateNoFocus(t *testing.T) {
	rr := &recRunner{}
	c := &CLIClient{Bin: "/bin/herdr", Runner: rr}
	_, err := c.WorkspaceCreate(context.Background(), WorkspaceCreateRequest{CWD: "/tmp/api", Label: "api"})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"/bin/herdr", "workspace", "create", "--cwd", "/tmp/api", "--label", "api", "--no-focus"}}
	if !reflect.DeepEqual(rr.calls, want) {
		t.Fatalf("got %#v want %#v", rr.calls, want)
	}
}

func TestCLIClientConstructsWorkspaceClose(t *testing.T) {
	rr := &recRunner{}
	c := &CLIClient{Bin: "/bin/herdr", Runner: rr}
	if err := c.WorkspaceClose(context.Background(), "ws1"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"/bin/herdr", "workspace", "close", "ws1"}}
	if !reflect.DeepEqual(rr.calls, want) {
		t.Fatalf("got %#v want %#v", rr.calls, want)
	}
}

func TestCLIClientDecodesWorkspaceListEnvelope(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte(`{"result":{"workspaces":[{"workspace_id":"w1","label":"api","agent_status":"working"}]}}`)}}
	got, err := c.WorkspaceList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "w1" || got[0].Label != "api" || got[0].AgentStatus != "working" {
		t.Fatalf("workspaces=%#v", got)
	}
}

func TestCLIClientDecodesWorkspaceListWorktreeMetadata(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte(`{"result":{"workspaces":[{"workspace_id":"w-root","label":"project","worktree":{"checkout_path":"/repos/project","is_linked_worktree":false,"repo_key":"/repos/project/.git","repo_name":"project","repo_root":"/repos/project"}},{"workspace_id":"w-child","label":"feature","worktree":{"checkout_path":"/worktrees/feature","is_linked_worktree":true,"repo_key":"/repos/project/.git","repo_name":"project","repo_root":"/repos/project"}}]}}`)}}
	got, err := c.WorkspaceList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("workspaces=%#v", got)
	}
	root, child := got[0].Worktree, got[1].Worktree
	if root == nil || root.IsLinkedWorktree || root.CheckoutPath != "/repos/project" || root.RepoKey != "/repos/project/.git" || root.RepoName != "project" || root.RepoRoot != "/repos/project" {
		t.Fatalf("root worktree=%#v", root)
	}
	if child == nil || !child.IsLinkedWorktree || child.CheckoutPath != "/worktrees/feature" || child.RepoKey != root.RepoKey {
		t.Fatalf("child worktree=%#v", child)
	}
}

func TestCLIClientDecodesWorkspaceListArrayWithoutWorktreeMetadata(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte(`[{"id":"w1","label":"api","agent_status":"blocked"}]`)}}
	got, err := c.WorkspaceList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "w1" || got[0].Label != "api" || got[0].AgentStatus != "blocked" || got[0].Worktree != nil {
		t.Fatalf("workspaces=%#v", got)
	}
}

func TestCLIClientDecodesWorkspaceCreateEnvelope(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte(`{"result":{"root_pane":{"cwd":"/tmp/api","pane_id":"p1"},"workspace":{"workspace_id":"w1","label":"api"}}}`)}}
	got, err := c.WorkspaceCreate(context.Background(), WorkspaceCreateRequest{CWD: "/tmp/api", Label: "api", Focus: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "w1" || got.Label != "api" || got.CWD != "/tmp/api" {
		t.Fatalf("workspace=%#v", got)
	}
}

func TestCLIClientDecodesTabCreateEnvelope(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte(`{"result":{"root_pane":{"cwd":"/tmp/api","pane_id":"p1"},"tab":{"tab_id":"w1:t2","workspace_id":"w1","label":"api"}}}`)}}
	got, err := c.TabCreate(context.Background(), TabCreateRequest{WorkspaceID: "w1", CWD: "/tmp/api", Label: "api", Focus: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "w1:t2" || got.WorkspaceID != "w1" || got.CWD != "/tmp/api" || got.PaneID != "p1" {
		t.Fatalf("tab=%#v", got)
	}
}

func TestCLIClientDecodesTabListArray(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte(`[{"id":"w1:t1","workspace_id":"w1","label":"api"}]`)}}
	got, err := c.TabList(context.Background(), "w1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "w1:t1" || got[0].WorkspaceID != "w1" || got[0].Label != "api" {
		t.Fatalf("tabs=%#v", got)
	}
}

func TestCLIClientDecodesPaneListEnvelope(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte(`{"result":{"panes":[{"pane_id":"p1","workspace_id":"w1","tab_id":"w1:t1","cwd":"/tmp/api","foreground_cwd":"/tmp/api/sub","focused":true}]}}`)}}
	got, err := c.PaneList(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "p1" || got[0].WorkspaceID != "w1" || got[0].ForegroundCWD != "/tmp/api/sub" || !got[0].Focused {
		t.Fatalf("panes=%#v", got)
	}
}

func TestCLIClientDecodesPaneCurrentEnvelope(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte(`{"result":{"pane":{"pane_id":"p1","workspace_id":"w1","tab_id":"w1:t1","cwd":"/tmp/api"}}}`)}}
	got, err := c.PaneCurrent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "p1" || got.WorkspaceID != "w1" || got.TabID != "w1:t1" || got.CWD != "/tmp/api" {
		t.Fatalf("pane=%#v", got)
	}
}

func TestCLIClientPaneFocusedOmitsCallerPane(t *testing.T) {
	d := t.TempDir()
	bin := filepath.Join(d, "herdr")
	script := `#!/bin/sh
if [ -n "$HERDR_PANE_ID" ]; then
  printf '{"workspace_id":"caller"}\n'
else
  printf '{"workspace_id":"focused"}\n'
fi
`
	//nolint:gosec // test creates a local executable fixture.
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_PANE_ID", "stale-pane")
	c := &CLIClient{Bin: bin, Runner: ExecRunner{}}

	caller, err := c.PaneCurrent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	focused, err := c.PaneFocused(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if caller.WorkspaceID != "caller" || focused.WorkspaceID != "focused" {
		t.Fatalf("caller=%q focused=%q", caller.WorkspaceID, focused.WorkspaceID)
	}
}
func TestFakeClientRecordsPaneRun(t *testing.T) {
	f := &FakeClient{}
	_ = f.PaneRun(context.Background(), "p1", "npm test")
	if f.PaneRuns[0] != "p1:npm test" {
		t.Fatal(f.PaneRuns)
	}
}

type fixedRunner struct {
	stdout []byte
	stderr []byte
	err    error
}

func (r fixedRunner) Run(context.Context, string, ...string) ([]byte, []byte, error) {
	return r.stdout, r.stderr, r.err
}

func TestCLIClientReturnsDecodeErrors(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stdout: []byte("not json")}}
	_, err := c.WorkspaceList(context.Background())
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode herdr workspace list JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCLIClientIncludesStderrOnCommandFailure(t *testing.T) {
	c := &CLIClient{Bin: "/bin/herdr", Runner: fixedRunner{stderr: []byte("boom\n"), err: errors.New("exit status 1")}}
	_, err := c.WorkspaceList(context.Background())
	if err == nil {
		t.Fatal("expected command error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func stubWorktree(root, parent string) WorktreeResolver {
	return func(_ context.Context, cwd string) (string, string, bool) {
		if cwd != root {
			return "", "", false
		}
		return root, parent, true
	}
}

func noWorktree(_ context.Context, _ string) (string, string, bool) { return "", "", false }

func TestCLIClientOpensWorktreeRootThroughWorktreeOpen(t *testing.T) {
	rr := &recRunner{}
	c := &CLIClient{Bin: "/bin/herdr", Runner: rr, Worktree: stubWorktree("/wt/feature", "/repos/project")}
	_, err := c.WorkspaceCreate(context.Background(), WorkspaceCreateRequest{CWD: "/wt/feature", Label: "feature", Focus: true})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"/bin/herdr", "worktree", "open", "--cwd", "/repos/project", "--path", "/wt/feature", "--label", "feature"},
		{"/bin/herdr", "workspace", "focus", "ws1"},
	}
	if !reflect.DeepEqual(rr.calls, want) {
		t.Fatalf("got %#v want %#v", rr.calls, want)
	}
}

func TestCLIClientWorktreeOpenNoFocus(t *testing.T) {
	rr := &recRunner{}
	c := &CLIClient{Bin: "/bin/herdr", Runner: rr, Worktree: stubWorktree("/wt/feature", "/repos/project")}
	if _, err := c.WorkspaceCreate(context.Background(), WorkspaceCreateRequest{CWD: "/wt/feature", Label: "feature"}); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"/bin/herdr", "worktree", "open", "--cwd", "/repos/project", "--path", "/wt/feature", "--label", "feature", "--no-focus"}}
	if !reflect.DeepEqual(rr.calls, want) {
		t.Fatalf("got %#v want %#v", rr.calls, want)
	}
}

type failThenRecordRunner struct {
	calls [][]string
}

func (r *failThenRecordRunner) Run(_ context.Context, bin string, args ...string) ([]byte, []byte, error) {
	r.calls = append(r.calls, append([]string{bin}, args...))
	if len(args) > 0 && args[0] == "worktree" {
		return []byte(`{"error":{"code":"linked_worktree_source"}}`), []byte("refused"), errors.New("exit status 1")
	}
	return []byte(`{"result":{"workspace":{"workspace_id":"w9","label":"feature"},"root_pane":{"cwd":"/wt/feature"}}}`), nil, nil
}

func TestCLIClientFallsBackToWorkspaceCreateWhenWorktreeOpenFails(t *testing.T) {
	rr := &failThenRecordRunner{}
	c := &CLIClient{Bin: "/bin/herdr", Runner: rr, Worktree: stubWorktree("/wt/feature", "/repos/project")}
	w, err := c.WorkspaceCreate(context.Background(), WorkspaceCreateRequest{CWD: "/wt/feature", Label: "feature"})
	if err != nil {
		t.Fatal(err)
	}
	if w.ID != "w9" || w.CWD != "/wt/feature" {
		t.Fatalf("workspace=%#v", w)
	}
	want := [][]string{
		{"/bin/herdr", "worktree", "open", "--cwd", "/repos/project", "--path", "/wt/feature", "--label", "feature", "--no-focus"},
		{"/bin/herdr", "workspace", "create", "--cwd", "/wt/feature", "--label", "feature", "--no-focus"},
	}
	if !reflect.DeepEqual(rr.calls, want) {
		t.Fatalf("got %#v want %#v", rr.calls, want)
	}
}

func TestCLIClientPlainDirectoryKeepsWorkspaceCreate(t *testing.T) {
	rr := &recRunner{}
	c := &CLIClient{Bin: "/bin/herdr", Runner: rr, Worktree: noWorktree}
	if _, err := c.WorkspaceCreate(context.Background(), WorkspaceCreateRequest{CWD: "/tmp/notes", Label: "notes"}); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"/bin/herdr", "workspace", "create", "--cwd", "/tmp/notes", "--label", "notes", "--no-focus"}}
	if !reflect.DeepEqual(rr.calls, want) {
		t.Fatalf("got %#v want %#v", rr.calls, want)
	}
}

func TestGitWorktreeRootsRecognizesCheckoutRootOnly(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"commit", "-q", "--allow-empty", "-m", "init"}} {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	sub := filepath.Join(repo, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	root, parent, ok := GitWorktreeRoots(context.Background(), repo)
	if !ok || !sameDir(root, repo) || !sameDir(parent, repo) {
		t.Fatalf("primary checkout: root=%q parent=%q ok=%v", root, parent, ok)
	}
	if _, _, ok := GitWorktreeRoots(context.Background(), sub); ok {
		t.Fatal("subdirectory of a checkout must not resolve as a worktree root")
	}
	linked := filepath.Join(t.TempDir(), "feature")
	if out, err := exec.Command("git", "-C", repo, "worktree", "add", "-q", linked, "-b", "feature").CombinedOutput(); err != nil {
		t.Fatalf("worktree add: %v: %s", err, out)
	}
	root, parent, ok = GitWorktreeRoots(context.Background(), linked)
	if !ok || !sameDir(root, linked) || !sameDir(parent, repo) {
		t.Fatalf("linked worktree: root=%q parent=%q ok=%v", root, parent, ok)
	}
	if _, _, ok := GitWorktreeRoots(context.Background(), t.TempDir()); ok {
		t.Fatal("a directory outside any repository must not resolve")
	}
}
