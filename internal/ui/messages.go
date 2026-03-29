package ui

import "github.com/Astro-Han/diffpane/internal"

// FilesUpdatedMsg notifies the UI that the computed file list changed.
type FilesUpdatedMsg struct {
	BaselineSHA  string
	Files        []internal.FileDiff
	ChangedPaths []string
}

// ManualResetMsg carries the result of a manual baseline reset (r key).
type ManualResetMsg struct {
	NewSHA string
	Files  []internal.FileDiff
}

// ManualResetFailedMsg carries a reset error back into the Update loop.
type ManualResetFailedMsg struct {
	Error string
}

// ClearNotificationMsg clears a temporary footer notification.
// Token must match the active notification generation; stale clears are ignored.
type ClearNotificationMsg struct {
	Token int
}

// ResetTimeoutMsg cancels a pending manual baseline reset after timeout.
type ResetTimeoutMsg struct{}
