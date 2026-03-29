package ui

import (
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

// TestHunkFingerprintsIgnoreHeaderChanges verifies header-only shifts keep the same signature.
func TestHunkFingerprintsIgnoreHeaderChanges(t *testing.T) {
	first := hunkFingerprints([]internal.DiffHunk{
		testHunk("@@ -1,1 +1,1 @@", 10, addLine("same content")),
	})
	second := hunkFingerprints([]internal.DiffHunk{
		testHunk("@@ -8,1 +9,1 @@", 99, addLine("same content")),
	})

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected signature lengths: %v %v", first, second)
	}
	if first[0] != second[0] {
		t.Fatalf("header-only change produced different signatures: %d vs %d", first[0], second[0])
	}
}

// TestHunkFingerprintsChangeOnContent verifies line-content edits change the signature.
func TestHunkFingerprintsChangeOnContent(t *testing.T) {
	first := hunkFingerprints([]internal.DiffHunk{
		testHunk("@@ -1,1 +1,1 @@", 10, addLine("before")),
	})
	second := hunkFingerprints([]internal.DiffHunk{
		testHunk("@@ -1,1 +1,1 @@", 10, addLine("after")),
	})

	if first[0] == second[0] {
		t.Fatalf("content change should alter fingerprint, got %d", first[0])
	}
}

// TestLastChangedHunkIndex verifies set-based hunk detection across key cases.
func TestLastChangedHunkIndex(t *testing.T) {
	baseHunks := []internal.DiffHunk{
		testHunk("@@ -1,1 +1,1 @@", 10, addLine("a")),
		testHunk("@@ -2,1 +2,1 @@", 20, addLine("b")),
	}
	baseSigs := hunkFingerprints(baseHunks)

	tests := []struct {
		name     string
		oldSigs  []uint64
		newHunks []internal.DiffHunk
		want     int
	}{
		{
			name:    "no change",
			oldSigs: baseSigs,
			newHunks: []internal.DiffHunk{
				testHunk("@@ -10,1 +10,1 @@", 100, addLine("a")),
				testHunk("@@ -20,1 +20,1 @@", 200, addLine("b")),
			},
			want: -1,
		},
		{
			name:    "content modification",
			oldSigs: baseSigs,
			newHunks: []internal.DiffHunk{
				testHunk("@@ -1,1 +1,1 @@", 10, addLine("a")),
				testHunk("@@ -2,1 +2,1 @@", 20, addLine("changed")),
			},
			want: 1,
		},
		{
			name:    "hunk insertion",
			oldSigs: baseSigs,
			newHunks: []internal.DiffHunk{
				testHunk("@@ -1,1 +1,1 @@", 10, addLine("a")),
				testHunk("@@ -3,1 +3,1 @@", 15, addLine("inserted")),
				testHunk("@@ -2,1 +2,1 @@", 20, addLine("b")),
			},
			want: 1,
		},
		{
			name: "hunk removal",
			oldSigs: hunkFingerprints([]internal.DiffHunk{
				testHunk("@@ -1,1 +1,1 @@", 10, addLine("a")),
				testHunk("@@ -2,1 +2,1 @@", 20, addLine("b")),
				testHunk("@@ -3,1 +3,1 @@", 30, addLine("c")),
			}),
			newHunks: []internal.DiffHunk{
				testHunk("@@ -1,1 +1,1 @@", 10, addLine("a")),
				testHunk("@@ -3,1 +3,1 @@", 30, addLine("c")),
			},
			want: -1,
		},
		{
			name:    "new file",
			oldSigs: nil,
			newHunks: []internal.DiffHunk{
				testHunk("@@ -1,1 +1,1 @@", 10, addLine("a")),
			},
			want: -1,
		},
		{
			name: "duplicate hunk content",
			oldSigs: hunkFingerprints([]internal.DiffHunk{
				testHunk("@@ -1,1 +1,1 @@", 10, addLine("same")),
			}),
			newHunks: []internal.DiffHunk{
				testHunk("@@ -1,1 +1,1 @@", 10, addLine("same")),
				testHunk("@@ -2,1 +2,1 @@", 20, addLine("same")),
			},
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastChangedHunkIndex(tt.oldSigs, tt.newHunks)
			if got != tt.want {
				t.Fatalf("lastChangedHunkIndex() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestHunkVisualOffsetCountsWrappedLines verifies wrapped lines count toward the scroll target.
func TestHunkVisualOffsetCountsWrappedLines(t *testing.T) {
	file := &internal.FileDiff{
		Path: "a.txt",
		Hunks: []internal.DiffHunk{
			testHunk("@@ -1,1 +1,1 @@", 10, addLine("abcdefghi")),
			testHunk("@@ -2,1 +2,1 @@", 20, addLine("z")),
		},
	}

	got := hunkVisualOffset(file, 1, 6)
	if got != 3 {
		t.Fatalf("hunkVisualOffset() = %d, want 3", got)
	}
}
