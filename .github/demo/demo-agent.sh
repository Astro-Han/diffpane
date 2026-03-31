#!/bin/bash
# Simulates Codex CLI terminal output for the demo GIF.
# Format matches real Codex CLI v0.117.0 interactive mode output.
# Runs in the LEFT tmux pane. Makes real file changes so diffpane picks them up.

DEMO_DIR="/tmp/diffpane-demo"

# ANSI colors matching Codex CLI palette
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

# Codex uses a dimmed bullet for tool/action markers
BULLET="${DIM}\xe2\x80\xa2${RESET}"  # •
PROMPT="${BOLD}\xe2\x80\xba${RESET}" # ›
CORNER="${DIM}\xe2\x94\x94${RESET}"  # └

# Simulate typing (character by character)
type_out() {
  local text="$1"
  for ((i = 0; i < ${#text}; i++)); do
    printf '%s' "${text:$i:1}"
    sleep 0.03
  done
}

# Horizontal separator matching Codex style
separator() {
  local cols
  cols=$(tput cols 2>/dev/null || echo 60)
  printf '%s' "${DIM}"
  printf '%*s' "$cols" '' | tr ' ' '─'
  printf '%s\n' "${RESET}"
}

# Wait for tmux setup (~2s already elapsed before recording starts)
sleep 3

# --- Codex prompt with user task ---
printf '%s ' "${PROMPT}"
type_out "Add a detail parameter to health endpoint, and a version to config"
echo
echo

sleep 0.8

# --- Phase 1: reading files ---
echo -e "${BULLET} Reading project files to understand current structure."
echo
sleep 0.4

echo -e "  ${CORNER} Read ${DIM}internal/handler/health.go${RESET}"
sleep 0.6
echo -e "  ${CORNER} Read ${DIM}internal/config/config.go${RESET}"
echo
sleep 0.8

# --- Phase 2: first edit (health.go) ---
echo -e "${BULLET} Edited ${BOLD}internal/handler/health.go${RESET}"
echo -e "  ${CORNER} ${DIM}+8 -1${RESET}"

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

echo
sleep 1.5

# --- Phase 3: second edit (config.go) ---
echo -e "${BULLET} Edited ${BOLD}internal/config/config.go${RESET}"
echo -e "  ${CORNER} ${DIM}+1 -0${RESET}"

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

echo
sleep 1

# --- Phase 4: verification ---
echo -e "${BULLET} Ran ${DIM}git diff --stat${RESET}"
echo -e "  ${CORNER} ${DIM}2 files changed, 9 insertions(+), 1 deletion(-)${RESET}"
echo

sleep 0.5
separator
echo

# --- Completion summary ---
echo -e "${BULLET} Added ${BOLD}detail${RESET} query parameter to health handler and"
echo -e "  ${BOLD}version${RESET} field to config map."
echo

# Keep alive so tmux pane doesn't close
sleep 60
