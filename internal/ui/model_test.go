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
