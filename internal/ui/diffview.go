package ui

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"

	"github.com/Astro-Han/diffpane/internal"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

const terminalTabStop = 8

type displayLineCacheKey struct {
	path         string
	width        int
	isBinary     bool
	status       internal.FileStatus
	signature    uint64
	highlightSig uint64
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

// get returns cached rendered lines when the file signature, width, and
// highlight state match.
func (c *displayLineCache) get(file *internal.FileDiff, width int, highlightSet map[int]bool, build func() []string) []string {
	if c == nil {
		return build()
	}

	key := newDisplayLineCacheKey(file, width, highlightSet)

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
func newDisplayLineCacheKey(file *internal.FileDiff, width int, highlightSet map[int]bool) displayLineCacheKey {
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
	key.highlightSig = highlightSignature(highlightSet)
	return key
}

// RenderDiffView renders the current file diff within the viewport.
func RenderDiffView(file *internal.FileDiff, scrollOffset, width, height int, highlightSet map[int]bool) string {
	lines := diffDisplayLines(file, width, highlightSet)
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
func diffDisplayLines(file *internal.FileDiff, width int, highlightSet map[int]bool) []string {
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
	for hunkIdx, hunk := range file.Hunks {
		highlightedHunk := highlightSet[hunkIdx]
		lines = append(lines, renderSeparator(width))
		for _, diffLine := range hunk.Lines {
			lineNo := displayedLineNo(diffLine)
			for i, segment := range wrapLineParts(diffLine.Content, contentWidth) {
				if gutterWidth(file, width) == 1 {
					prefix := "↳"
					if i == 0 {
						prefix = diffPrefix(diffLine.Type)
					}
					lines = append(lines, renderDiffSegment("", prefix, segment, diffLine.Type, width, file.Path, highlightedHunk))
					continue
				}

				lineNoText := strings.Repeat(" ", lineNumberWidth)
				prefix := "↳"
				if i == 0 {
					lineNoText = formatDisplayedLineNo(lineNo, lineNumberWidth)
					prefix = diffPrefix(diffLine.Type)
				}
				lines = append(lines, renderDiffSegment(lineNoText, prefix, segment, diffLine.Type, width, file.Path, highlightedHunk))
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

// styleDiffPrefix applies add/delete foreground color to prefixes when needed
// while leaving context-line prefixes unstyled.
func styleDiffPrefix(prefix string, lineType internal.LineType, forceColor bool) string {
	profile := colorProfileFn()
	// TrueColor with known theme: prefix inherits background, no foreground color needed.
	// Ascii: no ANSI codes at all.
	// TrueColor with unknown theme: use foreground colors (same as ANSI256/ANSI).
	if profile == termenv.Ascii {
		return prefix
	}
	if !forceColor && profile == termenv.TrueColor && GetTheme() != ThemeUnknown {
		return prefix
	}

	switch lineType {
	case internal.LineAdd:
		if forceColor && profile == termenv.TrueColor && GetTheme() != ThemeUnknown {
			return applyFg(prefix, resolvedAdaptiveHex(ColorAdd))
		}
		return StyleAdd.Render(prefix)
	case internal.LineDel:
		if forceColor && profile == termenv.TrueColor && GetTheme() != ThemeUnknown {
			return applyFg(prefix, resolvedAdaptiveHex(ColorDel))
		}
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
// current terminal background. Returns empty string when theme is unknown,
// which disables background coloring gracefully.
func resolvedBgHex(color lipgloss.AdaptiveColor) string {
	switch GetTheme() {
	case ThemeDark:
		return color.Dark
	case ThemeLight:
		return color.Light
	default:
		// Unknown theme: don't apply background to avoid wrong colors.
		return ""
	}
}

func resolvedAdaptiveHex(color lipgloss.AdaptiveColor) string {
	switch GetTheme() {
	case ThemeDark:
		return color.Dark
	case ThemeLight:
		return color.Light
	default:
		return ""
	}
}

func applyFg(text, hexColor string) string {
	if hexColor == "" {
		return text
	}

	return hexToFgANSI(hexColor) + text + "\033[0m"
}

func hexToFgANSI(hex string) string {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// renderDiffSegment assembles one visual row and applies prefix styling and
// optional true-color backgrounds depending on terminal capability.
func renderDiffSegment(lineNoText, prefix, code string, lineType internal.LineType, width int, filename string, highlighted bool) string {
	codeText := highlightDiffSegment(code, filename)
	forcePrefixColor := !highlighted && (lineType == internal.LineAdd || lineType == internal.LineDel)
	if lineNoText == "" {
		assembled := styleDiffPrefix(prefix, lineType, forcePrefixColor) + codeText
		if colorProfileFn() != termenv.TrueColor || !highlighted {
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

	assembled := lineNoRender + " " + styleDiffPrefix(prefix, lineType, forcePrefixColor) + codeText
	if colorProfileFn() != termenv.TrueColor || !highlighted {
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

func highlightSignature(highlightSet map[int]bool) uint64 {
	if len(highlightSet) == 0 {
		return 0
	}

	indices := make([]int, 0, len(highlightSet))
	for idx := range highlightSet {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	hasher := fnv.New64a()
	for _, idx := range indices {
		_, _ = hasher.Write([]byte(fmt.Sprintf("%d\x00", idx)))
	}

	return hasher.Sum64()
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
