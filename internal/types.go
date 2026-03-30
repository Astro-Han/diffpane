package internal

// LineType marks the semantic type of a diff line.
type LineType int

const (
	// LineContext keeps unchanged context around a hunk.
	LineContext LineType = iota
	// LineAdd marks an added line.
	LineAdd
	// LineDel marks a deleted line.
	LineDel
)

// DiffLine stores one rendered diff line without the +/- prefix.
type DiffLine struct {
	Type    LineType
	Content string
	// OldLineNo is the source file line number for deleted or context lines.
	OldLineNo int
	// NewLineNo is the destination file line number for added or context lines.
	NewLineNo int
}

// DiffHunk stores one @@ block from a unified diff.
type DiffHunk struct {
	Header string
	// OldStartLine is the first line number in the old-file range from the hunk header.
	OldStartLine int
	// StartLine is the first line number in the new-file range from the hunk header.
	StartLine int
	Lines     []DiffLine
}

// FileStatus marks the kind of change for a file.
type FileStatus int

const (
	// StatusModified means the file existed before and changed.
	StatusModified FileStatus = iota
	// StatusAdded means the file is new relative to baseline.
	StatusAdded
	// StatusDeleted means the file was removed relative to baseline.
	StatusDeleted
	// StatusBinary means the file changed but should not render textual hunks.
	StatusBinary
)

// FileDiff stores the parsed diff for one file.
type FileDiff struct {
	Path     string
	Status   FileStatus
	Hunks    []DiffHunk
	AddCount int
	DelCount int
	IsBinary bool
}
