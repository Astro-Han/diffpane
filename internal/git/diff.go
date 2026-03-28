package git

import (
	"fmt"
	"io/fs"
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

// getUntrackedDiff expands untracked files and directories to synthetic added diffs.
func getUntrackedDiff(repoDir string) ([]internal.FileDiff, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []internal.FileDiff
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" || !strings.HasPrefix(line, "?? ") {
			continue
		}

		path := strings.TrimPrefix(line, "?? ")
		if strings.HasSuffix(path, "/") {
			dirPath := filepath.Join(repoDir, strings.TrimSuffix(path, "/"))
			walkErr := filepath.WalkDir(dirPath, func(current string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}

				rel, relErr := filepath.Rel(repoDir, current)
				if relErr != nil {
					return nil
				}
				// #nosec G304,G122 -- current comes from walking inside the repository root.
				data, readErr := os.ReadFile(current)
				if readErr != nil {
					return nil
				}
				files = append(files, buildNewFileDiff(rel, string(data)))
				return nil
			})
			if walkErr != nil {
				return nil, walkErr
			}
			continue
		}

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
	lines := strings.Split(content, "\n")
	var diffLines []internal.DiffLine
	for _, line := range lines {
		if line == "" && len(diffLines) > 0 {
			continue
		}
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
			Header: fmt.Sprintf("@@ -0,0 +1,%d @@", addCount),
			Lines:  diffLines,
		}},
		AddCount: addCount,
	}
}
