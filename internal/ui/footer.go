package ui

import "fmt"

// RenderFooter renders the one-line footer or a transient notification.
func RenderFooter(followOn bool, notification string, width int) string {
	if notification != "" {
		return clampInlineWidth(StyleAdd.Render(notification), width)
	}

	status := "on"
	if !followOn {
		status = "off"
	}
	return clampInlineWidth(StyleDim.Render(fmt.Sprintf("q quit | ←/→ files | f follow: %s | r reset | tab list", status)), width)
}
