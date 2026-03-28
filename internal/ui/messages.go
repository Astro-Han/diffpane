package ui

import "github.com/Astro-Han/diffpane/internal"

// FilesUpdatedMsg notifies the UI that the computed file list changed.
type FilesUpdatedMsg struct {
	BaselineSHA  string
	Files        []internal.FileDiff
	ChangedPaths []string
}

// BaselineResetMsg notifies the UI that baseline SHA changed.
type BaselineResetMsg struct {
	NewSHA string
}

// ClearNotificationMsg clears a temporary footer notification.
type ClearNotificationMsg struct{}
