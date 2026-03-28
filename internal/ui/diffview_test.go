package ui

import (
	"strings"
	"testing"

	"github.com/Astro-Han/diffpane/internal"
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
