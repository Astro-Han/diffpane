package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Astro-Han/diffpane/internal"
)

// ComputeDiff computes the current worktree diff against the session baseline.
func ComputeDiff(repoDir, baselineSHA string) ([]internal.FileDiff, error) {
	tracked, err := getTrackedDiff(repoDir, baselineSHA)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	if baselineSHA == EmptyTreeSHA {
		tracked, err = overlayFreshRepoWorktreeChanges(repoDir, tracked)
		if err != nil {
			return nil, fmt.Errorf("git diff worktree: %w", err)
		}
	}

	untracked, err := getUntrackedDiff(repoDir)
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	seen := make(map[string]bool)
	var result []internal.FileDiff
	for _, file := range tracked {
		seen[file.Path] = true
		result = append(result, file)
	}
	for _, file := range untracked {
		if !seen[file.Path] {
			result = append(result, file)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result, nil
}

// getTrackedDiff reads tracked-file changes from git diff output.
func getTrackedDiff(repoDir, baselineSHA string) ([]internal.FileDiff, error) {
	args := []string{"diff", "--no-renames", baselineSHA, "--"}
	if baselineSHA == EmptyTreeSHA {
		// Fresh repos have no reachable empty-tree object yet, so compare staged files
		// against an implicit empty tree instead of diffing by SHA.
		args = []string{"diff", "--cached", "--no-renames", "--root", "--"}
	}

	// #nosec G204 -- git and its arguments are fixed by the application flow.
	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return ParseDiff(string(out)), nil
}

// overlayFreshRepoWorktreeChanges replaces cached added-file diffs with the
// current worktree content for unborn-repo paths that were edited after staging.
func overlayFreshRepoWorktreeChanges(repoDir string, tracked []internal.FileDiff) ([]internal.FileDiff, error) {
	cmd := exec.Command("git", "diff", "--name-only", "-z", "--")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	indexByPath := make(map[string]int, len(tracked))
	result := append([]internal.FileDiff(nil), tracked...)
	for i, file := range result {
		indexByPath[file.Path] = i
	}

	for _, path := range strings.Split(string(out), "\x00") {
		if path == "" {
			continue
		}

		// #nosec G304 -- path comes from git diff output scoped to the repository.
		data, readErr := os.ReadFile(filepath.Join(repoDir, path))
		if readErr != nil {
			continue
		}

		diff := buildNewFileDiff(path, string(data))
		if idx, ok := indexByPath[path]; ok {
			result[idx] = diff
			continue
		}
		indexByPath[path] = len(result)
		result = append(result, diff)
	}

	return result, nil
}

// getUntrackedDiff expands untracked files and directories to synthetic added diffs.
// TODO(v2): reads entire file into memory; consider capping read size for large
// generated files and delegating binary detection to git.
func getUntrackedDiff(repoDir string) ([]internal.FileDiff, error) {
	cmd := exec.Command("git", "status", "--porcelain", "-z", "--untracked-files=all")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []internal.FileDiff
	for _, entry := range strings.Split(string(out), "\x00") {
		if entry == "" || !strings.HasPrefix(entry, "?? ") {
			continue
		}

		path := strings.TrimPrefix(entry, "?? ")
		// #nosec G304 -- path comes from git status output scoped to the repository.
		data, readErr := os.ReadFile(filepath.Join(repoDir, path))
		if readErr != nil {
			continue
		}
		files = append(files, buildNewFileDiff(path, string(data)))
	}

	return files, nil
}

// buildNewFileDiff synthesizes an added-file diff for untracked content.
func buildNewFileDiff(path, content string) internal.FileDiff {
	if strings.ContainsRune(content, '\x00') {
		return internal.FileDiff{
			Path:     path,
			Status:   internal.StatusBinary,
			IsBinary: true,
		}
	}

	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var diffLines []internal.DiffLine
	for _, line := range lines {
		diffLines = append(diffLines, internal.DiffLine{
			Type:    internal.LineAdd,
			Content: line,
		})
	}

	addCount := len(diffLines)
	return internal.FileDiff{
		Path:   path,
		Status: internal.StatusAdded,
		Hunks: []internal.DiffHunk{{
			Header:    fmt.Sprintf("@@ -0,0 +1,%d @@", addCount),
			StartLine: 1,
			Lines:     diffLines,
		}},
		AddCount: addCount,
	}
}
