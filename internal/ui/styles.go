package ui

import "github.com/charmbracelet/lipgloss"

var (
	// ColorAdd and ColorDel are the only semantic colors used in V1.
	ColorAdd = lipgloss.AdaptiveColor{Light: "#22863a", Dark: "#56d364"}
	ColorDel = lipgloss.AdaptiveColor{Light: "#cb2431", Dark: "#f85149"}
	ColorDim = lipgloss.AdaptiveColor{Light: "#6a737d", Dark: "#8b949e"}

	StyleAdd = lipgloss.NewStyle().Foreground(ColorAdd)
	StyleDel = lipgloss.NewStyle().Foreground(ColorDel)
	StyleDim = lipgloss.NewStyle().Foreground(ColorDim)
)
