# diffpane demo GIF recording
#
# Steps:
#   1. bash .github/demo/demo-simulate.sh   (create test repo)
#   2. go build -o /tmp/diffpane-bin        (build binary)
#   3. cd .github/demo && vhs demo.tape     (record)
#
# Output: .github/demo.gif

Output ../demo.gif

Set Shell "zsh"
Set Width 720
Set Height 800
Set FontSize 16
Set FontFamily "JetBrains Mono"
Set Theme "Catppuccin Mocha"
Set Framerate 24
Set TypingSpeed 50ms
Set CursorBlink false

# Setup: cd, start background changes, launch diffpane in alt screen.
Hide
Type "cd /tmp/diffpane-demo"
Enter
Sleep 500ms
Type "(bash /Users/yuhan/workspace/dev/diffpane/.github/demo/demo-changes.sh &)"
Enter
Sleep 500ms
Type "DIFFPANE_THEME=dark /tmp/diffpane-bin"
Enter
Sleep 2s
Show

# Empty state: "watching", no changes yet (~2s)
Sleep 3s

# Change 1: internal/handler/health.go diff, syntax highlighted
Sleep 3s

# Change 2: internal/config/config.go, follow auto-jumped
Sleep 2s

# Open file list overlay: shows both changed files
Tab
Sleep 2s

# Quit
Type "q"
