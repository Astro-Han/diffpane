package ui

import (
	"fmt"
	"strings"

	"github.com/Astro-Han/diffpane/internal"
)

// RenderHeader renders the single-line top bar.
func RenderHeader(dirName string, files []internal.FileDiff, currentIdx, newCount int) string {
	if len(files) == 0 {
		return StyleDim.Render(fmt.Sprintf("%s · watching", dirName))
	}

	file := files[currentIdx]
	paths := make([]string, len(files))
	for i, item := range files {
		paths[i] = item.Path
	}
	shortPaths := ShortestUniquePaths(paths)
	displayPath := shortPaths[currentIdx]

	var stats string
	switch {
	case file.IsBinary:
		stats = StyleDim.Render("[binary]")
	case file.Status == internal.StatusDeleted:
		stats = StyleDel.Render("[deleted]")
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

	result := fmt.Sprintf("%s %s", displayPath, stats)
	if len(files) > 1 {
		result += StyleDim.Render(fmt.Sprintf(" ‹ %d/%d ›", currentIdx+1, len(files)))
	}
	if newCount > 0 {
		result += "  " + StyleAdd.Render(fmt.Sprintf("+%d new", newCount))
	}

	return result
}
