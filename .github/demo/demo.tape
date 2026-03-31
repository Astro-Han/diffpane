# diffpane split-screen demo GIF recording
#
# Prerequisites:
#   brew install tmux   (if not installed)
#
# Steps:
#   1. bash .github/demo/demo-simulate.sh     (create test repo)
#   2. go build -o /tmp/diffpane-bin           (build binary)
#   3. tmux kill-session -t demo 2>/dev/null   (clean stale session)
#   4. cd .github/demo && vhs demo.tape        (record)
#
# Output: .github/demo.gif

Output ../demo.gif

Set Shell "zsh"
Set Width 1200
Set Height 700
Set FontSize 15
Set FontFamily "JetBrains Mono"
Set Theme {"name":"Claude Light","black":"#0a0a0a","red":"#A84B3A","green":"#2E7C4C","yellow":"#8A6220","blue":"#184e95","purple":"#882b4d","cyan":"#066049","white":"#c3c2b7","brightBlack":"#51504d","brightRed":"#e34a4a","brightGreen":"#639900","brightYellow":"#b87700","brightBlue":"#3886e5","brightPurple":"#d55382","brightCyan":"#199f70","brightWhite":"#e2e1da","background":"#faf9f5","foreground":"#141413","selectionBackground":"#f0eee6","cursorColor":"#d97757"}
Set Framerate 24
Set TypingSpeed 50ms
Set CursorBlink false

# --- Hidden setup: instant typing so agent sleep timing is predictable ---
Hide

Type@1ms "tmux -f /Users/yuhan/workspace/dev/diffpane/.github/demo/tmux-demo.conf new-session -d -s demo 'bash /Users/yuhan/workspace/dev/diffpane/.github/demo/demo-agent.sh'"
Enter
Sleep 500ms

Type@1ms "tmux split-window -h -t demo 'cd /tmp/diffpane-demo && DIFFPANE_THEME=light /tmp/diffpane-bin'"
Enter
Sleep 300ms

Type@1ms "tmux select-layout -t demo even-horizontal"
Enter
Sleep 300ms

Type@1ms "tmux attach -t demo"
Enter
Sleep 2s

# --- Visible recording ---
Show

# ~1.5s empty state, then agent banner + two changes + diffpane reacting
Sleep 11s

# Show file list overlay
Tab
Sleep 1.5s

# Close overlay
Escape
Sleep 0.5s

# --- Hidden cleanup ---
Hide

Type "q"
Sleep 300ms
Type@1ms "tmux kill-session -t demo 2>/dev/null"
Enter
