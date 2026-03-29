package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/Astro-Han/diffpane/internal"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func file(path string, adds int) internal.FileDiff {
	return internal.FileDiff{Path: path, AddCount: adds}
}

// TestModelFollowSelectsLatestChanged verifies follow mode jumps to the latest changed path.
func TestModelFollowSelectsLatestChanged(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
	})

	updated, _ := model.Update(FilesUpdatedMsg{
		Files: []internal.FileDiff{
			file("a.txt", 1),
			file("b.txt", 1),
			file("c.txt", 1),
		},
		ChangedPaths: []string{"c.txt"},
	})

	got := updated.(Model)
	if got.CurrentIdx != 2 {
		t.Fatalf("CurrentIdx = %d, want 2", got.CurrentIdx)
	}
	if got.NewCount != 0 {
		t.Fatalf("NewCount = %d, want 0", got.NewCount)
	}
}

// TestModelFollowUsesMostRecentChangedPath verifies active follow mode prefers
// the last changed path in the debounce batch instead of file sort order.
func TestModelFollowUsesMostRecentChangedPath(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
		file("c.txt", 1),
	})

	updated, _ := model.Update(FilesUpdatedMsg{
		Files: []internal.FileDiff{
			file("a.txt", 1),
			file("b.txt", 1),
			file("c.txt", 1),
		},
		ChangedPaths: []string{"c.txt", "b.txt"},
	})

	got := updated.(Model)
	if got.CurrentIdx != 1 {
		t.Fatalf("CurrentIdx = %d, want 1 for most recent path b.txt", got.CurrentIdx)
	}
}

// TestModelPausedFollowTracksNewFiles verifies paused follow accumulates new unique files.
func TestModelPausedFollowTracksNewFiles(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
	})
	model.FollowOn = false
	model.CurrentIdx = 0

	updated, _ := model.Update(FilesUpdatedMsg{
		Files: []internal.FileDiff{
			file("a.txt", 1),
			file("b.txt", 1),
			file("c.txt", 1),
		},
		ChangedPaths: []string{"a.txt", "c.txt", "c.txt"},
	})

	got := updated.(Model)
	if got.CurrentIdx != 0 {
		t.Fatalf("CurrentIdx = %d, want 0", got.CurrentIdx)
	}
	if got.NewCount != 1 {
		t.Fatalf("NewCount = %d, want 1", got.NewCount)
	}
	if !got.NewFiles["c.txt"] {
		t.Fatal("expected c.txt to be tracked as new")
	}
}

// TestModelPausedFollowIgnoresVanishedChangedPaths verifies transient changes
// that disappear before the diff refresh lands do not inflate +N new.
func TestModelPausedFollowIgnoresVanishedChangedPaths(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
	})
	model.FollowOn = false
	model.CurrentIdx = 0

	updated, _ := model.Update(FilesUpdatedMsg{
		Files: []internal.FileDiff{
			file("a.txt", 1),
		},
		ChangedPaths: []string{"temp.txt"},
	})

	got := updated.(Model)
	if got.NewCount != 0 {
		t.Fatalf("NewCount = %d, want 0", got.NewCount)
	}
	if len(got.NewFiles) != 0 {
		t.Fatalf("NewFiles = %#v, want empty", got.NewFiles)
	}
}

// TestModelOverlayQueuesUpdates verifies overlay mode freezes the view and applies pending updates on close.
func TestModelOverlayQueuesUpdates(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
	})

	opened, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	withOverlay := opened.(Model)
	if !withOverlay.OverlayOpen {
		t.Fatal("expected overlay to be open")
	}

	queued, _ := withOverlay.Update(FilesUpdatedMsg{
		Files: []internal.FileDiff{
			file("a.txt", 1),
			file("b.txt", 1),
			file("c.txt", 1),
		},
		ChangedPaths: []string{"c.txt"},
	})
	pending := queued.(Model)
	if pending.PendingUpdate == nil {
		t.Fatal("expected pending update while overlay is open")
	}

	closed, _ := pending.Update(tea.KeyMsg{Type: tea.KeyEsc})
	final := closed.(Model)
	if final.OverlayOpen {
		t.Fatal("expected overlay to close")
	}
	if len(final.Files) != 3 {
		t.Fatalf("file count = %d, want 3", len(final.Files))
	}
}

// TestModelIgnoresStaleFilesUpdate verifies out-of-date baseline updates do not overwrite state.
func TestModelIgnoresStaleFilesUpdate(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "new-sha", []internal.FileDiff{
		file("stable.txt", 1),
	})

	updated, _ := model.Update(FilesUpdatedMsg{
		BaselineSHA: "old-sha",
		Files: []internal.FileDiff{
			file("stale.txt", 1),
		},
		ChangedPaths: []string{"stale.txt"},
	})

	got := updated.(Model)
	if len(got.Files) != 1 || got.Files[0].Path != "stable.txt" {
		t.Fatalf("stale update should be ignored, got %#v", got.Files)
	}
}

// TestModelRestoreFollowJumpsToLatestChanged verifies toggling follow back on
// immediately selects the latest pending file instead of waiting for another fs event.
func TestModelRestoreFollowJumpsToLatestChanged(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
		file("c.txt", 1),
	})
	model.FollowOn = false
	model.CurrentIdx = 0
	model.NewFiles["c.txt"] = true
	model.NewCount = 1

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	got := updated.(Model)
	if !got.FollowOn {
		t.Fatal("expected follow to be restored")
	}
	if got.CurrentIdx != 2 {
		t.Fatalf("CurrentIdx = %d, want 2", got.CurrentIdx)
	}
	if got.NewCount != 0 {
		t.Fatalf("NewCount = %d, want 0", got.NewCount)
	}
}

// TestModelRestoreFollowUsesMostRecentChangedPath verifies follow restore uses
// change order instead of file sort order when multiple pending files exist.
func TestModelRestoreFollowUsesMostRecentChangedPath(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
		file("c.txt", 1),
	})
	model.FollowOn = false
	model.CurrentIdx = 0

	updated, _ := model.Update(FilesUpdatedMsg{
		BaselineSHA: "sha",
		Files: []internal.FileDiff{
			file("a.txt", 1),
			file("b.txt", 1),
			file("c.txt", 1),
		},
		ChangedPaths: []string{"c.txt", "b.txt"},
	})
	restored, _ := updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	got := restored.(Model)
	if got.CurrentIdx != 1 {
		t.Fatalf("CurrentIdx = %d, want 1 for most recent path b.txt", got.CurrentIdx)
	}
}

// TestModelFollowAnchorsWhenPrecedingFilesDisappear verifies that follow mode
// keeps the current file selected when earlier files are removed from the list.
func TestModelFollowAnchorsWhenPrecedingFilesDisappear(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
		file("c.txt", 1),
		file("d.txt", 1),
	})
	model.CurrentIdx = 2 // viewing c.txt

	updated, _ := model.Update(FilesUpdatedMsg{
		Files: []internal.FileDiff{
			file("b.txt", 1),
			file("c.txt", 1),
			file("d.txt", 1),
		},
		ChangedPaths: []string{"a.txt"}, // reverted, no longer in list
	})

	got := updated.(Model)
	if got.CurrentIdx != 1 || got.Files[got.CurrentIdx].Path != "c.txt" {
		t.Fatalf("CurrentIdx = %d (%s), want 1 (c.txt)", got.CurrentIdx, got.Files[got.CurrentIdx].Path)
	}
}

// TestModelFollowClampsWhenCurrentFileDisappears verifies that follow mode
// selects the adjacent file when the currently viewed file is reverted.
func TestModelFollowClampsWhenCurrentFileDisappears(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
		file("c.txt", 1),
	})
	model.CurrentIdx = 2 // viewing c.txt

	updated, _ := model.Update(FilesUpdatedMsg{
		Files: []internal.FileDiff{
			file("a.txt", 1),
			file("b.txt", 1),
		},
		ChangedPaths: []string{"c.txt"}, // reverted, gone
	})

	got := updated.(Model)
	// Old CurrentIdx was 2, list now has 2 items, clamp to 1 (b.txt).
	if got.CurrentIdx != 1 {
		t.Fatalf("CurrentIdx = %d, want 1", got.CurrentIdx)
	}
}

// TestModelOverlayIgnoresResetKey verifies r key is ignored when overlay is open.
func TestModelOverlayIgnoresResetKey(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
	})
	model.Width = 80
	model.Height = 24

	opened, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	m := opened.(Model)
	if !m.OverlayOpen {
		t.Fatal("expected overlay open")
	}

	afterR, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got := afterR.(Model)
	if got.resetPending {
		t.Fatal("resetPending should be false in overlay")
	}
}

// TestModelManualResetDoublePressR verifies double-press r dispatches async
// reset and ManualResetMsg applies correctly.
func TestModelManualResetDoublePressR(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "old-sha", []internal.FileDiff{
		file("old.txt", 1),
	})
	model.Width = 80
	model.Height = 24
	model.ResetBaseline = func() (string, []internal.FileDiff, error) {
		return "new-sha", []internal.FileDiff{file("new.txt", 2)}, nil
	}

	first, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m := first.(Model)
	if !m.resetPending {
		t.Fatal("expected resetPending after first r")
	}
	if m.Notification != "press r to reset baseline" {
		t.Fatalf("notification = %q, want 'press r to reset baseline'", m.Notification)
	}

	second, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = second.(Model)
	if m.resetPending {
		t.Fatal("resetPending should be false after second r")
	}
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from second r press")
	}

	afterReset, _ := m.Update(ManualResetMsg{
		NewSHA: "new-sha",
		Files:  []internal.FileDiff{file("new.txt", 2)},
	})
	got := afterReset.(Model)
	if got.BaselineSHA != "new-sha" {
		t.Fatalf("BaselineSHA = %q, want 'new-sha'", got.BaselineSHA)
	}
	if len(got.Files) != 1 || got.Files[0].Path != "new.txt" {
		t.Fatalf("Files = %v, want [new.txt]", got.Files)
	}
	if got.Notification != "baseline reset" {
		t.Fatalf("notification = %q, want 'baseline reset'", got.Notification)
	}
}

// TestModelResetCancelledByOtherKey verifies non-r key cancels pending reset
// and dispatches normally.
func TestModelResetCancelledByOtherKey(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
		file("b.txt", 1),
	})
	model.Width = 80
	model.Height = 24
	model.ResetBaseline = func() (string, []internal.FileDiff, error) {
		return "sha", model.Files, nil
	}

	first, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m := first.(Model)
	if !m.resetPending {
		t.Fatal("expected resetPending")
	}

	afterN, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := afterN.(Model)
	if got.resetPending {
		t.Fatal("resetPending should be false after n")
	}
	if got.Notification != "" {
		t.Fatalf("notification = %q, want empty", got.Notification)
	}
	if got.CurrentIdx != 1 {
		t.Fatalf("CurrentIdx = %d, want 1 (n should navigate)", got.CurrentIdx)
	}
}

// TestModelResetTimeout verifies pending reset is cancelled by timeout.
func TestModelResetTimeout(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
	})
	model.Width = 80
	model.Height = 24
	model.ResetBaseline = func() (string, []internal.FileDiff, error) {
		return "sha", model.Files, nil
	}

	first, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m := first.(Model)
	if !m.resetPending {
		t.Fatal("expected resetPending")
	}

	afterTimeout, _ := m.Update(ResetTimeoutMsg{})
	got := afterTimeout.(Model)
	if got.resetPending {
		t.Fatal("resetPending should be false after timeout")
	}
	if got.Notification != "" {
		t.Fatalf("notification = %q, want empty", got.Notification)
	}
}

// TestModelManualResetToEmptyFiles verifies reset handles empty file list.
func TestModelManualResetToEmptyFiles(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "old-sha", []internal.FileDiff{
		file("a.txt", 1),
	})
	model.Width = 80
	model.Height = 24

	afterReset, _ := model.Update(ManualResetMsg{
		NewSHA: "new-sha",
		Files:  nil,
	})
	got := afterReset.(Model)
	if got.BaselineSHA != "new-sha" {
		t.Fatalf("BaselineSHA = %q, want 'new-sha'", got.BaselineSHA)
	}
	if len(got.Files) != 0 {
		t.Fatalf("Files should be empty, got %d", len(got.Files))
	}
	if got.CurrentIdx != 0 {
		t.Fatalf("CurrentIdx = %d, want 0", got.CurrentIdx)
	}
}

// TestModelManualResetFailureShowsNotification verifies reset errors surface a
// temporary footer notification instead of failing silently.
func TestModelManualResetFailureShowsNotification(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "old-sha", []internal.FileDiff{
		file("old.txt", 1),
	})
	model.Width = 80
	model.Height = 24
	model.ResetBaseline = func() (string, []internal.FileDiff, error) {
		return "", nil, errors.New("boom")
	}

	first, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	second, cmd := first.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m := second.(Model)
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from second r press")
	}

	msg := cmd()
	failed, ok := msg.(ManualResetFailedMsg)
	if !ok {
		t.Fatalf("message type = %T, want ManualResetFailedMsg", msg)
	}
	if failed.Error != "boom" {
		t.Fatalf("error = %q, want boom", failed.Error)
	}

	afterFailure, _ := m.Update(failed)
	got := afterFailure.(Model)
	if got.BaselineSHA != "old-sha" {
		t.Fatalf("BaselineSHA = %q, want unchanged", got.BaselineSHA)
	}
	if got.Notification != "baseline reset failed: boom" {
		t.Fatalf("notification = %q, want reset failure notice", got.Notification)
	}
}

// TestModelManualResetCanRunConsecutively verifies reset state fully clears so
// users can confirm and run another reset immediately afterward.
func TestModelManualResetCanRunConsecutively(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "old-sha", []internal.FileDiff{
		file("old.txt", 1),
	})
	model.Width = 80
	model.Height = 24

	callCount := 0
	model.ResetBaseline = func() (string, []internal.FileDiff, error) {
		callCount++
		switch callCount {
		case 1:
			return "sha-1", []internal.FileDiff{file("one.txt", 1)}, nil
		case 2:
			return "sha-2", []internal.FileDiff{file("two.txt", 2)}, nil
		default:
			t.Fatalf("unexpected reset call %d", callCount)
			return "", nil, nil
		}
	}

	firstPress, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	secondPress, firstCmd := firstPress.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if firstCmd == nil {
		t.Fatal("expected first reset cmd")
	}
	firstMsg, ok := firstCmd().(ManualResetMsg)
	if !ok {
		t.Fatalf("message type = %T, want ManualResetMsg", firstCmd())
	}
	afterFirstReset, _ := secondPress.(Model).Update(firstMsg)
	afterFirst := afterFirstReset.(Model)
	if afterFirst.BaselineSHA != "sha-1" {
		t.Fatalf("BaselineSHA = %q, want sha-1", afterFirst.BaselineSHA)
	}

	thirdPress, _ := afterFirst.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	pendingAgain := thirdPress.(Model)
	if !pendingAgain.resetPending {
		t.Fatal("expected resetPending after starting second reset")
	}
	fourthPress, secondCmd := pendingAgain.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if secondCmd == nil {
		t.Fatal("expected second reset cmd")
	}
	secondMsg, ok := secondCmd().(ManualResetMsg)
	if !ok {
		t.Fatalf("message type = %T, want ManualResetMsg", secondCmd())
	}
	afterSecondReset, _ := fourthPress.(Model).Update(secondMsg)
	got := afterSecondReset.(Model)
	if got.BaselineSHA != "sha-2" {
		t.Fatalf("BaselineSHA = %q, want sha-2", got.BaselineSHA)
	}
	if len(got.Files) != 1 || got.Files[0].Path != "two.txt" {
		t.Fatalf("Files = %v, want [two.txt]", got.Files)
	}
}

// TestModelManualResetQueuesUpdateWhileOverlayOpen verifies reset results do
// not mutate the underlying file list until the overlay closes.
func TestModelManualResetQueuesUpdateWhileOverlayOpen(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "old-sha", []internal.FileDiff{
		file("old-a.txt", 1),
		file("old-b.txt", 1),
	})
	model.Width = 80
	model.Height = 24
	model.CurrentIdx = 1
	model.ResetBaseline = func() (string, []internal.FileDiff, error) {
		return "new-sha", []internal.FileDiff{file("new.txt", 2)}, nil
	}

	first, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	second, cmd := first.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected reset cmd")
	}

	opened, _ := second.(Model).Update(tea.KeyMsg{Type: tea.KeyTab})
	withOverlay := opened.(Model)
	if !withOverlay.OverlayOpen {
		t.Fatal("expected overlay open")
	}

	afterMsg, _ := withOverlay.Update(ManualResetMsg{
		NewSHA: "new-sha",
		Files:  []internal.FileDiff{file("new.txt", 2)},
	})
	pending := afterMsg.(Model)
	if pending.BaselineSHA != "new-sha" {
		t.Fatalf("BaselineSHA = %q, want new-sha", pending.BaselineSHA)
	}
	if len(pending.Files) != 2 || pending.Files[0].Path != "old-a.txt" {
		t.Fatalf("Files changed under overlay: %v", pending.Files)
	}
	if pending.PendingUpdate == nil {
		t.Fatal("expected pending update while overlay stays open")
	}
	if len(pending.OverlaySnapshot) != 2 || pending.OverlaySnapshot[0].Path != "old-a.txt" {
		t.Fatalf("OverlaySnapshot = %v, want old snapshot", pending.OverlaySnapshot)
	}

	closed, _ := pending.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := closed.(Model)
	if len(got.Files) != 1 || got.Files[0].Path != "new.txt" {
		t.Fatalf("Files = %v, want [new.txt]", got.Files)
	}
	if got.CurrentIdx != 0 {
		t.Fatalf("CurrentIdx = %d, want 0 after clamping", got.CurrentIdx)
	}
}

// TestModelManualResetIgnoresSecondRequestWhileInFlight verifies a second reset
// cannot start before the previous async reset result returns.
func TestModelManualResetIgnoresSecondRequestWhileInFlight(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "old-sha", []internal.FileDiff{
		file("old.txt", 1),
	})
	model.Width = 80
	model.Height = 24
	callCount := 0
	model.ResetBaseline = func() (string, []internal.FileDiff, error) {
		callCount++
		return "new-sha", []internal.FileDiff{file("new.txt", 1)}, nil
	}

	first, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	second, firstCmd := first.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	inFlight := second.(Model)
	if firstCmd == nil {
		t.Fatal("expected first reset cmd")
	}
	if !inFlight.resetInFlight {
		t.Fatal("expected resetInFlight after dispatching reset")
	}

	third, secondCmd := inFlight.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	stillInFlight := third.(Model)
	if secondCmd != nil {
		t.Fatal("second reset should be ignored while first is in flight")
	}
	if !stillInFlight.resetInFlight {
		t.Fatal("resetInFlight should stay true until result arrives")
	}
	if stillInFlight.resetPending {
		t.Fatal("resetPending should stay false while request is in flight")
	}
	if callCount != 0 {
		t.Fatalf("ResetBaseline should not run until cmd executes, got %d calls", callCount)
	}

	msg := firstCmd()
	resetMsg, ok := msg.(ManualResetMsg)
	if !ok {
		t.Fatalf("message type = %T, want ManualResetMsg", msg)
	}
	if callCount != 1 {
		t.Fatalf("ResetBaseline call count = %d, want 1", callCount)
	}

	afterReset, _ := stillInFlight.Update(resetMsg)
	got := afterReset.(Model)
	if got.resetInFlight {
		t.Fatal("resetInFlight should clear after result")
	}
	if got.BaselineSHA != "new-sha" {
		t.Fatalf("BaselineSHA = %q, want new-sha", got.BaselineSHA)
	}
}

// TestRenderFooterIncludesResetKey verifies footer shows r reset.
func TestRenderFooterIncludesResetKey(t *testing.T) {
	footer := RenderFooter(true, "", 80)
	if !strings.Contains(footer, "r reset") {
		t.Fatalf("footer missing 'r reset': %q", footer)
	}
}

// TestModelViewSmallHeightDoesNotPanic verifies tiny terminal heights do not
// trigger negative content heights in overlay rendering.
func TestModelViewSmallHeightDoesNotPanic(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		file("a.txt", 1),
	})
	model.Width = 80
	model.Height = 1
	model.OverlayOpen = true
	model.OverlaySnapshot = append([]internal.FileDiff(nil), model.Files...)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked for small height: %v", r)
		}
	}()

	_ = model.View()
}

// TestModelViewEmptyStateFitsViewport verifies the empty-state header and footer
// both remain visible within the terminal height instead of scrolling off-screen.
func TestModelViewEmptyStateFitsViewport(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", nil)
	model.Width = 80
	model.Height = 4

	view := model.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4; view = %q", len(lines), view)
	}
	if !strings.Contains(lines[0], "repo") || !strings.Contains(lines[0], "watching") {
		t.Fatalf("first line = %q, want empty-state header", lines[0])
	}
	if !strings.Contains(lines[3], "q quit") {
		t.Fatalf("last line = %q, want footer", lines[3])
	}
}

// TestModelViewNarrowWidthKeepsChromeSingleLine verifies header and footer stay
// within the viewport width instead of relying on terminal soft-wrap.
func TestModelViewNarrowWidthKeepsChromeSingleLine(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{{
		Path:     "very/long/path/to/current-file-name.ts",
		AddCount: 12,
		DelCount: 3,
	}})
	model.Width = 20
	model.Height = 4

	view := model.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4; view = %q", len(lines), view)
	}
	if lipgloss.Width(lines[0]) > model.Width {
		t.Fatalf("header width = %d, want <= %d; header = %q", lipgloss.Width(lines[0]), model.Width, lines[0])
	}
	if lipgloss.Width(lines[3]) > model.Width {
		t.Fatalf("footer width = %d, want <= %d; footer = %q", lipgloss.Width(lines[3]), model.Width, lines[3])
	}
}

// TestModelOverlayNarrowWidthKeepsEntriesSingleLine verifies overlay rows fit
// the viewport width instead of depending on soft-wrap.
func TestModelOverlayNarrowWidthKeepsEntriesSingleLine(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{
		{Path: "very/long/path/to/current-file-name.ts", AddCount: 12, DelCount: 3},
		{Path: "another/extremely/long/path/to/second-file.ts", AddCount: 1},
	})
	model.Width = 20
	model.Height = 4
	model.OverlayOpen = true
	model.OverlaySnapshot = append([]internal.FileDiff(nil), model.Files...)

	view := model.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4; view = %q", len(lines), view)
	}
	for i := 1; i <= 2; i++ {
		if lipgloss.Width(lines[i]) > model.Width {
			t.Fatalf("overlay line %d width = %d, want <= %d; line = %q", i, lipgloss.Width(lines[i]), model.Width, lines[i])
		}
	}
}

// TestModelScrollOffsetStaysWithinContent verifies repeated down-navigation does
// not grow scroll state beyond the visible diff content.
func TestModelScrollOffsetStaysWithinContent(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "sha", []internal.FileDiff{{
		Path:     "a.txt",
		AddCount: 1,
		Hunks: []internal.DiffHunk{{
			Header: "@@ -0,0 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "hello",
			}},
		}},
	}})
	model.Width = 80
	model.Height = 4

	for range 3 {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model = updated.(Model)
	}

	if model.ScrollOffset != 0 {
		t.Fatalf("ScrollOffset = %d, want 0", model.ScrollOffset)
	}
}
