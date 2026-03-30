package ui

import (
	"reflect"
	"strings"
	"time"

	"github.com/Astro-Han/diffpane/internal"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is the root Bubble Tea model for diffpane.
type Model struct {
	DirName     string
	RepoDir     string
	BaselineSHA string

	Files      []internal.FileDiff
	CurrentIdx int

	FollowOn        bool
	ScrollOffset    int
	hunkSigs        map[string][]uint64
	highlightedHunks map[string]map[int]bool
	lastChangedPath string
	lastHighlightedPath string
	// followTargetPath/hunk track the last auto-follow target for resize recalculation.
	followTargetPath string
	followTargetHunk int
	Notification    string
	notificationSeq int

	OverlayOpen      bool
	OverlayCursor    int
	OverlaySnapshot  []internal.FileDiff
	OverlayFollowWas bool
	PendingUpdate    *FilesUpdatedMsg
	// resetPending is true between first and second r press.
	resetPending bool
	// resetInFlight is true while an async manual reset command is running.
	resetInFlight bool
	// ResetBaseline resets the session baseline asynchronously from a tea.Cmd.
	ResetBaseline func() (string, []internal.FileDiff, error)

	Width  int
	Height int

	// displayCache reuses rendered visual lines across repeated View calls.
	displayCache *displayLineCache
}

// NewModel constructs the initial UI state.
func NewModel(dirName, repoDir, baselineSHA string, files []internal.FileDiff) Model {
	return Model{
		DirName:      dirName,
		RepoDir:      repoDir,
		BaselineSHA:  baselineSHA,
		Files:        files,
		FollowOn:     true,
		hunkSigs:     buildPrevHunkSigs(files),
		highlightedHunks: make(map[string]map[int]bool),
		displayCache: newDisplayLineCache(),
	}
}

// Init starts with no asynchronous UI command.
func (m Model) Init() tea.Cmd { return nil }

// Update handles all UI events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.realignFollowTarget()
		m.clampScrollOffset()
		return m, nil
	case FilesUpdatedMsg:
		return m.handleFilesUpdated(msg)
	case ManualResetMsg:
		m.BaselineSHA = msg.NewSHA
		m.Notification = "baseline reset"
		m.resetPending = false
		m.resetInFlight = false
		m.hunkSigs = buildPrevHunkSigs(msg.Files)
		m.highlightedHunks = make(map[string]map[int]bool)
		m.lastChangedPath = ""
		m.lastHighlightedPath = ""
		m.clearFollowTarget()
		// Manual reset starts a new baseline epoch, so any queued pre-reset
		// update must be replaced rather than merged.
		if m.OverlayOpen {
			m.PendingUpdate = &FilesUpdatedMsg{
				BaselineSHA: msg.NewSHA,
				Files:       msg.Files,
			}
		} else {
			m = m.applyFilesUpdate(FilesUpdatedMsg{
				BaselineSHA: msg.NewSHA,
				Files:       msg.Files,
			})
		}
		m.notificationSeq++
		token := m.notificationSeq
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return ClearNotificationMsg{Token: token}
		})
	case ManualResetFailedMsg:
		m.resetPending = false
		m.resetInFlight = false
		m.Notification = "baseline reset failed"
		if msg.Error != "" {
			m.Notification += ": " + msg.Error
		}
		m.notificationSeq++
		token := m.notificationSeq
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return ClearNotificationMsg{Token: token}
		})
	case ResetTimeoutMsg:
		if m.resetPending {
			m.resetPending = false
			m.Notification = ""
		}
		return m, nil
	case ClearNotificationMsg:
		if msg.Token == 0 || msg.Token == m.notificationSeq {
			m.Notification = ""
		}
		return m, nil
	case tea.KeyMsg:
		if m.OverlayOpen {
			return m.handleOverlayKey(msg.String())
		}
		return m.handleKey(msg.String())
	}

	return m, nil
}

func (m Model) handleFilesUpdated(msg FilesUpdatedMsg) (tea.Model, tea.Cmd) {
	if msg.BaselineSHA != "" && msg.BaselineSHA != m.BaselineSHA {
		return m, nil
	}

	// Ignore empty repeated snapshots so the latest highlight batch persists
	// until a real diff change or manual reset advances the epoch.
	if len(msg.ChangedPaths) == 0 && reflect.DeepEqual(msg.Files, m.Files) {
		return m, nil
	}

	if m.OverlayOpen {
		if m.PendingUpdate == nil {
			m.PendingUpdate = &msg
		} else {
			m.PendingUpdate.BaselineSHA = msg.BaselineSHA
			m.PendingUpdate.Files = msg.Files
			m.PendingUpdate.ChangedPaths = append(m.PendingUpdate.ChangedPaths, msg.ChangedPaths...)
		}
		return m, nil
	}

	return m.applyFilesUpdate(msg), nil
}

func (m Model) applyFilesUpdate(msg FilesUpdatedMsg) Model {
	currentPath := ""
	if m.CurrentIdx < len(m.Files) {
		currentPath = m.Files[m.CurrentIdx].Path
	}

	m.Files = msg.Files
	m.highlightedHunks = make(map[string]map[int]bool, len(m.Files))

	for _, file := range m.Files {
		changed := changedHunkIndices(m.hunkSigs[file.Path], file.Hunks)
		if len(changed) == 0 {
			continue
		}
		hunks := make(map[int]bool, len(changed))
		for _, idx := range changed {
			hunks[idx] = true
		}
		m.highlightedHunks[file.Path] = hunks
	}

	m.lastChangedPath = lastChangedPathInFiles(msg.ChangedPaths, m.Files)
	m.lastHighlightedPath = lastHighlightedPathInFiles(msg.ChangedPaths, m.highlightedHunks, m.Files)

	if len(m.Files) == 0 {
		m.CurrentIdx = 0
		m.ScrollOffset = 0
		m.hunkSigs = make(map[string][]uint64)
		m.highlightedHunks = make(map[string]map[int]bool)
		m.lastChangedPath = ""
		m.lastHighlightedPath = ""
		m.clearFollowTarget()
		m.clampScrollOffset()
		return m
	}

	if m.FollowOn {
		if idx := fileIndexByPath(m.Files, m.lastChangedPath); idx >= 0 {
			m.setFollowTarget(idx, currentPath)
		} else {
			m.anchorCurrentPath(currentPath)
		}
	} else {
		m.anchorCurrentPath(currentPath)
	}

	m.hunkSigs = buildPrevHunkSigs(m.Files)
	m.clampScrollOffset()
	return m
}

func (m Model) handleKey(key string) (tea.Model, tea.Cmd) {
	// Cancel pending reset on any non-r key, then dispatch normally.
	if m.resetPending && key != "r" {
		m.resetPending = false
		m.Notification = ""
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "down":
		m.clearFollowTarget()
		m.ScrollOffset++
		m.clampScrollOffset()
		return m, nil
	case "up":
		m.clearFollowTarget()
		if m.ScrollOffset > 0 {
			m.ScrollOffset--
		}
		m.clampScrollOffset()
		return m, nil
	case "right":
		if len(m.Files) > 1 {
			m.clearFollowTarget()
			m.CurrentIdx = (m.CurrentIdx + 1) % len(m.Files)
			m.ScrollOffset = 0
			if m.FollowOn {
				m.FollowOn = false
			}
		}
		return m, nil
	case "left":
		if len(m.Files) > 1 {
			m.clearFollowTarget()
			m.CurrentIdx = (m.CurrentIdx - 1 + len(m.Files)) % len(m.Files)
			m.ScrollOffset = 0
			if m.FollowOn {
				m.FollowOn = false
			}
		}
		return m, nil
	case "f":
		m.FollowOn = !m.FollowOn
		if m.FollowOn {
			m.selectLatestPendingFile()
		}
		return m, nil
	case "r":
		if m.resetInFlight {
			return m, nil
		}
		if m.resetPending {
			// Second press dispatches async reset without blocking Update.
			m.resetPending = false
			m.resetInFlight = true
			m.Notification = ""
			resetFn := m.ResetBaseline
			if resetFn == nil {
				m.resetInFlight = false
				return m, nil
			}
			return m, func() tea.Msg {
				newSHA, newFiles, err := resetFn()
				if err != nil {
					return ManualResetFailedMsg{Error: err.Error()}
				}
				return ManualResetMsg{NewSHA: newSHA, Files: newFiles}
			}
		}
		if m.ResetBaseline == nil {
			return m, nil
		}
		m.resetPending = true
		m.Notification = "press r to reset baseline"
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return ResetTimeoutMsg{}
		})
	case "tab":
		m.OverlayOpen = true
		m.OverlayCursor = m.CurrentIdx
		m.OverlayFollowWas = m.FollowOn
		m.OverlaySnapshot = make([]internal.FileDiff, len(m.Files))
		copy(m.OverlaySnapshot, m.Files)
		m.PendingUpdate = nil
		return m, nil
	}

	return m, nil
}

func (m Model) handleOverlayKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "down":
		if m.OverlayCursor < len(m.OverlaySnapshot)-1 {
			m.OverlayCursor++
		}
		return m, nil
	case "up":
		if m.OverlayCursor > 0 {
			m.OverlayCursor--
		}
		return m, nil
	case "enter":
		if m.OverlayCursor < len(m.OverlaySnapshot) {
			m.clearFollowTarget()
			m.CurrentIdx = m.OverlayCursor
			m.ScrollOffset = 0
			m.FollowOn = false
		}
		return m.closeOverlay(), nil
	case "tab", "esc":
		m.FollowOn = m.OverlayFollowWas
		return m.closeOverlay(), nil
	case "f":
		m.FollowOn = true
		m.selectLatestPendingFile()
		return m.closeOverlay(), nil
	}

	return m, nil
}

func (m Model) closeOverlay() Model {
	m.OverlayOpen = false
	m.OverlaySnapshot = nil
	if m.PendingUpdate != nil && (m.PendingUpdate.BaselineSHA == "" || m.PendingUpdate.BaselineSHA == m.BaselineSHA) {
		m = m.applyFilesUpdate(*m.PendingUpdate)
	}
	m.PendingUpdate = nil
	return m
}

func (m *Model) selectLatestPendingFile() {
	currentPath := ""
	if m.CurrentIdx >= 0 && m.CurrentIdx < len(m.Files) {
		currentPath = m.Files[m.CurrentIdx].Path
	}

	if idx := fileIndexByPath(m.Files, m.lastHighlightedPath); idx >= 0 {
		m.setFollowTarget(idx, currentPath)
		return
	}
	if idx := fileIndexByPath(m.Files, m.lastChangedPath); idx >= 0 {
		m.setFollowTarget(idx, currentPath)
		return
	}
	m.clampScrollOffset()
}

// setFollowTarget applies follow-mode scrolling rules for one selected file.
func (m *Model) setFollowTarget(targetIdx int, currentPath string) {
	file := &m.Files[targetIdx]
	m.CurrentIdx = targetIdx

	if hunkIdx, ok := maxHighlightedHunkIndex(m.highlightedHunks[file.Path]); ok {
		m.ScrollOffset = hunkVisualOffset(file, hunkIdx, m.Width)
		m.followTargetPath = file.Path
		m.followTargetHunk = hunkIdx
	} else {
		_ = currentPath
		m.clearFollowTarget()
		m.ScrollOffset = 0
	}

	m.clampScrollOffset()
}

// clearFollowTarget drops width-sensitive auto-follow alignment state.
func (m *Model) clearFollowTarget() {
	m.followTargetPath = ""
	m.followTargetHunk = -1
}

func (m *Model) anchorCurrentPath(currentPath string) {
	found := false
	for i, file := range m.Files {
		if file.Path == currentPath {
			m.CurrentIdx = i
			found = true
			break
		}
	}
	if !found && len(m.Files) > 0 {
		if m.CurrentIdx >= len(m.Files) {
			m.CurrentIdx = len(m.Files) - 1
		}
		m.ScrollOffset = 0
	}
}

// realignFollowTarget recomputes the stored follow target after width changes.
func (m *Model) realignFollowTarget() {
	if !m.FollowOn || m.followTargetPath == "" || m.followTargetHunk < 0 {
		return
	}
	if m.CurrentIdx < 0 || m.CurrentIdx >= len(m.Files) {
		m.clearFollowTarget()
		return
	}

	file := &m.Files[m.CurrentIdx]
	if file.Path != m.followTargetPath || m.followTargetHunk >= len(file.Hunks) {
		m.clearFollowTarget()
		return
	}

	m.ScrollOffset = hunkVisualOffset(file, m.followTargetHunk, m.Width)
}

// buildPrevHunkSigs snapshots current file hunks for the next follow comparison.
func buildPrevHunkSigs(files []internal.FileDiff) map[string][]uint64 {
	prevHunkSigs := make(map[string][]uint64, len(files))
	for _, file := range files {
		prevHunkSigs[file.Path] = hunkFingerprints(file.Hunks)
	}
	return prevHunkSigs
}

func fileIndexByPath(files []internal.FileDiff, path string) int {
	if path == "" {
		return -1
	}
	for i, file := range files {
		if file.Path == path {
			return i
		}
	}
	return -1
}

func lastChangedPathInFiles(changedPaths []string, files []internal.FileDiff) string {
	for i := len(changedPaths) - 1; i >= 0; i-- {
		if fileIndexByPath(files, changedPaths[i]) >= 0 {
			return changedPaths[i]
		}
	}
	return ""
}

func lastHighlightedPathInFiles(changedPaths []string, highlighted map[string]map[int]bool, files []internal.FileDiff) string {
	for i := len(changedPaths) - 1; i >= 0; i-- {
		path := changedPaths[i]
		if fileIndexByPath(files, path) < 0 {
			continue
		}
		if len(highlighted[path]) > 0 {
			return path
		}
	}
	return ""
}

func maxHighlightedHunkIndex(highlighted map[int]bool) (int, bool) {
	if len(highlighted) == 0 {
		return -1, false
	}
	maxIdx := -1
	for idx := range highlighted {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	return maxIdx, true
}

// clampScrollOffset keeps scroll state within the current diff viewport bounds.
func (m *Model) clampScrollOffset() {
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
		return
	}

	maxOffset := m.maxScrollOffset()
	if m.ScrollOffset > maxOffset {
		m.ScrollOffset = maxOffset
	}
}

// maxScrollOffset returns the furthest scroll position that can show diff content.
func (m Model) maxScrollOffset() int {
	if len(m.Files) == 0 || m.CurrentIdx < 0 || m.CurrentIdx >= len(m.Files) {
		return 0
	}

	diffHeight := max(0, m.Height-2)
	if diffHeight == 0 {
		return 0
	}

	totalLines := countVisualDiffLines(&m.Files[m.CurrentIdx], m.Width)
	return max(0, totalLines-diffHeight)
}

// View renders header, content, and footer into the terminal viewport.
func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}

	diffHeight := max(0, m.Height-2)
	header := RenderHeader(m.DirName, m.Files, m.CurrentIdx, m.Width)
	footer := RenderFooter(m.FollowOn, m.Notification, m.Width)

	var content string
	if m.OverlayOpen {
		content = RenderOverlay(m.OverlaySnapshot, m.OverlayCursor, diffHeight, m.Width)
	} else if len(m.Files) > 0 && m.CurrentIdx < len(m.Files) {
		highlightSet := m.highlightedHunks[m.Files[m.CurrentIdx].Path]
		lines := m.displayCache.get(&m.Files[m.CurrentIdx], m.Width, highlightSet, func() []string {
			return diffDisplayLines(&m.Files[m.CurrentIdx], m.Width, highlightSet)
		})
		content = renderDisplayLines(lines, m.ScrollOffset, diffHeight)
	}

	contentLines := strings.Count(content, "\n") + 1
	if content == "" {
		contentLines = 0
	}
	padding := ""
	if contentLines < diffHeight {
		paddingLines := diffHeight - contentLines
		if contentLines == 0 && paddingLines > 0 {
			paddingLines--
		}
		padding = strings.Repeat("\n", paddingLines)
	}

	return header + "\n" + content + padding + "\n" + footer
}
