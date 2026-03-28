package ui

import (
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
	NewCount        int
	NewFiles        map[string]bool
	LastChangedPath string
	Notification    string

	OverlayOpen      bool
	OverlayCursor    int
	OverlaySnapshot  []internal.FileDiff
	OverlayFollowWas bool
	PendingUpdate    *FilesUpdatedMsg

	Width  int
	Height int
}

// NewModel constructs the initial UI state.
func NewModel(dirName, repoDir, baselineSHA string, files []internal.FileDiff) Model {
	return Model{
		DirName:     dirName,
		RepoDir:     repoDir,
		BaselineSHA: baselineSHA,
		Files:       files,
		FollowOn:    true,
		NewFiles:    make(map[string]bool),
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
		m.clampScrollOffset()
		return m, nil
	case FilesUpdatedMsg:
		return m.handleFilesUpdated(msg)
	case BaselineResetMsg:
		m.BaselineSHA = msg.NewSHA
		m.Notification = "baseline reset"
		m.NewCount = 0
		m.NewFiles = make(map[string]bool)
		m.LastChangedPath = ""
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return ClearNotificationMsg{}
		})
	case ClearNotificationMsg:
		m.Notification = ""
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

	if m.FollowOn && len(m.Files) > 0 {
		// Try to follow the most recently changed file that still exists in the list.
		target := -1
	findTarget:
		for i := len(msg.ChangedPaths) - 1; i >= 0; i-- {
			for j, file := range m.Files {
				if file.Path == msg.ChangedPaths[i] {
					target = j
					break findTarget
				}
			}
		}
		if target >= 0 {
			m.CurrentIdx = target
			m.ScrollOffset = 0
		} else {
			// No changed file matched; try to stay on the current file.
			anchored := false
			for i, file := range m.Files {
				if file.Path == currentPath {
					m.CurrentIdx = i
					anchored = true
					break
				}
			}
			if !anchored {
				// Current file disappeared; clamp to adjacent position.
				if m.CurrentIdx >= len(m.Files) {
					m.CurrentIdx = len(m.Files) - 1
				}
				m.ScrollOffset = 0
			}
		}
		m.NewFiles = make(map[string]bool)
		m.LastChangedPath = ""
		m.NewCount = 0
	} else if !m.FollowOn {
		presentPaths := make(map[string]bool, len(m.Files))
		for _, file := range m.Files {
			presentPaths[file.Path] = true
		}
		for path := range m.NewFiles {
			if !presentPaths[path] || path == currentPath {
				delete(m.NewFiles, path)
			}
		}
		for _, changedPath := range msg.ChangedPaths {
			if changedPath != currentPath && presentPaths[changedPath] {
				m.NewFiles[changedPath] = true
				m.LastChangedPath = changedPath
			}
		}
		if m.LastChangedPath != "" && !m.NewFiles[m.LastChangedPath] {
			m.LastChangedPath = ""
		}
		m.NewCount = len(m.NewFiles)

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

	if len(m.Files) == 0 {
		m.CurrentIdx = 0
		m.ScrollOffset = 0
		m.NewFiles = make(map[string]bool)
		m.LastChangedPath = ""
		m.NewCount = 0
	}

	m.clampScrollOffset()
	return m
}

func (m Model) handleKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		m.ScrollOffset++
		m.clampScrollOffset()
		return m, nil
	case "k", "up":
		if m.ScrollOffset > 0 {
			m.ScrollOffset--
		}
		m.clampScrollOffset()
		return m, nil
	case "n":
		if len(m.Files) > 1 {
			m.CurrentIdx = (m.CurrentIdx + 1) % len(m.Files)
			m.ScrollOffset = 0
			if m.FollowOn {
				m.FollowOn = false
			}
		}
		return m, nil
	case "p":
		if len(m.Files) > 1 {
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
			m.NewCount = 0
			m.NewFiles = make(map[string]bool)
		}
		return m, nil
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
	case "j", "down", "n":
		if m.OverlayCursor < len(m.OverlaySnapshot)-1 {
			m.OverlayCursor++
		}
		return m, nil
	case "k", "up", "p":
		if m.OverlayCursor > 0 {
			m.OverlayCursor--
		}
		return m, nil
	case "enter":
		if m.OverlayCursor < len(m.OverlaySnapshot) {
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
		m.NewCount = 0
		m.NewFiles = make(map[string]bool)
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
	if m.LastChangedPath != "" {
		for i, file := range m.Files {
			if file.Path == m.LastChangedPath {
				m.CurrentIdx = i
				m.ScrollOffset = 0
				return
			}
		}
	}

	for i := len(m.Files) - 1; i >= 0; i-- {
		if m.NewFiles[m.Files[i].Path] {
			m.CurrentIdx = i
			m.ScrollOffset = 0
			return
		}
	}
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
	header := RenderHeader(m.DirName, m.Files, m.CurrentIdx, m.NewCount, m.Width)
	footer := RenderFooter(m.FollowOn, m.Notification, m.Width)

	var content string
	if m.OverlayOpen {
		content = RenderOverlay(m.OverlaySnapshot, m.OverlayCursor, diffHeight, m.Width)
	} else if len(m.Files) > 0 && m.CurrentIdx < len(m.Files) {
		content = RenderDiffView(&m.Files[m.CurrentIdx], m.ScrollOffset, m.Width, diffHeight)
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
