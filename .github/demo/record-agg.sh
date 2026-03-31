#!/bin/bash
# Automated demo GIF recording with asciinema + agg.
# Uses a simulated Codex CLI session (demo-agent.sh) for deterministic output.
#
# Prerequisites: brew install asciinema agg gifsicle tmux
# Usage: bash .github/demo/record-agg.sh
# Output: .github/demo.gif

set -e

PROJ_DIR="/Users/yuhan/workspace/dev/diffpane"
SCRIPT_DIR="$PROJ_DIR/.github/demo"
DEMO_DIR="/tmp/diffpane-demo"
CAST_FILE="/tmp/diffpane-demo.cast"
GIF_RAW="/tmp/diffpane-demo-raw.gif"
GIF_FINAL="$PROJ_DIR/.github/demo.gif"

# --- 1. Build diffpane ---
echo "==> Building diffpane..."
go build -o /tmp/diffpane-bin "$PROJ_DIR"

# --- 2. Setup demo repo ---
echo "==> Setting up demo repo..."
bash "$SCRIPT_DIR/demo-simulate.sh"

# --- 3. Clean stale tmux session ---
tmux kill-session -t demo 2>/dev/null || true

# --- 4. Start tmux session with split panes ---
# Left pane: simulated Codex CLI output
tmux -f "$SCRIPT_DIR/tmux-demo.conf" new-session -d -s demo \
  "bash $SCRIPT_DIR/demo-agent.sh"

sleep 0.5

# Right pane: diffpane watching the demo repo
tmux split-window -h -t demo \
  "cd $DEMO_DIR && DIFFPANE_THEME=dark /tmp/diffpane-bin"

tmux select-layout -t demo even-horizontal
sleep 1.5

# --- 5. Background controller ---
(
  # demo-agent.sh takes ~12s for all changes, then sleeps
  sleep 16

  # Show file list overlay
  tmux send-keys -t demo:0.1 Tab
  sleep 2

  # Close overlay
  tmux send-keys -t demo:0.1 Escape
  sleep 1

  # End recording
  tmux kill-session -t demo
) &
CONTROLLER_PID=$!

# --- 6. Record with asciinema ---
echo "==> Recording..."
asciinema rec \
  --headless \
  --window-size 120x40 \
  --idle-time-limit 3 \
  -c "tmux attach -t demo" \
  --overwrite \
  "$CAST_FILE" || true

wait $CONTROLLER_PID 2>/dev/null || true
tmux kill-session -t demo 2>/dev/null || true

# --- 7. Check recording ---
EVENT_COUNT=$(grep -c '^\[' "$CAST_FILE" || echo 0)
echo "==> Recorded $EVENT_COUNT events"
if [ "$EVENT_COUNT" -lt 20 ]; then
  echo "WARNING: Very few events captured."
  echo "Cast file: $CAST_FILE"
  exit 1
fi

# --- 8. Convert to GIF ---
echo "==> Converting to GIF..."
agg \
  --font-size 28 \
  --theme github-dark \
  --fps-cap 20 \
  --idle-time-limit 2 \
  --speed 1 \
  --last-frame-duration 0.5 \
  "$CAST_FILE" "$GIF_RAW"

# --- 9. Trim + compress ---
echo "==> Trimming + compressing..."
FRAME_COUNT=$(gifsicle --info "$GIF_RAW" 2>&1 | head -1 | grep -o '[0-9]* images' | grep -o '[0-9]*')
LAST=$((FRAME_COUNT - 2))
gifsicle "$GIF_RAW" "#0-${LAST}" -O3 --lossy=30 -k 256 -o "$GIF_FINAL"

# --- 10. Report ---
echo "==> Done!"
ls -lh "$GIF_FINAL"
file "$GIF_FINAL"
