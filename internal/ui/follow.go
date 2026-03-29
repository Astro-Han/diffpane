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

// lastChangedHunkIndex finds the newest hunk whose fingerprint was absent before.
// Callers handle the "brand new path" case separately via prevHunkSigs membership.
func lastChangedHunkIndex(oldSigs []uint64, newHunks []internal.DiffHunk) int {
	seen := make(map[uint64]struct{}, len(oldSigs))
	for _, sig := range oldSigs {
		seen[sig] = struct{}{}
	}

	newSigs := hunkFingerprints(newHunks)
	for i := len(newSigs) - 1; i >= 0; i-- {
		if _, ok := seen[newSigs[i]]; !ok {
			return i
		}
	}

	return -1
}

// hunkVisualOffset counts rendered rows before hunkIdx for follow-mode scrolling.
func hunkVisualOffset(file *internal.FileDiff, hunkIdx, width int) int {
	if file == nil || hunkIdx <= 0 {
		return 0
	}

	offset := 0
	for i := 0; i < hunkIdx && i < len(file.Hunks); i++ {
		hunk := file.Hunks[i]
		offset++
		for _, line := range hunk.Lines {
			switch line.Type {
			case internal.LineAdd:
				offset += len(wrapLineParts("+"+line.Content, width))
			case internal.LineDel:
				offset += len(wrapLineParts("-"+line.Content, width))
			default:
				offset += len(wrapLineParts(" "+line.Content, width))
			}
		}
	}

	return offset
}
