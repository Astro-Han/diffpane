package main

import (
	"bytes"
	"errors"
	"testing"

	internal "github.com/Astro-Han/diffpane/internal"
	"github.com/Astro-Han/diffpane/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type recordingSender struct {
	msgs []any
}

func (s *recordingSender) Send(msg tea.Msg) {
	s.msgs = append(s.msgs, msg)
}

// TestHandleHeadChangeKeepsSessionBaseline verifies HEAD changes recompute
// against the session baseline instead of moving it forward automatically.
func TestHandleHeadChangeKeepsSessionBaseline(t *testing.T) {
	state := &sessionBaselineState{
		baseline:    "session-sha",
		lastHeadSHA: "old-head",
		branch:      "main",
	}
	sender := &recordingSender{}
	var gotBaseline string

	handleHeadChange(
		&bytes.Buffer{},
		sender,
		"/tmp/repo",
		state,
		func(string) (string, error) {
			return "new-head", nil
		},
		func(string) string {
			return "feature"
		},
		func(string, string) ([]internal.FileDiff, error) {
			gotBaseline = state.baseline
			return []internal.FileDiff{{Path: "tracked.txt", AddCount: 1}}, nil
		},
	)

	if state.baseline != "session-sha" {
		t.Fatalf("baseline = %q, want session baseline unchanged", state.baseline)
	}
	if state.lastHeadSHA != "new-head" {
		t.Fatalf("lastHeadSHA = %q, want updated HEAD", state.lastHeadSHA)
	}
	if state.branch != "feature" {
		t.Fatalf("branch = %q, want updated branch", state.branch)
	}
	if gotBaseline != "session-sha" {
		t.Fatalf("diff baseline = %q, want session baseline", gotBaseline)
	}
	if len(sender.msgs) != 1 {
		t.Fatalf("msg count = %d, want 1", len(sender.msgs))
	}
	msg, ok := sender.msgs[0].(ui.FilesUpdatedMsg)
	if !ok {
		t.Fatalf("message type = %T, want ui.FilesUpdatedMsg", sender.msgs[0])
	}
	if msg.BaselineSHA != "session-sha" {
		t.Fatalf("BaselineSHA = %q, want session baseline", msg.BaselineSHA)
	}
	if len(msg.Files) != 1 || msg.Files[0].Path != "tracked.txt" {
		t.Fatalf("Files = %#v, want tracked.txt", msg.Files)
	}
}

// TestHandleHeadChangeRetriesAfterComputeFailure verifies a transient diff
// failure does not mark the new HEAD as already handled.
func TestHandleHeadChangeRetriesAfterComputeFailure(t *testing.T) {
	state := &sessionBaselineState{
		baseline:    "session-sha",
		lastHeadSHA: "old-head",
		branch:      "main",
	}
	sender := &recordingSender{}
	stderr := &bytes.Buffer{}
	attempts := 0

	compute := func(string, string) ([]internal.FileDiff, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("boom")
		}
		return []internal.FileDiff{{Path: "tracked.txt", AddCount: 1}}, nil
	}

	handleHeadChange(
		stderr,
		sender,
		"/tmp/repo",
		state,
		func(string) (string, error) { return "new-head", nil },
		func(string) string { return "feature" },
		compute,
	)
	if state.lastHeadSHA != "old-head" {
		t.Fatalf("lastHeadSHA = %q, want unchanged after failed compute", state.lastHeadSHA)
	}
	if state.branch != "main" {
		t.Fatalf("branch = %q, want unchanged after failed compute", state.branch)
	}
	if len(sender.msgs) != 0 {
		t.Fatalf("msg count = %d, want 0 after failed compute", len(sender.msgs))
	}

	handleHeadChange(
		stderr,
		sender,
		"/tmp/repo",
		state,
		func(string) (string, error) { return "new-head", nil },
		func(string) string { return "feature" },
		compute,
	)
	if state.lastHeadSHA != "new-head" {
		t.Fatalf("lastHeadSHA = %q, want new-head after retry", state.lastHeadSHA)
	}
	if state.branch != "feature" {
		t.Fatalf("branch = %q, want feature after retry", state.branch)
	}
	if len(sender.msgs) != 1 {
		t.Fatalf("msg count = %d, want 1 after successful retry", len(sender.msgs))
	}
}

// TestResetSessionBaselineUpdatesOnlyAfterSuccessfulDiff verifies manual reset
// re-anchors to the latest HEAD only after diff computation succeeds.
func TestResetSessionBaselineUpdatesOnlyAfterSuccessfulDiff(t *testing.T) {
	state := &sessionBaselineState{
		baseline:    "session-sha",
		lastHeadSHA: "new-head",
		branch:      "main",
	}

	newSHA, files, err := resetSessionBaseline("/tmp/repo", state, func(string, string) ([]internal.FileDiff, error) {
		return []internal.FileDiff{{Path: "fresh.txt", AddCount: 2}}, nil
	})
	if err != nil {
		t.Fatalf("resetSessionBaseline returned error: %v", err)
	}
	if newSHA != "new-head" {
		t.Fatalf("newSHA = %q, want new-head", newSHA)
	}
	if state.baseline != "new-head" {
		t.Fatalf("baseline = %q, want new-head", state.baseline)
	}
	if len(files) != 1 || files[0].Path != "fresh.txt" {
		t.Fatalf("Files = %#v, want fresh.txt", files)
	}
}

// TestResetSessionBaselinePreservesBaselineOnDiffError verifies failed manual
// reset does not move the baseline and leave shared state inconsistent.
func TestResetSessionBaselinePreservesBaselineOnDiffError(t *testing.T) {
	state := &sessionBaselineState{
		baseline:    "session-sha",
		lastHeadSHA: "new-head",
		branch:      "main",
	}

	_, _, err := resetSessionBaseline("/tmp/repo", state, func(string, string) ([]internal.FileDiff, error) {
		return nil, errors.New("boom")
	})
	if err == nil {
		t.Fatal("resetSessionBaseline returned nil error, want failure")
	}
	if state.baseline != "session-sha" {
		t.Fatalf("baseline = %q, want unchanged session baseline", state.baseline)
	}
}
