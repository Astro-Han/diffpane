package ui

import (
	"fmt"
	"strings"

	"github.com/Astro-Han/diffpane/internal"
)

// RenderOverlay renders the frozen file list overlay.
func RenderOverlay(files []internal.FileDiff, cursor, height, width int) string {
	if len(files) == 0 {
		return clampInlineWidth(StyleDim.Render("No changed files"), width)
	}

	var lines []string
	for i, file := range files {
		if i == cursor {
			lines = append(lines, fmt.Sprintf("▸ %s  %s", file.Path, renderFileStats(file)))
		} else {
			lines = append(lines, fmt.Sprintf("  %s  %s", file.Path, renderFileStats(file)))
		}
	}

	start := 0
	if cursor >= height {
		start = cursor - height + 1
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}
	for i := start; i < end; i++ {
		lines[i] = clampInlineWidth(lines[i], width)
	}
	return strings.Join(lines[start:end], "\n")
}
