package git

import (
	"testing"

	"github.com/Astro-Han/diffpane/internal"
)

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
}

func TestParseDiffEmpty(t *testing.T) {
	if files := ParseDiff(""); len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}
