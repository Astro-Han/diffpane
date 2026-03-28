package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestMainNotGitRepoExitsCleanly verifies the CLI reports non-git usage
// without treating the friendly message as an error.
func TestMainNotGitRepoExitsCleanly(t *testing.T) {
	dir := t.TempDir()

	// #nosec G204,G702 -- tests execute the current test binary with controlled args.
	cmd := exec.Command(os.Args[0], "-test.run=TestMainHelperProcess", "--", dir)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	out, err := cmd.CombinedOutput()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("exit code = %d, want 0; output = %q", exitErr.ExitCode(), out)
		}
		t.Fatalf("expected clean exit, got %v with output %q", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := "Not a git repository. Run git init to get started."
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

// TestSendFilesUpdatedLogsComputeErrors verifies asynchronous diff failures are
// surfaced instead of silently ignored.
func TestSendFilesUpdatedLogsComputeErrors(t *testing.T) {
	var stderr bytes.Buffer
	sender := &fakeSender{}

	sendFilesUpdated(&stderr, sender, "/definitely/missing", "sha", []string{"a.txt"})

	if !strings.Contains(stderr.String(), "Error computing diff:") {
		t.Fatalf("stderr = %q, want compute diff error", stderr.String())
	}
	if len(sender.messages) != 0 {
		t.Fatalf("expected no messages on error, got %d", len(sender.messages))
	}
}

type fakeSender struct {
	messages []tea.Msg
}

func (f *fakeSender) Send(msg tea.Msg) {
	f.messages = append(f.messages, msg)
}
