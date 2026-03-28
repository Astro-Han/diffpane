package watcher

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches the repo worktree and real git dir for changes.
type FileWatcher struct {
	fsw          *fsnotify.Watcher
	repoDir      string
	gitDir       string
	commonGitDir string

	onChange func(changedPaths []string)
	onHead   func()

	debouncer     *Debouncer
	headDebouncer *Debouncer

	pendingMu    sync.Mutex
	pendingPaths []string
	pendingSeen  map[string]bool

	done      chan struct{}
	errWriter io.Writer
}

// New creates and starts a file watcher for the worktree and git metadata.
func New(repoDir, gitDir, commonGitDir string, onChange func([]string), onHead func()) (*FileWatcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &FileWatcher{
		fsw:          fsw,
		repoDir:      repoDir,
		gitDir:       gitDir,
		commonGitDir: commonGitDir,
		onChange:     onChange,
		onHead:       onHead,
		pendingSeen:  make(map[string]bool),
		done:         make(chan struct{}),
		errWriter:    os.Stderr,
	}

	watcher.debouncer = NewDebouncer(300*time.Millisecond, time.Second, func() {
		watcher.pendingMu.Lock()
		paths := watcher.pendingPaths
		watcher.pendingPaths = nil
		watcher.pendingSeen = make(map[string]bool)
		watcher.pendingMu.Unlock()
		if len(paths) > 0 {
			watcher.onChange(paths)
		}
	})

	watcher.headDebouncer = NewDebouncer(100*time.Millisecond, 100*time.Millisecond, onHead)

	watcher.addDirRecursive(repoDir)
	if err := watcher.fsw.Add(gitDir); err != nil {
		_ = watcher.fsw.Close()
		return nil, err
	}
	refsDir := filepath.Join(gitDir, "refs")
	watcher.addDirRecursive(refsDir)
	if commonGitDir != "" && commonGitDir != gitDir {
		commonRefsDir := filepath.Join(commonGitDir, "refs")
		watcher.addDirRecursive(commonRefsDir)
	}

	go watcher.loop()
	return watcher, nil
}

func (fw *FileWatcher) loop() {
	for {
		select {
		case event, ok := <-fw.fsw.Events:
			if !ok {
				return
			}

			path := event.Name
			if isGitInternalPath(path, fw.gitDir) || isGitInternalPath(path, fw.commonGitDir) {
				if isHeadOrRefPath(path, fw.gitDir) || isHeadOrRefPath(path, fw.commonGitDir) {
					fw.headDebouncer.Trigger()
				}
				continue
			}

			worktreeGit := filepath.Join(fw.repoDir, ".git")
			if path == worktreeGit {
				continue
			}
			if fw.isIgnored(path) {
				continue
			}

			if event.Has(fsnotify.Create) {
				fw.addDirRecursive(path)
			}

			rel, err := filepath.Rel(fw.repoDir, path)
			if err == nil {
				fw.pendingMu.Lock()
				if !fw.pendingSeen[rel] {
					fw.pendingSeen[rel] = true
					fw.pendingPaths = append(fw.pendingPaths, rel)
				}
				fw.pendingMu.Unlock()
			}
			fw.debouncer.Trigger()

		case err, ok := <-fw.fsw.Errors:
			if !ok {
				return
			}
			if err != nil {
				fw.reportError(err.Error())
			}
		case <-fw.done:
			return
		}
	}
}

// Stop closes underlying watchers and timers.
func (fw *FileWatcher) Stop() {
	close(fw.done)
	fw.debouncer.Stop()
	fw.headDebouncer.Stop()
	_ = fw.fsw.Close()
}

// isGitInternalPath returns true only for the real git internal directory tree.
func isGitInternalPath(path, gitDir string) bool {
	if gitDir == "" {
		return false
	}
	return path == gitDir || strings.HasPrefix(path, gitDir+string(filepath.Separator))
}

// isHeadOrRefPath returns true for HEAD and refs changes that reset the baseline.
func isHeadOrRefPath(path, gitDir string) bool {
	if gitDir == "" {
		return false
	}
	rel, err := filepath.Rel(gitDir, path)
	if err != nil {
		return false
	}

	return rel == "HEAD" || strings.HasPrefix(rel, "refs")
}

// isIgnored checks whether git would ignore this path.
func (fw *FileWatcher) isIgnored(path string) bool {
	rel := path
	if computed, err := filepath.Rel(fw.repoDir, path); err == nil {
		rel = computed
	}

	// #nosec G204 -- git command and checked path are scoped to the current repo.
	cmd := exec.Command("git", "check-ignore", "-q", rel)
	cmd.Dir = fw.repoDir
	return cmd.Run() == nil
}

func (fw *FileWatcher) reportError(message string) {
	if fw.errWriter == nil {
		return
	}
	_, _ = fmt.Fprintln(fw.errWriter, message)
}

func (fw *FileWatcher) addDirRecursive(dir string) {
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			_ = fw.fsw.Add(path)
		}
		return nil
	})
}
