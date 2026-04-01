package ui

import (
	"hash/fnv"

	"github.com/Astro-Han/diffpane/internal"
)

// lineKey identifies one non-context line inside a hunk.
type lineKey struct {
	HunkIdx int
	LineIdx int
}

// hashDiffLine hashes one added or deleted line by its semantic type and content.
func hashDiffLine(line internal.DiffLine) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte{byte(line.Type), 0})
	_, _ = hasher.Write([]byte(line.Content))
	_, _ = hasher.Write([]byte{0})
	return hasher.Sum64()
}

// lineFingerprints hashes only added and deleted lines, skipping context rows.
func lineFingerprints(hunks []internal.DiffHunk) []uint64 {
	sigs := make([]uint64, 0)
	for _, hunk := range hunks {
		for _, line := range hunk.Lines {
			if line.Type == internal.LineContext {
				continue
			}
			sigs = append(sigs, hashDiffLine(line))
		}
	}
	return sigs
}

// changedLineKeys reports the non-context lines that do not appear in the old snapshot.
func changedLineKeys(oldSigs []uint64, newHunks []internal.DiffHunk) map[lineKey]bool {
	seen := make(map[uint64]int, len(oldSigs))
	for _, sig := range oldSigs {
		seen[sig]++
	}

	changed := make(map[lineKey]bool)
	for hunkIdx, hunk := range newHunks {
		lineIdx := 0
		for _, line := range hunk.Lines {
			if line.Type == internal.LineContext {
				continue
			}

			sig := hashDiffLine(line)
			if seen[sig] > 0 {
				seen[sig]--
			} else {
				changed[lineKey{HunkIdx: hunkIdx, LineIdx: lineIdx}] = true
			}
			lineIdx++
		}
	}

	return changed
}

// hunkFingerprints hashes each hunk by rendered line semantics, excluding headers.
func hunkFingerprints(hunks []internal.DiffHunk) []uint64 {
	sigs := make([]uint64, 0, len(hunks))
	for _, hunk := range hunks {
		hasher := fnv.New64a()
		for _, line := range hunk.Lines {
			_, _ = hasher.Write([]byte{byte(line.Type), 0})
			_, _ = hasher.Write([]byte(line.Content))
			_, _ = hasher.Write([]byte{0})
		}
		sigs = append(sigs, hasher.Sum64())
	}
	return sigs
}

// changedHunkIndices finds every hunk whose fingerprint count exceeds the previous snapshot.
func changedHunkIndices(oldSigs []uint64, newHunks []internal.DiffHunk) []int {
	seen := make(map[uint64]int, len(oldSigs))
	for _, sig := range oldSigs {
		seen[sig]++
	}

	var changed []int
	for i, sig := range hunkFingerprints(newHunks) {
		if seen[sig] > 0 {
			seen[sig]--
			continue
		}
		changed = append(changed, i)
	}

	return changed
}

// hunkVisualOffset counts rendered rows before the target line for follow-mode scrolling.
func hunkVisualOffset(file *internal.FileDiff, targetLineIdx, width int) int {
	if file == nil || targetLineIdx < 0 {
		return 0
	}

	contentWidth := max(1, width-gutterWidth(file, width))
	// Keep the legacy hunk-based behavior for multi-hunk files until the
	// higher layers start passing a line index instead of a hunk index.
	if len(file.Hunks) > 1 && targetLineIdx < len(file.Hunks) {
		offset := 0
		for i := 0; i < targetLineIdx && i < len(file.Hunks); i++ {
			hunk := file.Hunks[i]
			offset++
			for _, line := range hunk.Lines {
				offset += len(wrapLineParts(line.Content, contentWidth))
			}
		}
		return offset
	}

	if len(file.Hunks) == 0 {
		return 0
	}

	offset := 1
	seenLines := 0
	for _, line := range file.Hunks[0].Lines {
		if seenLines >= targetLineIdx {
			break
		}
		offset += len(wrapLineParts(line.Content, contentWidth))
		if line.Type != internal.LineContext {
			seenLines++
		}
	}

	return offset
}
