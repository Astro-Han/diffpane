package watcher

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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
