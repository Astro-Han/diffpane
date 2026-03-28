package ui

import "github.com/charmbracelet/lipgloss"

var (
	// ColorAdd colors added lines and counters.
	ColorAdd = lipgloss.AdaptiveColor{Light: "#22863a", Dark: "#56d364"}
	// ColorDel colors deleted lines and counters.
	ColorDel = lipgloss.AdaptiveColor{Light: "#cb2431", Dark: "#f85149"}
	// ColorDim colors neutral metadata such as headers and footer text.
	ColorDim = lipgloss.AdaptiveColor{Light: "#6a737d", Dark: "#8b949e"}

	// StyleAdd renders added content.
	StyleAdd = lipgloss.NewStyle().Foreground(ColorAdd)
	// StyleDel renders deleted content.
	StyleDel = lipgloss.NewStyle().Foreground(ColorDel)
	// StyleDim renders neutral metadata.
	StyleDim = lipgloss.NewStyle().Foreground(ColorDim)
)
