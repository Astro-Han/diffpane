package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// samePath reports whether two paths refer to the same filesystem object.
func samePath(t *testing.T, got, want string) {
	t.Helper()

	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat got %q: %v", got, err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat want %q: %v", want, err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("paths are different, got %q want %q", got, want)
	}
}

// initGitRepo creates a git repo for tests and returns its root.
func initGitRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "config", "user.email", "test@example.com")
	return root
}

// runGit runs a git command in dir and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}

	return string(out)
}

// commitFile writes a file and creates a commit so HEAD exists.
func commitFile(t *testing.T, root, name, content string) string {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	runGit(t, root, "add", name)
	runGit(t, root, "commit", "-m", "initial")

	sha := runGit(t, root, "rev-parse", "HEAD")
	return strings.TrimSpace(sha)
}

// TestFindWorktreeRootInGitRepo finds the repo root from a nested path.
func TestFindWorktreeRootInGitRepo(t *testing.T) {
	root := initGitRepo(t)
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, err := FindWorktreeRoot(nested)
	if err != nil {
		t.Fatalf("FindWorktreeRoot returned error: %v", err)
	}
	samePath(t, got, root)
}

// TestFindWorktreeRootNotGitRepo reports an error outside git worktrees.
func TestFindWorktreeRootNotGitRepo(t *testing.T) {
	root := t.TempDir()

	_, err := FindWorktreeRoot(root)
	if err == nil {
		t.Fatal("FindWorktreeRoot returned nil error, want failure")
	}
}

// TestGetHeadSHAWithCommit returns the commit SHA for a normal repo.
func TestGetHeadSHAWithCommit(t *testing.T) {
	root := initGitRepo(t)
	want := commitFile(t, root, "README.md", "hello\n")

	got, err := GetHeadSHA(root)
	if err != nil {
		t.Fatalf("GetHeadSHA returned error: %v", err)
	}
	if got != want {
		t.Fatalf("GetHeadSHA = %q, want %q", got, want)
	}
}

// TestGetHeadSHAEmptyRepo returns the empty tree SHA for an empty repo.
func TestGetHeadSHAEmptyRepo(t *testing.T) {
	root := initGitRepo(t)

	got, err := GetHeadSHA(root)
	if err != nil {
		t.Fatalf("GetHeadSHA returned error: %v", err)
	}
	if got != EmptyTreeSHA {
		t.Fatalf("GetHeadSHA = %q, want %q", got, EmptyTreeSHA)
	}
}

// TestGetHeadSHAHeadRefError returns an error when HEAD points to a missing ref.
func TestGetHeadSHAHeadRefError(t *testing.T) {
	root := initGitRepo(t)
	commitFile(t, root, "README.md", "hello\n")

	headPath := filepath.Join(root, ".git", "HEAD")
	if err := os.WriteFile(headPath, []byte("ref: refs/heads/missing\n"), 0o644); err != nil {
		t.Fatalf("write HEAD: %v", err)
	}

	_, err := GetHeadSHA(root)
	if err == nil {
		t.Fatal("GetHeadSHA returned nil error, want failure for broken HEAD ref")
	}
}

// TestResolveGitDirNormalRepo resolves .git inside a standard repo.
func TestResolveGitDirNormalRepo(t *testing.T) {
	root := initGitRepo(t)

	got := ResolveGitDir(root)
	want := filepath.Join(root, ".git")
	samePath(t, got, want)
}

// TestResolveGitDirLinkedWorktree resolves the gitdir from a linked worktree.
func TestResolveGitDirLinkedWorktree(t *testing.T) {
	repo := initGitRepo(t)
	runGit(t, repo, "commit", "--allow-empty", "-m", "root")

	worktrees := filepath.Join(t.TempDir(), "worktrees")
	if err := os.MkdirAll(worktrees, 0o755); err != nil {
		t.Fatalf("mkdir worktrees: %v", err)
	}
	worktreeDir := filepath.Join(worktrees, "linked")
	runGit(t, repo, "worktree", "add", worktreeDir)

	got := ResolveGitDir(worktreeDir)
	want := filepath.Join(repo, ".git", "worktrees", "linked")
	samePath(t, got, want)
}
