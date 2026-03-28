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
		fmt.Println(err)
		return
	}

	root, err := git.FindWorktreeRoot(cwd)
	if err != nil {
		fmt.Println("Not a git repository. Run `git init` to get started.")
		return
	}

	head, err := git.GetHeadSHA(root)
	if err != nil {
		fmt.Println(err)
		return
	}

	name := filepath.Base(root)
	sha := head
	if len(sha) > 7 {
		sha = sha[:7]
	}

	fmt.Printf("%s · baseline: %s\n", name, sha)
}
