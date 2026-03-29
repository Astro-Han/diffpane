package ui

import (
	"strings"
	"testing"

	"github.com/Astro-Han/diffpane/internal"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// terminalRows reports how many physical terminal rows one rendered line uses
// after stripping ANSI codes and expanding tabs to the default 8-column stops.
func terminalRows(line string, width int) int {
	if width <= 0 {
		return 0
	}

	rows := 1
	column := 0

	for _, r := range ansi.Strip(line) {
		cellWidth := runewidth.RuneWidth(r)
		if r == '\t' {
			cellWidth = 8 - (column % 8)
			if cellWidth == 0 {
				cellWidth = 8
			}
		}
		if cellWidth <= 0 {
			continue
		}
		if column > 0 && column+cellWidth > width {
			rows++
			column = 0
			if r == '\t' {
				cellWidth = 8
			}
		}
		column += cellWidth
	}

	return rows
}

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
		if strings.Contains(ansi.Strip(line), "──") {
			separatorCount++
		}
	}
	if separatorCount != 2 {
		t.Fatalf("expected 2 separator lines, got %d", separatorCount)
	}
}

// TestDiffDisplayLinesTabIndentedLineFitsViewport verifies each returned visual
// line still occupies exactly one terminal row when diff content contains tabs.
func TestDiffDisplayLinesTabIndentedLineFitsViewport(t *testing.T) {
	width := 24
	file := &internal.FileDiff{
		Path: "test.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "\t\t// fileWithHunks builds a FileDiff with textual hunks",
			}},
		}},
	}

	lines := diffDisplayLines(file, width)
	for i, line := range lines {
		if rows := terminalRows(line, width); rows != 1 {
			t.Fatalf("line %d occupies %d terminal rows, want 1: %q", i, rows, ansi.Strip(line))
		}
	}
}

// TestDiffDisplayLinesTabAndCJKLineFitsViewport verifies tab expansion also
// stays aligned when the diff line mixes tabs with double-width runes.
func TestDiffDisplayLinesTabAndCJKLineFitsViewport(t *testing.T) {
	width := 16
	file := &internal.FileDiff{
		Path: "test.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "\t中文注释\tmixed content",
			}},
		}},
	}

	lines := diffDisplayLines(file, width)
	if len(lines) < 3 {
		t.Fatalf("line count = %d, want at least 3 to exercise wrapped tab+CJK rendering", len(lines))
	}
	for i, line := range lines {
		if rows := terminalRows(line, width); rows != 1 {
			t.Fatalf("line %d occupies %d terminal rows, want 1: %q", i, rows, ansi.Strip(line))
		}
	}
}

// TestDiffDisplayLinesTabLineFitsNarrowViewport verifies tab-expanded content
// still wraps into logical segments when the viewport is extremely narrow.
func TestDiffDisplayLinesTabLineFitsNarrowViewport(t *testing.T) {
	width := 8
	file := &internal.FileDiff{
		Path: "test.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "\t\tabcdefghijk",
			}},
		}},
	}

	lines := diffDisplayLines(file, width)
	if len(lines) < 5 {
		t.Fatalf("line count = %d, want at least 5 to exercise narrow multi-segment wrapping", len(lines))
	}
	for i, line := range lines {
		if rows := terminalRows(line, width); rows != 1 {
			t.Fatalf("line %d occupies %d terminal rows, want 1: %q", i, rows, ansi.Strip(line))
		}
	}
}
