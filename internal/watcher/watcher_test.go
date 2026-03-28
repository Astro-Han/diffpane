package watcher

import (
	"path/filepath"
	"testing"
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
