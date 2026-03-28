package ui

import (
	"strings"

	"github.com/Astro-Han/diffpane/internal"
	"github.com/mattn/go-runewidth"
)

// RenderDiffView renders the current file diff within the viewport.
func RenderDiffView(file *internal.FileDiff, scrollOffset, width, height int) string {
	if file == nil {
		return ""
	}
	if file.IsBinary {
		return StyleDim.Render("Binary file changed")
	}

	var lines []string
	for _, hunk := range file.Hunks {
		lines = append(lines, StyleDim.Render(hunk.Header))
		for _, diffLine := range hunk.Lines {
			switch diffLine.Type {
			case internal.LineAdd:
				lines = append(lines, StyleAdd.Render(wrapLine("+"+diffLine.Content, width)))
			case internal.LineDel:
				lines = append(lines, StyleDel.Render(wrapLine("-"+diffLine.Content, width)))
			default:
				lines = append(lines, wrapLine(" "+diffLine.Content, width))
			}
		}
	}

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

// wrapLine wraps one rendered line by terminal cell width.
func wrapLine(line string, width int) string {
	if width <= 0 || runewidth.StringWidth(line) <= width {
		return line
	}

	var result strings.Builder
	first := truncateToWidth(line, width)
	result.WriteString(first)
	remaining := line[len(first):]

	for len(remaining) > 0 {
		result.WriteString("\n")
		prefix := "  "
		chunkWidth := width - runewidth.StringWidth(prefix)
		if chunkWidth <= 0 {
			chunkWidth = 1
		}
		chunk := truncateToWidth(remaining, chunkWidth)
		result.WriteString(prefix)
		result.WriteString(chunk)
		remaining = remaining[len(chunk):]
	}

	return result.String()
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
