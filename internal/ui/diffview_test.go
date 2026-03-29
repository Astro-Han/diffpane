package ui

import (
	"strings"
	"testing"

	"github.com/Astro-Han/diffpane/internal"
	"github.com/charmbracelet/lipgloss"
)

// TestWrapLineShort verifies lines shorter than the viewport stay untouched.
func TestWrapLineShort(t *testing.T) {
	result := wrapLine("+hello", 40)
	if result != "+hello" {
		t.Fatalf("wrapLine returned %q, want +hello", result)
	}
}

// TestWrapLineLong verifies continuation lines use indentation instead of a diff prefix.
func TestWrapLineLong(t *testing.T) {
	line := "+" + strings.Repeat("a", 50)
	result := wrapLine(line, 30)
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Fatal("expected wrapped output")
	}
	if !strings.HasPrefix(lines[1], "  ") {
		t.Fatalf("continuation = %q, want 2-space indent", lines[1])
	}
}

// TestWrapLineCJK verifies wrapping never cuts through a rune.
func TestWrapLineCJK(t *testing.T) {
	line := "+你好世界测试"
	result := wrapLine(line, 8)
	for _, r := range result {
		if r == '\uFFFD' {
			t.Fatal("wrapLine cut a rune and produced replacement character")
		}
	}
}

// TestRenderDiffViewCountsWrappedDisplayLines verifies scroll and height work on
// visual lines after wrapping, not just on raw diff entries.
func TestRenderDiffViewCountsWrappedDisplayLines(t *testing.T) {
	file := &internal.FileDiff{
		Path: "long.txt",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -0,0 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: strings.Repeat("a", 12),
			}},
		}},
	}

	firstPage := RenderDiffView(file, 0, 8, 2)
	firstLines := strings.Split(firstPage, "\n")
	if len(firstLines) != 2 {
		t.Fatalf("first page line count = %d, want 2", len(firstLines))
	}

	secondPage := RenderDiffView(file, 1, 8, 2)
	secondLines := strings.Split(secondPage, "\n")
	if len(secondLines) != 2 {
		t.Fatalf("second page line count = %d, want 2", len(secondLines))
	}
	if !strings.HasPrefix(secondLines[1], "  ") {
		t.Fatalf("second page should include wrapped continuation, got %q", secondLines[1])
	}

	thirdPage := RenderDiffView(file, 2, 8, 2)
	thirdLines := strings.Split(thirdPage, "\n")
	if !strings.HasPrefix(thirdLines[0], "  ") {
		t.Fatalf("third page should start with wrapped continuation, got %q", thirdLines[0])
	}
}

func TestSeparatorLineWithLineNumber(t *testing.T) {
	file := &internal.FileDiff{
		Path: "test.go",
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -10,3 +38,5 @@",
			StartLine: 38,
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "hello",
			}},
		}},
	}

	lines := diffDisplayLines(file, 40)
	separator := lines[0]
	if !strings.Contains(separator, "L38") {
		t.Fatalf("separator should contain L38, got %q", separator)
	}
	if !strings.Contains(separator, "──") {
		t.Fatalf("separator should contain ── chars, got %q", separator)
	}
}

func TestSeparatorLineNewFile(t *testing.T) {
	file := &internal.FileDiff{
		Path:   "new.go",
		Status: internal.StatusAdded,
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -0,0 +1,2 @@",
			StartLine: 1,
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "hello",
			}},
		}},
	}

	lines := diffDisplayLines(file, 40)
	separator := lines[0]
	if strings.Contains(separator, "L1") {
		t.Fatalf("new file separator should not contain L1, got %q", separator)
	}
	if !strings.Contains(separator, "──") {
		t.Fatalf("separator should contain ── chars, got %q", separator)
	}
}

func TestSeparatorLineDeletedFile(t *testing.T) {
	file := &internal.FileDiff{
		Path:   "old.go",
		Status: internal.StatusDeleted,
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -1,2 +0,0 @@",
			StartLine: 0,
			Lines: []internal.DiffLine{{
				Type:    internal.LineDel,
				Content: "goodbye",
			}},
		}},
	}

	lines := diffDisplayLines(file, 40)
	separator := lines[0]
	if strings.Contains(separator, "L0") {
		t.Fatalf("deleted file separator should not contain L0, got %q", separator)
	}
	if !strings.Contains(separator, "──") {
		t.Fatalf("separator should contain ── chars, got %q", separator)
	}
}

func TestSeparatorLineNarrowWidth(t *testing.T) {
	file := &internal.FileDiff{
		Path: "test.go",
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -1,3 +99999,5 @@",
			StartLine: 99999,
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "x",
			}},
		}},
	}

	width := 10
	lines := diffDisplayLines(file, width)
	if len(lines) < 1 {
		t.Fatal("expected at least 1 line")
	}
	if separatorWidth := lipgloss.Width(lines[0]); separatorWidth > width {
		t.Fatalf("separator width = %d, want <= %d", separatorWidth, width)
	}
}

func TestSeparatorLineMultiHunk(t *testing.T) {
	file := &internal.FileDiff{
		Path: "multi.go",
		Hunks: []internal.DiffHunk{
			{
				Header:    "@@ -1,3 +10,5 @@",
				StartLine: 10,
				Lines: []internal.DiffLine{{
					Type:    internal.LineAdd,
					Content: "first",
				}},
			},
			{
				Header:    "@@ -20,3 +150,5 @@",
				StartLine: 150,
				Lines: []internal.DiffLine{{
					Type:    internal.LineAdd,
					Content: "second",
				}},
			},
		},
	}

	lines := diffDisplayLines(file, 40)
	separatorCount := 0
	for _, line := range lines {
		if strings.Contains(line, "──") {
			separatorCount++
		}
	}
	if separatorCount != 2 {
		t.Fatalf("expected 2 separator lines, got %d", separatorCount)
	}
}
