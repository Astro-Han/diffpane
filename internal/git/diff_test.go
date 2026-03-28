package git

import (
	"os"
	"path/filepath"
	"testing"
)

// TestComputeDiffModifiedFile verifies tracked modifications become unified diffs.
func TestComputeDiffModifiedFile(t *testing.T) {
	root := initGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "init")

	baseline, err := GetHeadSHA(root)
	if err != nil {
		t.Fatalf("GetHeadSHA returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello\nworld\n"), 0o600); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}

	files, err := ComputeDiff(root, baseline)
	if err != nil {
		t.Fatalf("ComputeDiff returned error: %v", err)
	}
	if len(files) != 1 || files[0].Path != "hello.txt" {
		t.Fatalf("expected 1 file hello.txt, got %#v", files)
	}
	if files[0].AddCount != 1 {
		t.Fatalf("add count = %d, want 1", files[0].AddCount)
	}
}

// TestComputeDiffUntrackedFile verifies new files are represented as added hunks.
func TestComputeDiffUntrackedFile(t *testing.T) {
	root := initGitRepo(t)
	runGit(t, root, "commit", "--allow-empty", "-m", "init")

	baseline, err := GetHeadSHA(root)
	if err != nil {
		t.Fatalf("GetHeadSHA returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("a\nb\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	files, err := ComputeDiff(root, baseline)
	if err != nil {
		t.Fatalf("ComputeDiff returned error: %v", err)
	}
	if len(files) != 1 || files[0].Path != "new.txt" {
		t.Fatalf("expected 1 file new.txt, got %#v", files)
	}
	if files[0].AddCount != 2 {
		t.Fatalf("add count = %d, want 2", files[0].AddCount)
	}
}

// TestComputeDiffUntrackedDirectory verifies new directories are expanded to file level.
func TestComputeDiffUntrackedDirectory(t *testing.T) {
	root := initGitRepo(t)
	runGit(t, root, "commit", "--allow-empty", "-m", "init")

	baseline, err := GetHeadSHA(root)
	if err != nil {
		t.Fatalf("GetHeadSHA returned error: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "src"), 0o750); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "a.go"), []byte("package src\n"), 0o600); err != nil {
		t.Fatalf("write a.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "b.go"), []byte("package src\n"), 0o600); err != nil {
		t.Fatalf("write b.go: %v", err)
	}

	files, err := ComputeDiff(root, baseline)
	if err != nil {
		t.Fatalf("ComputeDiff returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

// TestComputeDiffFreshRepoNoChanges verifies an unborn repo can render the empty state.
func TestComputeDiffFreshRepoNoChanges(t *testing.T) {
	root := initGitRepo(t)

	baseline, err := GetHeadSHA(root)
	if err != nil {
		t.Fatalf("GetHeadSHA returned error: %v", err)
	}

	files, err := ComputeDiff(root, baseline)
	if err != nil {
		t.Fatalf("ComputeDiff returned error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

// TestComputeDiffFreshRepoUntrackedFile verifies fresh repos still show untracked files.
func TestComputeDiffFreshRepoUntrackedFile(t *testing.T) {
	root := initGitRepo(t)

	baseline, err := GetHeadSHA(root)
	if err != nil {
		t.Fatalf("GetHeadSHA returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "fresh.txt"), []byte("hello\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	files, err := ComputeDiff(root, baseline)
	if err != nil {
		t.Fatalf("ComputeDiff returned error: %v", err)
	}
	if len(files) != 1 || files[0].Path != "fresh.txt" {
		t.Fatalf("expected 1 file fresh.txt, got %#v", files)
	}
}

// TestComputeDiffNoChanges verifies a clean worktree produces no diffs.
func TestComputeDiffNoChanges(t *testing.T) {
	root := initGitRepo(t)
	runGit(t, root, "commit", "--allow-empty", "-m", "init")

	baseline, err := GetHeadSHA(root)
	if err != nil {
		t.Fatalf("GetHeadSHA returned error: %v", err)
	}

	files, err := ComputeDiff(root, baseline)
	if err != nil {
		t.Fatalf("ComputeDiff returned error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}
