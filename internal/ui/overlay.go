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
		var stats string
		switch {
		case file.IsBinary:
			stats = StyleDim.Render("[binary]")
		case file.Status == internal.StatusDeleted:
			stats = StyleDim.Render("[deleted]")
		default:
			var parts []string
			if file.AddCount > 0 {
				parts = append(parts, StyleAdd.Render(fmt.Sprintf("+%d", file.AddCount)))
			}
			if file.DelCount > 0 {
				parts = append(parts, StyleDel.Render(fmt.Sprintf("-%d", file.DelCount)))
			}
			stats = strings.Join(parts, " ")
		}

		if i == cursor {
			lines = append(lines, fmt.Sprintf("▸ %s  %s", file.Path, stats))
		} else {
			lines = append(lines, fmt.Sprintf("  %s  %s", file.Path, stats))
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
