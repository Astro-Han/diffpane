package watcher

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	gitpkg "github.com/Astro-Han/diffpane/internal/git"
	"github.com/fsnotify/fsnotify"
)

// TestIsGitInternalPath verifies only the real git internals are filtered.
func TestIsGitInternalPath(t *testing.T) {
	repo := "/Users/test/myrepo"
	gitDir := filepath.Join(repo, ".git")

	tests := []struct {
		path string
		want bool
	}{
		{filepath.Join(repo, ".git", "HEAD"), true},
		{filepath.Join(repo, ".git", "refs", "heads", "main"), true},
		{filepath.Join(repo, ".git"), true},
		{filepath.Join(repo, ".github", "workflows", "ci.yml"), false},
		{filepath.Join(repo, ".gitignore"), false},
		{filepath.Join(repo, ".gitmodules"), false},
		{filepath.Join(repo, "src", "main.go"), false},
	}

	for _, tc := range tests {
		got := isGitInternalPath(tc.path, gitDir)
		if got != tc.want {
			t.Fatalf("isGitInternalPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestIsGitInternalPathEmptyGitDir verifies an empty gitDir never matches absolute paths.
func TestIsGitInternalPathEmptyGitDir(t *testing.T) {
	if isGitInternalPath("/tmp/repo/file.go", "") {
		t.Fatal("empty gitDir should not match arbitrary paths")
	}
}

// TestIsHeadOrRefPath verifies only HEAD and refs trigger baseline resets.
func TestIsHeadOrRefPath(t *testing.T) {
	gitDir := "/Users/test/myrepo/.git"
	tests := []struct {
		path string
		want bool
	}{
		{filepath.Join(gitDir, "HEAD"), true},
		{filepath.Join(gitDir, "refs", "heads", "main"), true},
		{filepath.Join(gitDir, "objects", "pack", "abc"), false},
		{filepath.Join(gitDir, "config"), false},
	}

	for _, tc := range tests {
		got := isHeadOrRefPath(tc.path, gitDir)
		if got != tc.want {
			t.Fatalf("isHeadOrRefPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestReportError writes watcher async errors to the configured error writer.
func TestReportError(t *testing.T) {
	var stderr bytes.Buffer
	fw := &FileWatcher{errWriter: &stderr}

	fw.reportError("watcher error")

	if stderr.String() != "watcher error\n" {
		t.Fatalf("stderr = %q, want watcher error line", stderr.String())
	}
}

// TestIsMetadataOnlyEvent verifies pure chmod notifications do not trigger a
// diff refresh, while real content-changing events still pass through.
func TestIsMetadataOnlyEvent(t *testing.T) {
	tests := []struct {
		name  string
		event fsnotify.Event
		want  bool
	}{
		{
			name:  "pure chmod",
			event: fsnotify.Event{Op: fsnotify.Chmod},
			want:  true,
		},
		{
			name:  "write plus chmod",
			event: fsnotify.Event{Op: fsnotify.Write | fsnotify.Chmod},
			want:  false,
		},
		{
			name:  "write only",
			event: fsnotify.Event{Op: fsnotify.Write},
			want:  false,
		},
	}

	for _, tc := range tests {
		got := isMetadataOnlyEvent(tc.event)
		if got != tc.want {
			t.Fatalf("%s: isMetadataOnlyEvent(%v) = %v, want %v", tc.name, tc.event.Op, got, tc.want)
		}
	}
}

// TestAddDirRecursiveSkipsIgnoredDirectories verifies startup watching does not
// recurse into directories git would ignore.
func TestAddDirRecursiveSkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("node_modules/\n"), 0o600); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o750); err != nil {
		t.Fatalf("mkdir ignored dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o750); err != nil {
		t.Fatalf("mkdir watched dir: %v", err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = fsw.Close() }()

	fw := &FileWatcher{fsw: fsw, repoDir: root}
	fw.addDirRecursive(root)

	watched := fsw.WatchList()
	if containsPath(watched, filepath.Join(root, "node_modules")) {
		t.Fatalf("watch list should skip ignored directory, got %#v", watched)
	}
	if !containsPath(watched, filepath.Join(root, "src")) {
		t.Fatalf("watch list should include normal directory, got %#v", watched)
	}
}

// TestInfoExcludeChangesTriggerRefresh verifies watcher refreshes diffs when
// repo-local exclude rules change.
func TestInfoExcludeChangesTriggerRefresh(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")

	changes, fw := startTestWatcher(t, root, filepath.Join(root, ".git"), filepath.Join(root, ".git"))
	defer fw.Stop()

	time.Sleep(100 * time.Millisecond)

	excludePath := filepath.Join(root, ".git", "info", "exclude")
	if err := os.WriteFile(excludePath, []byte("ignored.log\n"), 0o600); err != nil {
		t.Fatalf("write exclude: %v", err)
	}

	paths := waitForPaths(t, changes, ".git/info/exclude refresh")
	if !containsPath(paths, filepath.Join(".git", "info", "exclude")) {
		t.Fatalf("changed paths = %#v, want .git/info/exclude", paths)
	}
}

// TestInfoExcludeAtomicReplaceTriggersRefresh verifies editor-style atomic
// replacement still refreshes diffs when exclude rules change.
func TestInfoExcludeAtomicReplaceTriggersRefresh(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")

	changes, fw := startTestWatcher(t, root, filepath.Join(root, ".git"), filepath.Join(root, ".git"))
	defer fw.Stop()

	time.Sleep(100 * time.Millisecond)

	infoDir := filepath.Join(root, ".git", "info")
	tempPath := filepath.Join(infoDir, "exclude.tmp")
	excludePath := filepath.Join(infoDir, "exclude")
	if err := os.WriteFile(tempPath, []byte("ignored.log\n"), 0o600); err != nil {
		t.Fatalf("write temp exclude: %v", err)
	}
	if err := os.Rename(tempPath, excludePath); err != nil {
		t.Fatalf("rename temp exclude: %v", err)
	}

	waitForPaths(t, changes, "atomic .git/info/exclude refresh")
}

// TestLinkedWorktreeCommonExcludeChangesTriggerRefresh verifies linked
// worktrees refresh when the shared repo-local exclude file changes.
func TestLinkedWorktreeCommonExcludeChangesTriggerRefresh(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "commit", "--allow-empty", "-m", "root")

	worktrees := filepath.Join(t.TempDir(), "worktrees")
	if err := os.MkdirAll(worktrees, 0o750); err != nil {
		t.Fatalf("mkdir worktrees: %v", err)
	}
	worktreeDir := filepath.Join(worktrees, "linked")
	runGit(t, repo, "worktree", "add", worktreeDir)

	gitDir := gitpkg.ResolveGitDir(worktreeDir)
	commonGitDir := gitpkg.GetGitCommonDir(worktreeDir)
	changes, fw := startTestWatcher(t, worktreeDir, gitDir, commonGitDir)
	defer fw.Stop()

	time.Sleep(100 * time.Millisecond)

	excludePath := filepath.Join(commonGitDir, "info", "exclude")
	if err := os.WriteFile(excludePath, []byte("ignored.log\n"), 0o600); err != nil {
		t.Fatalf("write common exclude: %v", err)
	}

	paths := waitForPaths(t, changes, "linked worktree .git/info/exclude refresh")
	want, err := filepath.Rel(worktreeDir, excludePath)
	if err != nil {
		t.Fatalf("filepath.Rel: %v", err)
	}
	if !containsPath(paths, want) {
		t.Fatalf("changed paths = %#v, want %q", paths, want)
	}
}

func startTestWatcher(t *testing.T, repoDir, gitDir, commonGitDir string) (chan []string, *FileWatcher) {
	t.Helper()

	changes := make(chan []string, 1)
	fw, err := New(
		repoDir,
		gitDir,
		commonGitDir,
		func(paths []string) {
			changes <- append([]string(nil), paths...)
		},
		func() {},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return changes, fw
}

func waitForPaths(t *testing.T, changes <-chan []string, label string) []string {
	t.Helper()

	select {
	case paths := <-changes:
		return paths
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
		return nil
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	// #nosec G204 -- test helper with fixed git command and controlled arguments.
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func containsPath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want {
			return true
		}
	}
	return false
}
