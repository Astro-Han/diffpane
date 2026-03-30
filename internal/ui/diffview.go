package ui

import (
	"fmt"
	"hash/fnv"
	"strings"
	"sync"

	"github.com/Astro-Han/diffpane/internal"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

const terminalTabStop = 8

type displayLineCacheKey struct {
	path      string
	width     int
	isBinary  bool
	status    internal.FileStatus
	signature uint64
}

// displayLineCache reuses the most recent rendered visual lines for the same
// file content and viewport width so repeated View calls do not re-highlight.
type displayLineCache struct {
	mu    sync.Mutex
	valid bool
	key   displayLineCacheKey
	lines []string
}

// newDisplayLineCache creates an empty one-entry cache for rendered diff lines.
func newDisplayLineCache() *displayLineCache {
	return &displayLineCache{}
}

// get returns cached rendered lines when the file signature and width match.
func (c *displayLineCache) get(file *internal.FileDiff, width int, build func() []string) []string {
	if c == nil {
		return build()
	}

	key := newDisplayLineCacheKey(file, width)

	c.mu.Lock()
	if c.valid && c.key == key {
		lines := c.lines
		c.mu.Unlock()
		return lines
	}
	c.mu.Unlock()

	lines := build()

	c.mu.Lock()
	c.key = key
	c.lines = lines
	c.valid = true
	c.mu.Unlock()

	return lines
}

// newDisplayLineCacheKey fingerprints the rendered inputs that affect visual lines.
func newDisplayLineCacheKey(file *internal.FileDiff, width int) displayLineCacheKey {
	key := displayLineCacheKey{width: width}
	if file == nil {
		return key
	}

	key.path = file.Path
	key.isBinary = file.IsBinary
	key.status = file.Status

	hasher := fnv.New64a()
	for _, hunk := range file.Hunks {
		_, _ = hasher.Write([]byte(fmt.Sprintf("%d\x00%d\x00", hunk.OldStartLine, hunk.StartLine)))
		for _, line := range hunk.Lines {
			_, _ = hasher.Write([]byte(fmt.Sprintf("%d\x00%d\x00%d\x00%s\x00", line.Type, line.OldLineNo, line.NewLineNo, line.Content)))
		}
	}
	key.signature = hasher.Sum64()
	return key
}

// RenderDiffView renders the current file diff within the viewport.
func RenderDiffView(file *internal.FileDiff, scrollOffset, width, height int) string {
	lines := diffDisplayLines(file, width)
	return renderDisplayLines(lines, scrollOffset, height)
}

// renderDisplayLines slices already-rendered visual lines into the visible viewport.
func renderDisplayLines(lines []string, scrollOffset, height int) string {
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset >= len(lines) {
		scrollOffset = max(0, len(lines)-1)
	}

	end := scrollOffset + height
	if end > len(lines) {
		end = len(lines)
	}
	if scrollOffset >= end {
		return ""
	}

	return strings.Join(lines[scrollOffset:end], "\n")
}

// countVisualDiffLines returns how many terminal rows the diff content occupies.
func countVisualDiffLines(file *internal.FileDiff, width int) int {
	return countWrappedDiffLines(file, width)
}

// lineNoWidth returns the width of the line number column for one file.
func lineNoWidth(file *internal.FileDiff) int {
	maxLineNo := 0
	if file == nil {
		return 4
	}

	for _, hunk := range file.Hunks {
		oldCount := 0
		newCount := 0
		for _, line := range hunk.Lines {
			switch line.Type {
			case internal.LineDel, internal.LineContext:
				oldCount++
			}
			switch line.Type {
			case internal.LineAdd, internal.LineContext:
				newCount++
			}
		}

		if hunk.OldStartLine > 0 && oldCount > 0 {
			maxLineNo = max(maxLineNo, hunk.OldStartLine+oldCount-1)
		}
		if hunk.StartLine > 0 && newCount > 0 {
			maxLineNo = max(maxLineNo, hunk.StartLine+newCount-1)
		}
	}

	return max(4, len(fmt.Sprintf("%d", maxLineNo)))
}

// gutterWidth returns the fixed gutter width for the current viewport.
func gutterWidth(file *internal.FileDiff, viewportWidth int) int {
	if viewportWidth < 40 {
		return 1
	}

	return lineNoWidth(file) + 2
}

// renderSeparator builds a fixed-width hunk separator for the current viewport.
func renderSeparator(width int) string {
	const dash = "─"

	count := width
	if count < 1 {
		count = 1
	}

	text := strings.Repeat(dash, count)
	if colorProfileFn() == termenv.Ascii {
		return text
	}

	return StyleDim.Render(text)
}

// diffDisplayLines expands one file diff into the exact visual lines shown in the viewport.
func diffDisplayLines(file *internal.FileDiff, width int) []string {
	if file == nil {
		return nil
	}
	if file.IsBinary {
		if colorProfileFn() == termenv.Ascii {
			return []string{"Binary file changed"}
		}
		return []string{StyleDim.Render("Binary file changed")}
	}

	contentWidth := max(1, width-gutterWidth(file, width))
	lineNumberWidth := lineNoWidth(file)

	var lines []string
	for _, hunk := range file.Hunks {
		lines = append(lines, renderSeparator(width))
		for _, diffLine := range hunk.Lines {
			lineNo := displayedLineNo(diffLine)
			for i, segment := range wrapLineParts(diffLine.Content, contentWidth) {
				if gutterWidth(file, width) == 1 {
					prefix := "↳"
					if i == 0 {
						prefix = diffPrefix(diffLine.Type)
					}
					lines = append(lines, renderDiffSegment("", prefix, segment, diffLine.Type, width, file.Path))
					continue
				}

				lineNoText := strings.Repeat(" ", lineNumberWidth)
				prefix := "↳"
				if i == 0 {
					lineNoText = formatDisplayedLineNo(lineNo, lineNumberWidth)
					prefix = diffPrefix(diffLine.Type)
				}
				lines = append(lines, renderDiffSegment(lineNoText, prefix, segment, diffLine.Type, width, file.Path))
			}
		}
	}

	return lines
}

// countWrappedDiffLines counts visual rows using only diff prefixes and
// wrapping rules, so scroll math does not need to recompute ANSI highlighting.
func countWrappedDiffLines(file *internal.FileDiff, width int) int {
	if file == nil {
		return 0
	}
	if file.IsBinary {
		return 1
	}

	contentWidth := max(1, width-gutterWidth(file, width))

	total := 0
	for _, hunk := range file.Hunks {
		total++

		for _, diffLine := range hunk.Lines {
			total += len(wrapLineParts(diffLine.Content, contentWidth))
		}
	}

	return total
}

// wrapLine wraps one rendered line by terminal cell width.
func wrapLine(line string, width int) string {
	return strings.Join(wrapLineParts(line, width), "\n")
}

// highlightDiffSegment applies syntax highlighting to one wrapped code segment.
func highlightDiffSegment(segment, filename string) string {
	return HighlightCode(segment, filename)
}

// styleDiffPrefix applies the existing add/delete color to the diff prefix
// while leaving context-line prefixes unstyled.
func styleDiffPrefix(prefix string, lineType internal.LineType) string {
	if colorProfileFn() == termenv.Ascii || colorProfileFn() == termenv.TrueColor {
		return prefix
	}

	switch lineType {
	case internal.LineAdd:
		return StyleAdd.Render(prefix)
	case internal.LineDel:
		return StyleDel.Render(prefix)
	default:
		return prefix
	}
}

// wrapLineParts wraps one code line into visual lines by content width.
func wrapLineParts(code string, contentWidth int) []string {
	code = expandTabs(code)

	if contentWidth <= 0 || runewidth.StringWidth(code) <= contentWidth {
		return []string{code}
	}

	var result []string
	remaining := code

	for len(remaining) > 0 {
		chunk := truncateToWidth(remaining, contentWidth)
		result = append(result, chunk)
		remaining = remaining[len(chunk):]
	}

	return result
}

// diffPrefix returns the prefix symbol for the current diff line type.
func diffPrefix(lineType internal.LineType) string {
	switch lineType {
	case internal.LineAdd:
		return "+"
	case internal.LineDel:
		return "-"
	default:
		return " "
	}
}

// displayedLineNo selects the line number shown for one diff line.
func displayedLineNo(line internal.DiffLine) int {
	if line.Type == internal.LineDel {
		return line.OldLineNo
	}

	return line.NewLineNo
}

// formatDisplayedLineNo renders one line number or spaces when the number is unknown.
func formatDisplayedLineNo(lineNo, width int) string {
	if lineNo == 0 {
		return strings.Repeat(" ", width)
	}

	return fmt.Sprintf("%*d", width, lineNo)
}

// resolvedBgHex selects the light or dark adaptive color variant for the
// current terminal background.
func resolvedBgHex(color lipgloss.AdaptiveColor) string {
	if hasDarkBackgroundFn() {
		return color.Dark
	}

	return color.Light
}

// renderDiffSegment assembles one visual row and applies low-color prefix
// styling or true-color backgrounds depending on terminal capability.
func renderDiffSegment(lineNoText, prefix, code string, lineType internal.LineType, width int, filename string) string {
	highlighted := highlightDiffSegment(code, filename)
	if lineNoText == "" {
		assembled := styleDiffPrefix(prefix, lineType) + highlighted
		if colorProfileFn() != termenv.TrueColor {
			return assembled
		}

		padded := assembled + strings.Repeat(" ", max(0, width-lipgloss.Width(assembled)))
		switch lineType {
		case internal.LineAdd:
			return applyBg(padded, resolvedBgHex(BgAdd))
		case internal.LineDel:
			return applyBg(padded, resolvedBgHex(BgDel))
		default:
			return padded
		}
	}

	lineNoRender := lineNoText
	if colorProfileFn() != termenv.Ascii {
		lineNoRender = StyleDim.Render(lineNoText)
	}

	assembled := lineNoRender + " " + styleDiffPrefix(prefix, lineType) + highlighted
	if colorProfileFn() != termenv.TrueColor {
		return assembled
	}

	padded := assembled + strings.Repeat(" ", max(0, width-lipgloss.Width(assembled)))
	switch lineType {
	case internal.LineAdd:
		return applyBg(padded, resolvedBgHex(BgAdd))
	case internal.LineDel:
		return applyBg(padded, resolvedBgHex(BgDel))
	default:
		return padded
	}
}

// expandTabs replaces tab characters with spaces so width calculations match
// how the terminal will actually lay out the rendered diff lines.
func expandTabs(line string) string {
	if !strings.ContainsRune(line, '\t') {
		return line
	}

	var expanded strings.Builder
	column := 0

	for _, r := range line {
		if r != '\t' {
			expanded.WriteRune(r)
			if runeWidth := runewidth.RuneWidth(r); runeWidth > 0 {
				column += runeWidth
			}
			continue
		}

		spaces := terminalTabStop - (column % terminalTabStop)
		if spaces == 0 {
			spaces = terminalTabStop
		}
		expanded.WriteString(strings.Repeat(" ", spaces))
		column += spaces
	}

	return expanded.String()
}

// truncateToWidth returns the longest prefix that fits within width cells.
func truncateToWidth(s string, width int) string {
	cellWidth := 0
	for i, r := range s {
		runeWidth := runewidth.RuneWidth(r)
		if cellWidth+runeWidth > width {
			return s[:i]
		}
		cellWidth += runeWidth
	}

	return s
}
