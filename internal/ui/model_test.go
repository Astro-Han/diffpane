package ui

import (
	"testing"

	"github.com/Astro-Han/diffpane/internal"
	tea "github.com/charmbracelet/bubbletea"
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

// TestModelDropsQueuedOverlayUpdateAfterBaselineReset verifies a queued update
// from the old baseline cannot overwrite the reset state when overlay closes.
func TestModelDropsQueuedOverlayUpdateAfterBaselineReset(t *testing.T) {
	model := NewModel("repo", "/tmp/repo", "old-sha", []internal.FileDiff{
		file("old.txt", 1),
	})

	opened, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	withOverlay := opened.(Model)

	queued, _ := withOverlay.Update(FilesUpdatedMsg{
		BaselineSHA: "old-sha",
		Files: []internal.FileDiff{
			file("stale.txt", 1),
		},
		ChangedPaths: []string{"stale.txt"},
	})
	withPending := queued.(Model)
	if withPending.PendingUpdate == nil {
		t.Fatal("expected pending update")
	}

	reset, _ := withPending.Update(BaselineResetMsg{NewSHA: "new-sha"})
	closed, _ := reset.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := closed.(Model)
	if len(got.Files) != 1 || got.Files[0].Path != "old.txt" {
		t.Fatalf("stale overlay update should be dropped, got %#v", got.Files)
	}
}
