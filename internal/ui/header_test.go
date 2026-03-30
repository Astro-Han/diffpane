package ui

import (
	"strings"
	"testing"

	"github.com/Astro-Han/diffpane/internal"
)

// TestRenderHeaderEmpty verifies the empty state header shows directory and watching state.
func TestRenderHeaderEmpty(t *testing.T) {
	header := RenderHeader("myproject", nil, 0, 80)
	if !strings.Contains(header, "myproject") || !strings.Contains(header, "watching") {
		t.Fatalf("header = %q, want dir name and watching", header)
	}
}

// TestRenderHeaderSingleFile verifies one file does not show the file counter.
func TestRenderHeaderSingleFile(t *testing.T) {
	files := []internal.FileDiff{{Path: "src/auth.ts", AddCount: 3, DelCount: 1}}
	header := RenderHeader("myproject", files, 0, 80)
	if strings.Contains(header, "1/1") {
		t.Fatalf("single file header should not show counter, got %q", header)
	}
}

// TestRenderHeaderMultipleFiles verifies multiple files show the current position.
func TestRenderHeaderMultipleFiles(t *testing.T) {
	files := []internal.FileDiff{
		{Path: "src/auth.ts", AddCount: 3},
		{Path: "src/utils.ts", AddCount: 1},
	}
	header := RenderHeader("myproject", files, 0, 80)
	if !strings.Contains(header, "1/2") {
		t.Fatalf("header = %q, want counter 1/2", header)
	}
}

// TestRenderHeaderDoesNotShowPendingNewCount verifies the old paused-follow
// indicator is removed from the top bar.
func TestRenderHeaderDoesNotShowPendingNewCount(t *testing.T) {
	files := []internal.FileDiff{{Path: "a.ts", AddCount: 1}}
	header := RenderHeader("myproject", files, 0, 80)
	if strings.Contains(header, "new") {
		t.Fatalf("header = %q, want no pending-new indicator", header)
	}
}

// TestRenderHeaderOutOfRangeIndex verifies defensive index clamping avoids panics.
func TestRenderHeaderOutOfRangeIndex(t *testing.T) {
	files := []internal.FileDiff{{Path: "a.ts", AddCount: 1}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RenderHeader panicked for out-of-range index: %v", r)
		}
	}()

	header := RenderHeader("myproject", files, 9, 80)
	if !strings.Contains(header, "a.ts") {
		t.Fatalf("header = %q, want file path", header)
	}
}
