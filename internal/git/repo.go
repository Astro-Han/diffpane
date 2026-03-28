package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EmptyTreeSHA is the SHA for an empty git tree object.
const EmptyTreeSHA = "4b825dc642cb6eb9a060e54bf899d15006578022"

// FindWorktreeRoot returns the top-level directory for the git worktree at path.
func FindWorktreeRoot(path string) (string, error) {
	out, err := gitOutput(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

// GetHeadSHA returns HEAD for repoDir, or EmptyTreeSHA for an unborn repo.
func GetHeadSHA(repoDir string) (string, error) {
	root, err := FindWorktreeRoot(repoDir)
	if err != nil {
		return "", err
	}

	out, err := gitOutput(root, "rev-parse", "HEAD")
	if err == nil {
		return strings.TrimSpace(out), nil
	}

	// Empty repos have no reachable commits from any ref yet.
	allCommits, listErr := gitOutput(root, "rev-list", "--max-count=1", "--all")
	if listErr == nil && strings.TrimSpace(allCommits) == "" {
		return EmptyTreeSHA, nil
	}

	return "", err
}

// GetBranchName returns the current branch name, or an empty string when detached.
func GetBranchName(repoDir string) string {
	root, err := FindWorktreeRoot(repoDir)
	if err != nil {
		return ""
	}

	out, err := gitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(out)
}

// ResolveGitDir returns the .git path for repoDir, including linked worktrees.
func ResolveGitDir(repoDir string) string {
	gitPath := filepath.Join(repoDir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return gitPath
	}

	// #nosec G304 -- gitPath is always repoDir/.git for the current repository.
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir:") {
		return ""
	}

	gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if gitDir == "" {
		return ""
	}
	if filepath.IsAbs(gitDir) {
		return gitDir
	}

	return filepath.Clean(filepath.Join(repoDir, gitDir))
}

// gitOutput runs git in dir and returns stdout.
func gitOutput(dir string, args ...string) (string, error) {
	// #nosec G204 -- git is fixed and arguments are constructed by trusted callers.
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}

	return stdout.String(), nil
}
