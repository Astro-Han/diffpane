#!/bin/bash
# Sets up a test git repo for the diffpane demo GIF.
# Run BEFORE vhs: bash demo-simulate.sh

set -e

DEMO_DIR="/tmp/diffpane-demo"

rm -rf "$DEMO_DIR"
mkdir -p "$DEMO_DIR/internal/handler" "$DEMO_DIR/internal/config"
cd "$DEMO_DIR"

git init -q
git config user.email "demo@example.com"
git config user.name "Demo"

cat >internal/handler/health.go <<'EOF'
package handler

import (
	"fmt"
	"net/http"
)

// Health responds with service status.
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}
EOF

cat >internal/config/config.go <<'EOF'
package config

// Config holds application settings.
var Config = map[string]string{
	"port": "8080",
	"host": "localhost",
}
EOF

git add -A
git commit -q -m "init"

echo "Demo repo ready at $DEMO_DIR"
