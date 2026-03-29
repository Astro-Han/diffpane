package git

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Astro-Han/diffpane/internal"
)

// hunkHeaderRe extracts the new-file start line from a unified diff hunk header.
// The count after the comma is optional when git omits a value of 1.
var hunkHeaderRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// ParseDiff parses unified git diff output into file-level diff data.
func ParseDiff(raw string) []internal.FileDiff {
	if raw == "" {
		return nil
	}

	var files []internal.FileDiff
	var current *internal.FileDiff
	var hunk *internal.DiffHunk

	// flushHunk appends the current hunk before the parser moves on.
	flushHunk := func() {
		if current != nil && hunk != nil {
			current.Hunks = append(current.Hunks, *hunk)
			hunk = nil
		}
	}

	// flushFile appends the current file after all its hunks are complete.
	flushFile := func() {
		flushHunk()
		if current != nil {
			files = append(files, *current)
			current = nil
		}
	}

	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			flushFile()

			path := ""
			if parts := strings.SplitN(line, " b/", 2); len(parts) == 2 {
				path = parts[1]
			}

			current = &internal.FileDiff{
				Path:   path,
				Status: internal.StatusModified,
			}
			continue
		}
		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "Binary files"):
			current.IsBinary = true
			current.Status = internal.StatusBinary
		case strings.HasPrefix(line, "deleted file"):
			current.Status = internal.StatusDeleted
		case strings.HasPrefix(line, "new file"):
			current.Status = internal.StatusAdded
		case strings.HasPrefix(line, "--- /dev/null"):
			current.Status = internal.StatusAdded
		case strings.HasPrefix(line, "+++ /dev/null"):
			current.Status = internal.StatusDeleted
		case strings.HasPrefix(line, "---"),
			strings.HasPrefix(line, "+++"),
			strings.HasPrefix(line, "index "):
			// Skip metadata lines that do not belong to hunk content.
		case strings.HasPrefix(line, "@@"):
			flushHunk()
			startLine := 0
			if match := hunkHeaderRe.FindStringSubmatch(line); match != nil {
				startLine, _ = strconv.Atoi(match[1])
			}
			hunk = &internal.DiffHunk{
				Header:    line,
				StartLine: startLine,
			}
		default:
			if hunk == nil {
				continue
			}

			switch {
			case strings.HasPrefix(line, "+"):
				hunk.Lines = append(hunk.Lines, internal.DiffLine{
					Type:    internal.LineAdd,
					Content: line[1:],
				})
				current.AddCount++
			case strings.HasPrefix(line, "-"):
				hunk.Lines = append(hunk.Lines, internal.DiffLine{
					Type:    internal.LineDel,
					Content: line[1:],
				})
				current.DelCount++
			case strings.HasPrefix(line, " "):
				hunk.Lines = append(hunk.Lines, internal.DiffLine{
					Type:    internal.LineContext,
					Content: line[1:],
				})
			}
		}
	}

	flushFile()
	return files
}
