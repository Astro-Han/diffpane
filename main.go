// Package main wires repository diffing, watching, and the TUI into one CLI.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	gitpkg "github.com/Astro-Han/diffpane/internal/git"
	"github.com/Astro-Han/diffpane/internal/ui"
	"github.com/Astro-Han/diffpane/internal/watcher"
	tea "github.com/charmbracelet/bubbletea"
)

// main starts the TUI and wires file-system events into diff recomputation.
func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	root, err := gitpkg.FindWorktreeRoot(cwd)
	if err != nil {
		fmt.Println("Not a git repository. Run `git init` to get started.")
		os.Exit(1)
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
	model := ui.NewModel(dirName, root, sha, files)
	program := tea.NewProgram(model, tea.WithAltScreen())

	// The watcher callbacks share mutable baseline state across goroutines.
	var mu sync.Mutex
	baseline := sha
	branch := gitpkg.GetBranchName(root)
	gitDir := gitpkg.ResolveGitDir(root)

	fileWatcher, err := watcher.New(
		root,
		gitDir,
		func(changedPaths []string) {
			mu.Lock()
			currentBaseline := baseline
			mu.Unlock()

			newFiles, computeErr := gitpkg.ComputeDiff(root, currentBaseline)
			if computeErr != nil {
				return
			}
			program.Send(ui.FilesUpdatedMsg{Files: newFiles, ChangedPaths: changedPaths})
		},
		func() {
			newSHA, shaErr := gitpkg.GetHeadSHA(root)
			if shaErr != nil {
				return
			}
			newBranch := gitpkg.GetBranchName(root)

			mu.Lock()
			changed := newSHA != baseline || newBranch != branch
			if changed {
				baseline = newSHA
				branch = newBranch
			}
			mu.Unlock()

			if !changed {
				return
			}

			program.Send(ui.BaselineResetMsg{NewSHA: newSHA})
			newFiles, computeErr := gitpkg.ComputeDiff(root, newSHA)
			if computeErr != nil {
				return
			}
			program.Send(ui.FilesUpdatedMsg{Files: newFiles, ChangedPaths: nil})
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
