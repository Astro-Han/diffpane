package ui

import (
	"fmt"
	"hash/fnv"
	"strings"
	"sync"

	"github.com/Astro-Han/diffpane/internal"
	"github.com/mattn/go-runewidth"
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
		_, _ = hasher.Write([]byte(fmt.Sprintf("%d\x00", hunk.StartLine)))
		for _, line := range hunk.Lines {
			_, _ = hasher.Write([]byte(fmt.Sprintf("%d\x00%s\x00", line.Type, line.Content)))
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

// renderSeparator builds a fixed-width hunk separator for the current viewport.
func renderSeparator(startLine, width int) string {
	const dash = "─"

	plainSeparator := func() string {
		count := width
		if count < 1 {
			count = 1
		}
		return StyleDim.Render(strings.Repeat(dash, count))
	}

	if startLine <= 1 {
		return plainSeparator()
	}

	label := fmt.Sprintf(" L%d ", startLine)
	minWidth := 2 + runewidth.StringWidth(label) + 2
	if width < minWidth {
		return plainSeparator()
	}

	return StyleDim.Render("──" + label + strings.Repeat(dash, width-runewidth.StringWidth("──"+label)))
}

// diffDisplayLines expands one file diff into the exact visual lines shown in the viewport.
func diffDisplayLines(file *internal.FileDiff, width int) []string {
	if file == nil {
		return nil
	}
	if file.IsBinary {
		return []string{StyleDim.Render("Binary file changed")}
	}

	var lines []string
	for _, hunk := range file.Hunks {
		lines = append(lines, renderSeparator(hunk.StartLine, width))
		for _, diffLine := range hunk.Lines {
			prefix := " "
			switch diffLine.Type {
			case internal.LineAdd:
				prefix = "+"
			case internal.LineDel:
				prefix = "-"
			}

			for i, segment := range wrapLineParts(prefix+diffLine.Content, width) {
				lines = append(lines, highlightDiffSegment(segment, i, diffLine.Type, file.Path))
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

	total := 0
	for _, hunk := range file.Hunks {
		total++

		for _, diffLine := range hunk.Lines {
			prefix := " "
			switch diffLine.Type {
			case internal.LineAdd:
				prefix = "+"
			case internal.LineDel:
				prefix = "-"
			}

			total += len(wrapLineParts(prefix+diffLine.Content, width))
		}
	}

	return total
}

// wrapLine wraps one rendered line by terminal cell width.
func wrapLine(line string, width int) string {
	return strings.Join(wrapLineParts(line, width), "\n")
}

// highlightDiffSegment applies diff-prefix styling and chroma code colors to
// one already-wrapped visual diff segment.
func highlightDiffSegment(segment string, segmentIndex int, lineType internal.LineType, filename string) string {
	prefixLength := 1
	if segmentIndex > 0 {
		prefixLength = 2
	}

	if len(segment) <= prefixLength {
		return styleDiffPrefix(segment, lineType)
	}

	prefix := segment[:prefixLength]
	code := segment[prefixLength:]

	return styleDiffPrefix(prefix, lineType) + HighlightCode(code, filename)
}

// styleDiffPrefix applies the existing add/delete color to the diff prefix
// while leaving context-line prefixes unstyled.
func styleDiffPrefix(prefix string, lineType internal.LineType) string {
	switch lineType {
	case internal.LineAdd:
		return StyleAdd.Render(prefix)
	case internal.LineDel:
		return StyleDel.Render(prefix)
	default:
		return prefix
	}
}

// wrapLineParts wraps one rendered line into visual lines by terminal cell width.
func wrapLineParts(line string, width int) []string {
	line = expandTabs(line)

	if width <= 0 || runewidth.StringWidth(line) <= width {
		return []string{line}
	}

	var result []string
	first := truncateToWidth(line, width)
	result = append(result, first)
	remaining := line[len(first):]

	for len(remaining) > 0 {
		prefix := "  "
		chunkWidth := width - runewidth.StringWidth(prefix)
		if chunkWidth <= 0 {
			chunkWidth = 1
		}
		chunk := truncateToWidth(remaining, chunkWidth)
		result = append(result, prefix+chunk)
		remaining = remaining[len(chunk):]
	}

	return result
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
