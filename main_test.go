package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestMainNotGitRepoExitsWithCodeOne verifies the CLI reports non-git usage as a failure.
func TestMainNotGitRepoExitsWithCodeOne(t *testing.T) {
	dir := t.TempDir()

	// #nosec G204,G702 -- tests execute the current test binary with controlled args.
	cmd := exec.Command(os.Args[0], "-test.run=TestMainHelperProcess", "--", dir)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected process exit error, got %v with output %q", err, out)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("exit code = %d, want 1", exitErr.ExitCode())
	}

	got := strings.TrimSpace(string(out))
	want := "Not a git repository. Run `git init` to get started."
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

// TestMainHelperProcess runs main in a subprocess so os.Exit terminates only the child.
func TestMainHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	dir := os.Args[len(os.Args)-1]
	if err := os.Chdir(dir); err != nil {
		os.Exit(2)
	}

	main()
	os.Exit(0)
}
