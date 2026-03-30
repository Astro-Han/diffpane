package git

import (
	"reflect"
	"testing"

	"github.com/Astro-Han/diffpane/internal"
)

func TestDiffHunkHasStartLineField(t *testing.T) {
	// The diff hunk API must expose the parsed new-file start line for rendering.
	field, ok := reflect.TypeOf(internal.DiffHunk{}).FieldByName("StartLine")
	if !ok {
		t.Fatal("DiffHunk should include StartLine field")
	}
	if field.Type.Kind() != reflect.Int {
		t.Fatalf("StartLine kind = %s, want int", field.Type.Kind())
	}
}

// TestDiffLineHasLineNumberFields verifies diff lines expose both old-side and
// new-side line numbers for later rendering.
func TestDiffLineHasLineNumberFields(t *testing.T) {
	field, ok := reflect.TypeOf(internal.DiffLine{}).FieldByName("OldLineNo")
	if !ok || field.Type.Kind() != reflect.Int {
		t.Fatal("DiffLine should include OldLineNo int field")
	}

	field, ok = reflect.TypeOf(internal.DiffLine{}).FieldByName("NewLineNo")
	if !ok || field.Type.Kind() != reflect.Int {
		t.Fatal("DiffLine should include NewLineNo int field")
	}
}

// TestDiffHunkHasOldStartLineField verifies hunks expose the old-side start
// line so deleted rows can render the correct numbers.
func TestDiffHunkHasOldStartLineField(t *testing.T) {
	field, ok := reflect.TypeOf(internal.DiffHunk{}).FieldByName("OldStartLine")
	if !ok || field.Type.Kind() != reflect.Int {
		t.Fatal("DiffHunk should include OldStartLine int field")
	}
}

func TestParseDiffSingleFile(t *testing.T) {
	input := `diff --git a/src/auth.ts b/src/auth.ts
index abc1234..def5678 100644
--- a/src/auth.ts
+++ b/src/auth.ts
@@ -47,6 +47,8 @@ function validate() {
   const decoded = jwt.verify(token);
+  if (!decoded) return null;
+  log("token validated");
   return decoded.payload;
 }
`
	files := ParseDiff(input)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]
	if file.Path != "src/auth.ts" {
		t.Fatalf("path = %q, want %q", file.Path, "src/auth.ts")
	}
	if file.AddCount != 2 {
		t.Fatalf("add count = %d, want 2", file.AddCount)
	}
	if len(file.Hunks) != 1 {
		t.Fatalf("hunk count = %d, want 1", len(file.Hunks))
	}
	if file.Hunks[0].StartLine != 47 {
		t.Fatalf("StartLine = %d, want 47", file.Hunks[0].StartLine)
	}
}

// TestParseDiffAssignsOldAndNewLineNumbers verifies line numbers are assigned
// during parsing for context, deleted, and added lines.
func TestParseDiffAssignsOldAndNewLineNumbers(t *testing.T) {
	input := `diff --git a/a.txt b/a.txt
--- a/a.txt
+++ b/a.txt
@@ -10,2 +20,3 @@
 line1
-line2
+line2 changed
+line3
`

	file := ParseDiff(input)[0]
	lines := file.Hunks[0].Lines

	if lines[0].OldLineNo != 10 || lines[0].NewLineNo != 20 {
		t.Fatalf("context line numbers = (%d,%d), want (10,20)", lines[0].OldLineNo, lines[0].NewLineNo)
	}
	if lines[1].OldLineNo != 11 || lines[1].NewLineNo != 0 {
		t.Fatalf("deleted line numbers = (%d,%d), want (11,0)", lines[1].OldLineNo, lines[1].NewLineNo)
	}
	if lines[2].OldLineNo != 0 || lines[2].NewLineNo != 21 {
		t.Fatalf("added line numbers = (%d,%d), want (0,21)", lines[2].OldLineNo, lines[2].NewLineNo)
	}
}

// TestParseDiffMalformedHeaderLeavesLineNumbersBlank verifies malformed hunk
// headers do not invent line numbers.
func TestParseDiffMalformedHeaderLeavesLineNumbersBlank(t *testing.T) {
	input := `diff --git a/x.txt b/x.txt
--- a/x.txt
+++ b/x.txt
@@ broken header @@
+added line
`

	file := ParseDiff(input)[0]
	line := file.Hunks[0].Lines[0]
	if line.OldLineNo != 0 || line.NewLineNo != 0 {
		t.Fatalf("malformed header line numbers = (%d,%d), want zeros", line.OldLineNo, line.NewLineNo)
	}
}

func TestParseDiffBinary(t *testing.T) {
	input := `diff --git a/img.png b/img.png
Binary files /dev/null and b/img.png differ
`
	files := ParseDiff(input)
	if len(files) != 1 || !files[0].IsBinary {
		t.Fatal("expected 1 binary file")
	}
}

func TestParseDiffDeleted(t *testing.T) {
	input := `diff --git a/old.txt b/old.txt
deleted file mode 100644
--- a/old.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2
`
	files := ParseDiff(input)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Status != internal.StatusDeleted {
		t.Fatal("expected StatusDeleted")
	}
	if files[0].DelCount != 2 {
		t.Fatalf("delete count = %d, want 2", files[0].DelCount)
	}
}

func TestParseDiffMultipleFiles(t *testing.T) {
	input := `diff --git a/a.txt b/a.txt
--- a/a.txt
+++ b/a.txt
@@ -1 +1,2 @@
 hello
+world
diff --git a/b.txt b/b.txt
--- a/b.txt
+++ b/b.txt
@@ -1 +1 @@
-old
+new
`
	files := ParseDiff(input)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Hunks[0].StartLine != 1 {
		t.Fatalf("file[0] StartLine = %d, want 1", files[0].Hunks[0].StartLine)
	}
	if files[1].Hunks[0].StartLine != 1 {
		t.Fatalf("file[1] StartLine = %d, want 1", files[1].Hunks[0].StartLine)
	}
}

func TestParseDiffEmpty(t *testing.T) {
	if files := ParseDiff(""); len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestParseDiffStartLineAbbreviated(t *testing.T) {
	input := `diff --git a/a.txt b/a.txt
--- a/a.txt
+++ b/a.txt
@@ -1 +1,2 @@
 hello
+world
`
	files := ParseDiff(input)
	if files[0].Hunks[0].StartLine != 1 {
		t.Fatalf("StartLine = %d, want 1", files[0].Hunks[0].StartLine)
	}
}

func TestParseDiffStartLineBothAbbreviated(t *testing.T) {
	input := `diff --git a/b.txt b/b.txt
--- a/b.txt
+++ b/b.txt
@@ -1 +1 @@
-old
+new
`
	files := ParseDiff(input)
	if files[0].Hunks[0].StartLine != 1 {
		t.Fatalf("StartLine = %d, want 1", files[0].Hunks[0].StartLine)
	}
}

// TestParseDiffStartLineAbbreviatedNonOne verifies abbreviated headers keep
// non-1 new-file start lines.
func TestParseDiffStartLineAbbreviatedNonOne(t *testing.T) {
	input := `diff --git a/c.txt b/c.txt
--- a/c.txt
+++ b/c.txt
@@ -1 +38 @@
-old
+new
`
	files := ParseDiff(input)
	if files[0].Hunks[0].StartLine != 38 {
		t.Fatalf("StartLine = %d, want 38", files[0].Hunks[0].StartLine)
	}
}

// TestParseDiffStartLineWithSectionHeading verifies function-context suffixes
// do not interfere with start-line parsing.
func TestParseDiffStartLineWithSectionHeading(t *testing.T) {
	input := `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -10,2 +38,3 @@ func renderHeader() {
 line1
+line2
`
	files := ParseDiff(input)
	if files[0].Hunks[0].StartLine != 38 {
		t.Fatalf("StartLine = %d, want 38", files[0].Hunks[0].StartLine)
	}
}

func TestParseDiffStartLineDeleted(t *testing.T) {
	input := `diff --git a/old.txt b/old.txt
deleted file mode 100644
--- a/old.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2
`
	files := ParseDiff(input)
	if files[0].Hunks[0].StartLine != 0 {
		t.Fatalf("StartLine = %d, want 0", files[0].Hunks[0].StartLine)
	}
}

func TestParseDiffStartLineMalformed(t *testing.T) {
	input := `diff --git a/x.txt b/x.txt
--- a/x.txt
+++ b/x.txt
@@ broken header @@
+added line
`
	files := ParseDiff(input)
	if files[0].Hunks[0].StartLine != 0 {
		t.Fatalf("StartLine = %d, want 0", files[0].Hunks[0].StartLine)
	}
}

func TestBuildNewFileDiffStartLine(t *testing.T) {
	diff := buildNewFileDiff("new.txt", "line1\nline2\n")
	if len(diff.Hunks) != 1 {
		t.Fatalf("hunk count = %d, want 1", len(diff.Hunks))
	}
	if diff.Hunks[0].StartLine != 1 {
		t.Fatalf("StartLine = %d, want 1", diff.Hunks[0].StartLine)
	}
}

// TestBuildNewFileDiffLineNumbers verifies synthetic added-file diffs stamp
// the new-file line numbers onto each generated line.
func TestBuildNewFileDiffLineNumbers(t *testing.T) {
	diff := buildNewFileDiff("new.txt", "line1\nline2\n")
	if diff.Hunks[0].Lines[0].NewLineNo != 1 || diff.Hunks[0].Lines[1].NewLineNo != 2 {
		t.Fatalf("new file line numbers = %d,%d, want 1,2", diff.Hunks[0].Lines[0].NewLineNo, diff.Hunks[0].Lines[1].NewLineNo)
	}
}
