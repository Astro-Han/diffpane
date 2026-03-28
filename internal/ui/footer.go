package ui

import "fmt"

// RenderFooter renders the one-line footer or a transient notification.
func RenderFooter(followOn bool, notification string) string {
	if notification != "" {
		return StyleAdd.Render(notification)
	}

	status := "on"
	if !followOn {
		status = "off"
	}
	return StyleDim.Render(fmt.Sprintf("q quit · n/p files · f follow: %s · tab list", status))
}
