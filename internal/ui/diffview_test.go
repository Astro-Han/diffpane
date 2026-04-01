package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Astro-Han/diffpane/internal"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
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
	result := wrapLine("hello", 40)
	if result != "hello" {
		t.Fatalf("wrapLine returned %q, want hello", result)
	}
}

// TestWrapLineLong verifies the pure-content wrapper does not add continuation
// prefixes on its own.
func TestWrapLineLong(t *testing.T) {
	line := strings.Repeat("a", 50)
	result := wrapLine(line, 30)
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Fatal("expected wrapped output")
	}
	if strings.HasPrefix(lines[1], "  ") {
		t.Fatalf("continuation = %q, should not include an embedded indent", lines[1])
	}
}

// TestWrapLineCJK verifies wrapping never cuts through a rune.
func TestWrapLineCJK(t *testing.T) {
	line := "你好世界测试"
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

	firstPage := RenderDiffView(file, 0, 8, 2, nil)
	firstLines := strings.Split(firstPage, "\n")
	if len(firstLines) != 2 {
		t.Fatalf("first page line count = %d, want 2", len(firstLines))
	}

	secondPage := RenderDiffView(file, 1, 8, 2, nil)
	secondLines := strings.Split(secondPage, "\n")
	if len(secondLines) != 2 {
		t.Fatalf("second page line count = %d, want 2", len(secondLines))
	}
	if !strings.HasPrefix(secondLines[1], "↳") {
		t.Fatalf("second page should include wrapped continuation, got %q", secondLines[1])
	}

	thirdPage := RenderDiffView(file, 2, 8, 2, nil)
	thirdLines := strings.Split(thirdPage, "\n")
	if !strings.HasPrefix(thirdLines[0], "↳") {
		t.Fatalf("third page should start with wrapped continuation, got %q", thirdLines[0])
	}
}

func TestRenderSeparatorIsAlwaysPlainDashes(t *testing.T) {
	got := ansi.Strip(renderSeparator(24))
	if strings.Contains(got, "L24") {
		t.Fatalf("separator should not include a line-number label, got %q", got)
	}
	if got != strings.Repeat("─", 24) {
		t.Fatalf("separator = %q, want 24 plain dashes", got)
	}
}

func TestSeparatorLineNarrowWidth(t *testing.T) {
	got := ansi.Strip(renderSeparator(0))
	if got != "─" {
		t.Fatalf("separator for width 0 = %q, want one dash", got)
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

	lines := diffDisplayLines(file, 40, nil)
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

	lines := diffDisplayLines(file, width, nil)
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

	lines := diffDisplayLines(file, width, nil)
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

	lines := diffDisplayLines(file, width, nil)
	if len(lines) < 5 {
		t.Fatalf("line count = %d, want at least 5 to exercise narrow multi-segment wrapping", len(lines))
	}
	for i, line := range lines {
		if rows := terminalRows(line, width); rows != 1 {
			t.Fatalf("line %d occupies %d terminal rows, want 1: %q", i, rows, ansi.Strip(line))
		}
	}
}

// TestDiffDisplayLinesGoFileHighlighted verifies Go diffs get syntax colors
// in addition to the diff prefix styling.
func TestDiffDisplayLinesGoFileHighlighted(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.ANSI256 }
	defer func() { colorProfileFn = prev }()

	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -0,0 +1,1 @@",
			StartLine: 1,
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "func main() {",
			}},
		}},
	}

	lines := diffDisplayLines(file, 80, nil)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (separator + code), got %d", len(lines))
	}

	codeLine := lines[1]
	stripped := ansi.Strip(codeLine)
	if !strings.Contains(stripped, "+func main() {") {
		t.Fatalf("stripped = %q, should contain '+func main() {'", stripped)
	}
	if len(codeLine) <= len(stripped) {
		t.Fatalf("expected ANSI-highlighted output (len %d <= stripped len %d)", len(codeLine), len(stripped))
	}
}

// TestDiffDisplayLinesPlaintextNoHighlight verifies unknown file types keep
// plain code content without extra chroma ANSI styling.
func TestDiffDisplayLinesPlaintextNoHighlight(t *testing.T) {
	file := &internal.FileDiff{
		Path: "data.randomext123",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -0,0 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:    internal.LineContext,
				Content: "just plain text",
			}},
		}},
	}

	lines := diffDisplayLines(file, 80, nil)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	codeLine := lines[1]
	stripped := ansi.Strip(codeLine)
	if codeLine != stripped {
		t.Fatalf("unknown file context line should be plain text, got %q", codeLine)
	}
}

// TestDiffDisplayLinesWrappedContinuationHighlighted verifies wrapped code
// segments keep continuation prefixes while still getting syntax colors.
func TestDiffDisplayLinesWrappedContinuationHighlighted(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.ANSI256 }
	defer func() { colorProfileFn = prev }()

	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -0,0 +1,1 @@",
			StartLine: 1,
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "func veryLongFunctionName(parameterOne int, parameterTwo string) error {",
			}},
		}},
	}

	lines := diffDisplayLines(file, 30, nil)
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (separator + first + continuation), got %d", len(lines))
	}

	first := lines[1]
	firstStripped := ansi.Strip(first)
	if !strings.HasPrefix(firstStripped, "+") {
		t.Fatalf("first segment stripped = %q, want '+' prefix", firstStripped)
	}
	if len(first) <= len(firstStripped) {
		t.Fatalf("first segment should contain ANSI codes")
	}

	continuation := lines[2]
	continuationStripped := ansi.Strip(continuation)
	if !strings.HasPrefix(continuationStripped, "↳") {
		t.Fatalf("continuation stripped = %q, want '↳' prefix", continuationStripped)
	}
	if len(continuation) <= len(continuationStripped) {
		t.Fatalf("continuation segment should contain ANSI codes")
	}
}

// TestCountWrappedDiffLinesMatchesRenderedLines verifies the lightweight line
// counter stays in sync with actual rendered output for highlighted diffs.
func TestCountWrappedDiffLinesMatchesRenderedLines(t *testing.T) {
	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -0,0 +1,2 @@",
			StartLine: 1,
			Lines: []internal.DiffLine{
				{
					Type:    internal.LineAdd,
					Content: "func main() {",
				},
				{
					Type:    internal.LineAdd,
					Content: "fmt.Println(\"a very long string that should wrap across lines\")",
				},
			},
		}},
	}

	width := 24
	rendered := diffDisplayLines(file, width, nil)
	counted := countWrappedDiffLines(file, width)
	if counted != len(rendered) {
		t.Fatalf("counted lines = %d, want %d", counted, len(rendered))
	}
}

// TestDisplayLineCacheReusesBuilderOutput verifies repeated requests for the
// same file signature and width reuse cached visual lines.
func TestDisplayLineCacheReusesBuilderOutput(t *testing.T) {
	cache := newDisplayLineCache()
	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -0,0 +1,1 @@",
			StartLine: 1,
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "func main() {",
			}},
		}},
	}

	builds := 0
	first := cache.get(file, 80, nil, func() []string {
		builds++
		return []string{"first"}
	})
	second := cache.get(file, 80, nil, func() []string {
		builds++
		return []string{"second"}
	})

	if builds != 1 {
		t.Fatalf("builds = %d, want 1", builds)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("cached lines = %#v, want %#v", second, first)
	}
}

// TestDisplayLineCacheMissesWhenHighlightStateChanges verifies highlight-only
// visual changes invalidate the one-entry render cache.
func TestDisplayLineCacheMissesWhenHighlightStateChanges(t *testing.T) {
	cache := newDisplayLineCache()
	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:    "@@ -0,0 +1,1 @@",
			StartLine: 1,
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "func main() {}",
			}},
		}},
	}

	builds := 0
	cache.get(file, 80, map[lineKey]bool{{HunkIdx: 0, LineIdx: 0}: true}, func() []string {
		builds++
		return []string{"first"}
	})
	cache.get(file, 80, nil, func() []string {
		builds++
		return []string{"second"}
	})

	if builds != 2 {
		t.Fatalf("builds = %d, want 2 when highlight state changes", builds)
	}
}

// TestDisplayLineCacheKeyChangesWhenLineNumbersShift verifies the cache key
// changes when line-number metadata changes but line content stays the same.
func TestDisplayLineCacheKeyChangesWhenLineNumbersShift(t *testing.T) {
	base := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:       "@@ -1,1 +1,1 @@",
			OldStartLine: 1,
			StartLine:    1,
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				Content:   "func main() {}",
				NewLineNo: 1,
			}},
		}},
	}
	shifted := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:       "@@ -9,1 +9,1 @@",
			OldStartLine: 9,
			StartLine:    9,
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				Content:   "func main() {}",
				NewLineNo: 9,
			}},
		}},
	}

	if newDisplayLineCacheKey(base, 80, nil) == newDisplayLineCacheKey(shifted, 80, nil) {
		t.Fatal("cache key should change when displayed line numbers shift")
	}
}

// TestGutterWidthThresholds verifies the renderer switches between full and
// compact gutter modes at the spec's width boundary.
func TestGutterWidthThresholds(t *testing.T) {
	file := &internal.FileDiff{
		Hunks: []internal.DiffHunk{{
			StartLine: 42,
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				NewLineNo: 42,
			}},
		}},
	}

	if got := gutterWidth(file, 60); got != 6 {
		t.Fatalf("gutterWidth(60) = %d, want 6", got)
	}
	if got := gutterWidth(file, 40); got != 6 {
		t.Fatalf("gutterWidth(40) = %d, want 6", got)
	}
	if got := gutterWidth(file, 39); got != 1 {
		t.Fatalf("gutterWidth(39) = %d, want 1", got)
	}
}

// TestLineNoWidthUsesOldAndNewMaximums verifies gutter sizing looks at both
// old-side and new-side line number ranges.
func TestLineNoWidthUsesOldAndNewMaximums(t *testing.T) {
	file := &internal.FileDiff{
		Hunks: []internal.DiffHunk{{
			OldStartLine: 9997,
			StartLine:    40,
			Lines: []internal.DiffLine{
				{Type: internal.LineDel},
				{Type: internal.LineDel},
				{Type: internal.LineContext},
			},
		}},
	}

	if got := lineNoWidth(file); got != 4 {
		t.Fatalf("lineNoWidth() = %d, want 4", got)
	}
}

// TestDiffDisplayLinesFullGutterShowsLineNumbersAndContinuationMarker verifies
// wrapped lines show the numbered gutter once and use ↳ on continuation rows.
func TestDiffDisplayLinesFullGutterShowsLineNumbersAndContinuationMarker(t *testing.T) {
	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:       "@@ -10,1 +20,1 @@",
			OldStartLine: 10,
			StartLine:    20,
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				Content:   strings.Repeat("a", 70),
				NewLineNo: 20,
			}},
		}},
	}

	lines := diffDisplayLines(file, 40, nil)
	first := ansi.Strip(lines[1])
	second := ansi.Strip(lines[2])

	if !strings.HasPrefix(first, "  20 +") {
		t.Fatalf("first line = %q, want line number then prefix", first)
	}
	if !strings.HasPrefix(second, "     ↳") {
		t.Fatalf("continuation line = %q, want blank line number field plus ↳", second)
	}
}

// TestDiffDisplayLinesDeletedLineUsesOldLineNumber verifies deleted rows render
// the old-side line number instead of the new-side one.
func TestDiffDisplayLinesDeletedLineUsesOldLineNumber(t *testing.T) {
	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header:       "@@ -42,1 +0,0 @@",
			OldStartLine: 42,
			Lines: []internal.DiffLine{{
				Type:      internal.LineDel,
				Content:   "old line",
				OldLineNo: 42,
			}},
		}},
	}

	lines := diffDisplayLines(file, 60, nil)
	if stripped := ansi.Strip(lines[1]); !strings.HasPrefix(stripped, "  42 -") {
		t.Fatalf("deleted line gutter = %q, want old-side line number", stripped)
	}
}

// TestDiffDisplayLinesMalformedHunkLeavesBlankLineNumber verifies malformed
// hunks render spaces instead of a literal zero.
func TestDiffDisplayLinesMalformedHunkLeavesBlankLineNumber(t *testing.T) {
	file := &internal.FileDiff{
		Path: "broken.txt",
		Hunks: []internal.DiffHunk{{
			Header: "@@ broken header @@",
			Lines: []internal.DiffLine{{
				Type:    internal.LineAdd,
				Content: "added line",
			}},
		}},
	}

	lines := diffDisplayLines(file, 60, nil)
	stripped := ansi.Strip(lines[1])
	if !strings.HasPrefix(stripped, "     +") {
		t.Fatalf("malformed hunk gutter = %q, want blank line number field", stripped)
	}
	if strings.Contains(stripped[:6], "0") {
		t.Fatalf("malformed hunk gutter should not render zero digits, got %q", stripped)
	}
}

// TestDiffDisplayLinesCompactGutterOmitsLineNumbers verifies narrow terminals
// fall back to prefix-only gutter mode.
func TestDiffDisplayLinesCompactGutterOmitsLineNumbers(t *testing.T) {
	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -1,1 +7,1 @@",
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				Content:   "abc",
				NewLineNo: 7,
			}},
		}},
	}

	lines := diffDisplayLines(file, 39, nil)
	if stripped := ansi.Strip(lines[1]); stripped != "+abc" {
		t.Fatalf("compact gutter line = %q, want prefix only", stripped)
	}
}

// TestDiffDisplayLinesCompactContinuationUsesArrow verifies compact mode still
// marks wrapped continuation rows explicitly.
func TestDiffDisplayLinesCompactContinuationUsesArrow(t *testing.T) {
	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				Content:   strings.Repeat("a", 50),
				NewLineNo: 1,
			}},
		}},
	}

	lines := diffDisplayLines(file, 39, nil)
	if stripped := ansi.Strip(lines[2]); !strings.HasPrefix(stripped, "↳") {
		t.Fatalf("compact continuation = %q, want ↳ prefix", stripped)
	}
}

// TestCountWrappedDiffLinesMatchesRenderedLinesAfterGutterRefactor verifies the
// lightweight row counter stays aligned with rendered output after the layout change.
func TestCountWrappedDiffLinesMatchesRenderedLinesAfterGutterRefactor(t *testing.T) {
	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -1,1 +1,2 @@",
			Lines: []internal.DiffLine{
				{Type: internal.LineAdd, Content: strings.Repeat("a", 35), NewLineNo: 1},
				{Type: internal.LineDel, Content: strings.Repeat("b", 35), OldLineNo: 1},
			},
		}},
	}

	width := 32
	if got, want := countWrappedDiffLines(file, width), len(diffDisplayLines(file, width, nil)); got != want {
		t.Fatalf("countWrappedDiffLines() = %d, want %d", got, want)
	}
}

// TestDiffDisplayLinesTrueColorPadsBackgroundAcrossViewport verifies add lines
// paint a full-width background in true-color terminals.
func TestDiffDisplayLinesTrueColorPadsBackgroundAcrossViewport(t *testing.T) {
	prevProfile := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.TrueColor }
	defer func() { colorProfileFn = prevProfile }()
	defer setThemeForTest(ThemeDark)()

	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -0,0 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				Content:   "func main() {",
				NewLineNo: 1,
			}},
		}},
	}

	line := diffDisplayLines(file, 30, map[lineKey]bool{{HunkIdx: 0, LineIdx: 0}: true})[1]
	if !strings.Contains(line, "\033[48;2;") {
		t.Fatalf("true-color add line should contain background ANSI, got %q", line)
	}
	if lipgloss.Width(line) != 30 {
		t.Fatalf("rendered width = %d, want 30", lipgloss.Width(line))
	}
}

// TestDiffDisplayLinesHighlightsOnlyMarkedLine verifies line-level highlight
// state does not spill across other lines in the same hunk.
func TestDiffDisplayLinesHighlightsOnlyMarkedLine(t *testing.T) {
	prevProfile := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.TrueColor }
	defer func() { colorProfileFn = prevProfile }()
	defer setThemeForTest(ThemeDark)()

	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -0,0 +1,2 @@",
			Lines: []internal.DiffLine{
				{
					Type:      internal.LineAdd,
					Content:   "first",
					NewLineNo: 1,
				},
				{
					Type:      internal.LineAdd,
					Content:   "second",
					NewLineNo: 2,
				},
			},
		}},
	}

	lines := diffDisplayLines(file, 30, map[lineKey]bool{{HunkIdx: 0, LineIdx: 1}: true})
	if strings.Contains(lines[1], "\033[48;2;") {
		t.Fatalf("first line should not contain background ANSI, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "\033[48;2;") {
		t.Fatalf("second line should contain background ANSI, got %q", lines[2])
	}
}

// TestRenderDiffSegmentTrueColorNonHighlightedForcesPrefixColor verifies
// true-color terminals still color add/delete prefixes when backgrounds are off.
func TestRenderDiffSegmentTrueColorNonHighlightedForcesPrefixColor(t *testing.T) {
	prevProfile := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.TrueColor }
	defer func() { colorProfileFn = prevProfile }()
	defer setThemeForTest(ThemeDark)()

	line := renderDiffSegment("", "+", "plain content", internal.LineAdd, 30, "notes.txt", false)
	if strings.Contains(line, "\033[48;2;") {
		t.Fatalf("non-highlighted line should not contain background ANSI, got %q", line)
	}
	if !strings.Contains(line, "\033[") {
		t.Fatalf("non-highlighted line should contain prefix ANSI, got %q", line)
	}
	if got := ansi.Strip(line); got != "+plain content" {
		t.Fatalf("rendered line = %q, want +plain content", got)
	}
}

// TestDiffDisplayLinesAnsi256KeepsColoredPrefixWithoutBackground verifies
// non-truecolor terminals stay on the old foreground-only add/delete signal.
func TestDiffDisplayLinesAnsi256KeepsColoredPrefixWithoutBackground(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.ANSI256 }
	defer func() { colorProfileFn = prev }()

	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -0,0 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				Content:   "func main() {",
				NewLineNo: 1,
			}},
		}},
	}

	line := diffDisplayLines(file, 30, nil)[1]
	if strings.Contains(line, "\033[48;2;") {
		t.Fatalf("ANSI256 profile should not contain true-color background, got %q", line)
	}
	if !strings.Contains(line, "\033[") {
		t.Fatalf("ANSI256 profile should still style the prefix, got %q", line)
	}
}

// TestDiffDisplayLinesAnsiKeepsColoredPrefixWithoutBackground verifies
// 16-color terminals stay on the foreground-only add/delete signal as well.
func TestDiffDisplayLinesAnsiKeepsColoredPrefixWithoutBackground(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.ANSI }
	defer func() { colorProfileFn = prev }()

	file := &internal.FileDiff{
		Path: "main.go",
		Hunks: []internal.DiffHunk{{
			Header: "@@ -0,0 +1,1 @@",
			Lines: []internal.DiffLine{{
				Type:      internal.LineAdd,
				Content:   "func main() {",
				NewLineNo: 1,
			}},
		}},
	}

	line := diffDisplayLines(file, 30, nil)[1]
	if strings.Contains(line, "\033[48;2;") {
		t.Fatalf("ANSI profile should not contain true-color background, got %q", line)
	}
	if !strings.Contains(line, "\033[") {
		t.Fatalf("ANSI profile should still style the prefix, got %q", line)
	}
}

// TestRenderSeparatorAsciiProfileHasNoANSI verifies Ascii mode removes ANSI
// styling from separators as well.
func TestRenderSeparatorAsciiProfileHasNoANSI(t *testing.T) {
	prev := colorProfileFn
	colorProfileFn = func() termenv.Profile { return termenv.Ascii }
	defer func() { colorProfileFn = prev }()

	line := renderSeparator(24)
	if line != ansi.Strip(line) {
		t.Fatalf("Ascii separator should not contain ANSI codes, got %q", line)
	}
}
