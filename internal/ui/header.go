package ui

import (
	"fmt"
	"strings"

	"github.com/Astro-Han/diffpane/internal"
)

// RenderHeader renders the single-line top bar.
func RenderHeader(dirName string, files []internal.FileDiff, currentIdx, width int) string {
	if len(files) == 0 {
		return clampInlineWidth(StyleDim.Render(fmt.Sprintf("%s | watching", dirName)), width)
	}

	currentIdx = min(max(currentIdx, 0), len(files)-1)
	file := files[currentIdx]

	result := fmt.Sprintf("%s %s", file.Path, renderFileStats(file))
	if len(files) > 1 {
		result += StyleDim.Render(fmt.Sprintf(" < %d/%d >", currentIdx+1, len(files)))
	}

	return clampInlineWidth(result, width)
}

// renderFileStats formats one file's change summary for header and overlay views.
func renderFileStats(file internal.FileDiff) string {
	switch {
	case file.IsBinary:
		return StyleDim.Render("[binary]")
	case file.Status == internal.StatusDeleted:
		return StyleDel.Render("[deleted]")
	default:
		var parts []string
		if file.AddCount > 0 {
			parts = append(parts, StyleAdd.Render(fmt.Sprintf("+%d", file.AddCount)))
		}
		if file.DelCount > 0 {
			parts = append(parts, StyleDel.Render(fmt.Sprintf("-%d", file.DelCount)))
		}
		return strings.Join(parts, " ")
	}
}
