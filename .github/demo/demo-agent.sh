#!/bin/bash
# Simulates Claude Code terminal output for the demo GIF.
# Runs in the LEFT tmux pane. Makes real file changes so diffpane picks them up.

DEMO_DIR="/tmp/diffpane-demo"

# ANSI colors
BOLD='\033[1m'
DIM='\033[2m'
GREEN='\033[32m'
PURPLE='\033[35m'
RESET='\033[0m'

# Simulate typing (character by character)
type_out() {
  local text="$1"
  for ((i = 0; i < ${#text}; i++)); do
    printf '%s' "${text:$i:1}"
    sleep 0.03
  done
  echo
}

# Wait for VHS hidden setup + Show (~4s setup, then ~1.5s empty visible)
sleep 5

# Claude Code banner
echo -e "${BOLD}${PURPLE}  ╭────────────────────────────────────────╮${RESET}"
echo -e "${BOLD}${PURPLE}  │${RESET}  ${PURPLE}✻${RESET}${BOLD}  Welcome to Claude Code!              ${PURPLE}│${RESET}"
echo -e "${BOLD}${PURPLE}  │${RESET}     /help for help                      ${PURPLE}│${RESET}"
echo -e "${BOLD}${PURPLE}  ╰────────────────────────────────────────╯${RESET}"
echo
sleep 0.8

# --- Change 1: add detail param to health handler ---

printf '%b> %b' "$BOLD" "$RESET"
type_out "Add a detail parameter to the health endpoint"
echo
sleep 0.3

echo -e "  ${PURPLE}⏺${RESET} Adding a ${BOLD}detail${RESET} query parameter with JSON response."
echo
sleep 0.5

echo -e "    ${GREEN}Edit${RESET} ${DIM}internal/handler/health.go${RESET}"

# Make actual file change (triggers diffpane)
cat >"$DEMO_DIR/internal/handler/health.go" <<'GOEOF'
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
GOEOF

sleep 1
echo -e "    ${GREEN}✓${RESET} Changes applied"
echo
sleep 1.5

# --- Change 2: add version to config ---

printf '%b> %b' "$BOLD" "$RESET"
type_out "Add version to the config"
echo
sleep 0.3

echo -e "  ${PURPLE}⏺${RESET} Adding a version field to the config map."
echo
sleep 0.5

echo -e "    ${GREEN}Edit${RESET} ${DIM}internal/config/config.go${RESET}"

# Make actual file change (triggers diffpane)
cat >"$DEMO_DIR/internal/config/config.go" <<'GOEOF'
package config

// Config holds application settings.
var Config = map[string]string{
	"port":    "8080",
	"host":    "localhost",
	"version": "0.1.0",
}
GOEOF

sleep 1
echo -e "    ${GREEN}✓${RESET} Changes applied"

# Keep alive so tmux pane doesn't close
sleep 60
