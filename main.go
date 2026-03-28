package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Astro-Han/diffpane/internal/git"
)

// main prints the current worktree baseline or a git init hint.
func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	root, err := git.FindWorktreeRoot(cwd)
	if err != nil {
		fmt.Println("Not a git repository. Run `git init` to get started.")
		os.Exit(1)
	}

	head, err := git.GetHeadSHA(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading HEAD: %v\n", err)
		os.Exit(1)
	}

	name := filepath.Base(root)
	sha := head
	if len(sha) > 7 {
		sha = sha[:7]
	}

	fmt.Printf("%s · baseline: %s\n", name, sha)
}
