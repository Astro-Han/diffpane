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
