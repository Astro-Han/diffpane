package ui

import (
	"fmt"
	"strings"

	"github.com/Astro-Han/diffpane/internal"
	"github.com/mattn/go-runewidth"
)

// RenderDiffView renders the current file diff within the viewport.
func RenderDiffView(file *internal.FileDiff, scrollOffset, width, height int) string {
	lines := diffDisplayLines(file, width)

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
	return len(diffDisplayLines(file, width))
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
			switch diffLine.Type {
			case internal.LineAdd:
				for _, visualLine := range wrapLineParts("+"+diffLine.Content, width) {
					lines = append(lines, StyleAdd.Render(visualLine))
				}
			case internal.LineDel:
				for _, visualLine := range wrapLineParts("-"+diffLine.Content, width) {
					lines = append(lines, StyleDel.Render(visualLine))
				}
			default:
				lines = append(lines, wrapLineParts(" "+diffLine.Content, width)...)
			}
		}
	}

	return lines
}

// wrapLine wraps one rendered line by terminal cell width.
func wrapLine(line string, width int) string {
	return strings.Join(wrapLineParts(line, width), "\n")
}

// wrapLineParts wraps one rendered line into visual lines by terminal cell width.
func wrapLineParts(line string, width int) []string {
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
