package ui

import (
	"hash/fnv"

	"github.com/Astro-Han/diffpane/internal"
)

// lineKey identifies one diff line by its original position inside a hunk.
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

// changedLineKeys reports the add/delete lines that do not appear in the old snapshot.
func changedLineKeys(oldSigs []uint64, newHunks []internal.DiffHunk) map[lineKey]bool {
	seen := make(map[uint64]int, len(oldSigs))
	for _, sig := range oldSigs {
		seen[sig]++
	}

	changed := make(map[lineKey]bool)
	for hunkIdx, hunk := range newHunks {
		for lineIdx, line := range hunk.Lines {
			if line.Type == internal.LineContext {
				continue
			}

			sig := hashDiffLine(line)
			if seen[sig] > 0 {
				seen[sig]--
			} else {
				changed[lineKey{HunkIdx: hunkIdx, LineIdx: lineIdx}] = true
			}
		}
	}

	return changed
}

// hunkVisualOffset counts rendered rows before the target line for follow-mode scrolling.
func hunkVisualOffset(file *internal.FileDiff, hunkIdx, lineIdx, width int) int {
	if file == nil || hunkIdx < 0 || lineIdx < 0 {
		return 0
	}

	contentWidth := max(1, width-gutterWidth(file, width))
	offset := 0
	for i := 0; i < hunkIdx && i < len(file.Hunks); i++ {
		offset++
		for _, line := range file.Hunks[i].Lines {
			offset += len(wrapLineParts(line.Content, contentWidth))
		}
	}

	if hunkIdx >= len(file.Hunks) {
		return offset
	}

	offset++
	for i := 0; i < lineIdx && i < len(file.Hunks[hunkIdx].Lines); i++ {
		offset += len(wrapLineParts(file.Hunks[hunkIdx].Lines[i].Content, contentWidth))
	}

	return offset
}
