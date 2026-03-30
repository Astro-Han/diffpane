// Package main wires repository diffing, watching, and the TUI into one CLI.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	internal "github.com/Astro-Han/diffpane/internal"
	gitpkg "github.com/Astro-Han/diffpane/internal/git"
	"github.com/Astro-Han/diffpane/internal/ui"
	"github.com/Astro-Han/diffpane/internal/watcher"
	tea "github.com/charmbracelet/bubbletea"
)

type messageSender interface {
	Send(msg tea.Msg)
}

// computeDiffFunc abstracts diff computation for shared-state helpers.
type computeDiffFunc func(root, baselineSHA string) ([]internal.FileDiff, error)

// getHeadSHAFunc abstracts HEAD resolution for shared-state helpers.
type getHeadSHAFunc func(root string) (string, error)

// getBranchNameFunc abstracts branch-name lookup for shared-state helpers.
type getBranchNameFunc func(root string) string

// sessionBaselineState stores mutable baseline-tracking state shared by the UI
// reset callback and filesystem watcher goroutines.
type sessionBaselineState struct {
	mu          sync.Mutex
	baseline    string
	lastHeadSHA string
	branch      string
}

// main starts the TUI and wires file-system events into diff recomputation.
func main() {
	// Resolve theme before Bubble Tea takes over terminal I/O.
	ui.InitTheme()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	root, err := gitpkg.FindWorktreeRoot(cwd)
	if err != nil {
		fmt.Println("Not a git repository. Run git init to get started.")
		return
	}

	sha, err := gitpkg.GetHeadSHA(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading HEAD: %v\n", err)
		os.Exit(1)
	}

	files, err := gitpkg.ComputeDiff(root, sha)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error computing diff: %v\n", err)
		os.Exit(1)
	}

	dirName := filepath.Base(root)
	state := &sessionBaselineState{
		baseline:    sha,
		lastHeadSHA: sha,
		branch:      gitpkg.GetBranchName(root),
	}
	gitDir := gitpkg.ResolveGitDir(root)
	if gitDir == "" {
		fmt.Fprintln(os.Stderr, "Error starting watcher: could not resolve git directory")
		os.Exit(1)
	}
	commonGitDir := gitpkg.GetGitCommonDir(root)
	if commonGitDir == "" {
		commonGitDir = gitDir
	}

	model := ui.NewModel(dirName, root, sha, files)
	model.ResetBaseline = func() (string, []internal.FileDiff, error) {
		return resetSessionBaseline(root, state, gitpkg.ComputeDiff)
	}
	program := tea.NewProgram(model, tea.WithAltScreen())

	fileWatcher, err := watcher.New(
		root,
		gitDir,
		commonGitDir,
		func(changedPaths []string) {
			state.mu.Lock()
			currentBaseline := state.baseline
			state.mu.Unlock()
			sendFilesUpdated(os.Stderr, program, root, currentBaseline, changedPaths)
		},
		func() {
			handleHeadChange(
				os.Stderr,
				program,
				root,
				state,
				gitpkg.GetHeadSHA,
				gitpkg.GetBranchName,
				gitpkg.ComputeDiff,
			)
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting watcher: %v\n", err)
		os.Exit(1)
	}
	defer fileWatcher.Stop()

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// handleHeadChange refreshes the diff when HEAD or branch changes without
// moving the session baseline forward.
func handleHeadChange(
	stderr io.Writer,
	sender messageSender,
	root string,
	state *sessionBaselineState,
	getHeadSHA getHeadSHAFunc,
	getBranchName getBranchNameFunc,
	computeDiff computeDiffFunc,
) {
	newSHA, shaErr := getHeadSHA(root)
	if shaErr != nil {
		fmt.Fprintf(stderr, "Error reading HEAD: %v\n", shaErr)
		return
	}
	newBranch := getBranchName(root)

	state.mu.Lock()
	changed := newSHA != state.lastHeadSHA || newBranch != state.branch
	currentBaseline := state.baseline
	state.mu.Unlock()

	if !changed {
		return
	}

	if !sendFilesUpdatedWithCompute(stderr, sender, root, currentBaseline, nil, computeDiff) {
		return
	}

	state.mu.Lock()
	state.lastHeadSHA = newSHA
	state.branch = newBranch
	state.mu.Unlock()
}

func sendFilesUpdated(stderr io.Writer, sender messageSender, root, baselineSHA string, changedPaths []string) {
	_ = sendFilesUpdatedWithCompute(stderr, sender, root, baselineSHA, changedPaths, gitpkg.ComputeDiff)
}

// sendFilesUpdatedWithCompute recomputes the diff for the provided baseline and
// pushes the result into the Bubble Tea program.
func sendFilesUpdatedWithCompute(
	stderr io.Writer,
	sender messageSender,
	root,
	baselineSHA string,
	changedPaths []string,
	computeDiff computeDiffFunc,
) bool {
	newFiles, computeErr := computeDiff(root, baselineSHA)
	if computeErr != nil {
		_, _ = fmt.Fprintf(stderr, "Error computing diff: %v\n", computeErr)
		return false
	}
	sender.Send(ui.FilesUpdatedMsg{
		BaselineSHA:  baselineSHA,
		Files:        newFiles,
		ChangedPaths: changedPaths,
	})
	return true
}

// resetSessionBaseline recomputes the diff against the latest HEAD and only
// updates the shared baseline after diff computation succeeds.
func resetSessionBaseline(root string, state *sessionBaselineState, computeDiff computeDiffFunc) (string, []internal.FileDiff, error) {
	state.mu.Lock()
	newSHA := state.lastHeadSHA
	state.mu.Unlock()

	newFiles, err := computeDiff(root, newSHA)
	if err != nil {
		return "", nil, err
	}

	state.mu.Lock()
	state.baseline = newSHA
	state.mu.Unlock()

	return newSHA, newFiles, nil
}
