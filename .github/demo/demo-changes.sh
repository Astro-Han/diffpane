#!/bin/bash
# Background file changes for the diffpane demo GIF.
# Started inside the tape, runs in background while diffpane is open.

DEMO_DIR="/tmp/diffpane-demo"

# Wait for diffpane to start + show empty state (~2s visible)
sleep 5

# Change 1: add detail param to health handler
cat >"$DEMO_DIR/internal/handler/health.go" <<'EOF'
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Health responds with service status.
// When detail=true, returns JSON with uptime info.
func Health(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("detail") == "true" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"uptime": "12h",
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}
EOF

sleep 3

# Change 2: add version to config
cat >"$DEMO_DIR/internal/config/config.go" <<'EOF'
package config

// Config holds application settings.
var Config = map[string]string{
	"port":    "8080",
	"host":    "localhost",
	"version": "0.1.0",
}
EOF
