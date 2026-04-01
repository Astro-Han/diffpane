package ui

import (
	"reflect"
	"testing"

	"github.com/Astro-Han/diffpane/internal"
)

// testHunk builds a small hunk for follow helper tests.
func testHunk(header string, startLine int, lines ...internal.DiffLine) internal.DiffHunk {
	return internal.DiffHunk{
		Header:    header,
		StartLine: startLine,
		Lines:     lines,
	}
}

// addLine builds one added diff line for helper tests.
func addLine(content string) internal.DiffLine {
	return internal.DiffLine{
		Type:    internal.LineAdd,
		Content: content,
	}
}

// delLine builds one deleted diff line for helper tests.
func delLine(content string) internal.DiffLine {
	return internal.DiffLine{
		Type:    internal.LineDel,
		Content: content,
	}
}

// ctxLine builds one context diff line for helper tests.
func ctxLine(content string) internal.DiffLine {
	return internal.DiffLine{
		Type:    internal.LineContext,
		Content: content,
	}
}

// TestLineFingerprintsSkipContextAndHeader verifies that only add and delete
// lines affect the snapshot, not hunk headers or context lines.
func TestLineFingerprintsSkipContextAndHeader(t *testing.T) {
	first := lineFingerprints([]internal.DiffHunk{
		testHunk("@@ -1,3 +1,3 @@", 10,
			ctxLine("shared context"),
			addLine("shared add"),
			delLine("shared del"),
		),
	})
	second := lineFingerprints([]internal.DiffHunk{
		testHunk("@@ -9,3 +9,3 @@", 99,
			ctxLine("different context"),
			addLine("shared add"),
			delLine("shared del"),
		),
	})

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("lineFingerprints() = %#v, want %#v", first, second)
	}
}

// TestChangedLineKeys verifies multiset-based line detection across a few
// representative change shapes.
func TestChangedLineKeys(t *testing.T) {
	baseHunks := []internal.DiffHunk{
		testHunk("@@ -1,2 +1,2 @@", 10, addLine("same")),
		testHunk("@@ -2,2 +2,2 @@", 20, addLine("dup")),
	}
	baseSigs := lineFingerprints(baseHunks)

	tests := []struct {
		name     string
		oldSigs  []uint64
		newHunks []internal.DiffHunk
		want     map[lineKey]bool
	}{
		{
			name:    "no change",
			oldSigs: baseSigs,
			newHunks: []internal.DiffHunk{
				testHunk("@@ -10,2 +10,2 @@", 100, ctxLine("ignored"), addLine("same")),
				testHunk("@@ -20,2 +20,2 @@", 200, addLine("dup")),
			},
			want: map[lineKey]bool{},
		},
		{
			name:    "new duplicate only marks the extra line",
			oldSigs: baseSigs,
			newHunks: []internal.DiffHunk{
				testHunk("@@ -10,2 +10,2 @@", 100, addLine("same")),
				testHunk("@@ -20,2 +20,2 @@", 200, addLine("dup")),
				testHunk("@@ -30,2 +30,2 @@", 300, addLine("dup")),
			},
			want: map[lineKey]bool{
				{HunkIdx: 2, LineIdx: 0}: true,
			},
		},
		{
			name:    "changed content is flagged at its line key",
			oldSigs: baseSigs,
			newHunks: []internal.DiffHunk{
				testHunk("@@ -10,2 +10,2 @@", 100, addLine("same")),
				testHunk("@@ -20,2 +20,2 @@", 200, addLine("changed")),
			},
			want: map[lineKey]bool{
				{HunkIdx: 1, LineIdx: 0}: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := changedLineKeys(tt.oldSigs, tt.newHunks)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("changedLineKeys() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestHunkVisualOffsetTargetsHighlightedLine verifies the scroll offset lands
// on the target line inside the hunk, counting wrapped visual rows.
func TestHunkVisualOffsetTargetsHighlightedLine(t *testing.T) {
	file := &internal.FileDiff{
		Path: "a.txt",
		Hunks: []internal.DiffHunk{
			testHunk("@@ -1,2 +1,2 @@", 10,
				addLine("abcdefghi"),
				addLine("z"),
			),
		},
	}

	got := hunkVisualOffset(file, 1, 6)
	if got != 3 {
		t.Fatalf("hunkVisualOffset() = %d, want 3", got)
	}
}
