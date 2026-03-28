package ui

import (
	"fmt"
	"strings"

	"github.com/Astro-Han/diffpane/internal"
)

// RenderOverlay renders the frozen file list overlay.
func RenderOverlay(files []internal.FileDiff, cursor, height int) string {
	if len(files) == 0 {
		return StyleDim.Render("No changed files")
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
	return strings.Join(lines[start:end], "\n")
}
