package ui

import (
	"hash/fnv"

	"github.com/Astro-Han/diffpane/internal"
)

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

// lastChangedHunkIndex keeps legacy callers compiling until model follow logic
// switches to highlighted-hunk state in a later TDD step.
func lastChangedHunkIndex(oldSigs []uint64, newHunks []internal.DiffHunk) int {
	changed := changedHunkIndices(oldSigs, newHunks)
	if len(changed) == 0 {
		return -1
	}

	return changed[len(changed)-1]
}

// hunkVisualOffset counts rendered rows before hunkIdx for follow-mode scrolling.
func hunkVisualOffset(file *internal.FileDiff, hunkIdx, width int) int {
	if file == nil || hunkIdx <= 0 {
		return 0
	}

	contentWidth := max(1, width-gutterWidth(file, width))
	offset := 0
	for i := 0; i < hunkIdx && i < len(file.Hunks); i++ {
		hunk := file.Hunks[i]
		offset++
		for _, line := range hunk.Lines {
			offset += len(wrapLineParts(line.Content, contentWidth))
		}
	}

	return offset
}
