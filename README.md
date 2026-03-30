# diffpane

Real-time TUI diff viewer for AI coding agents.

Watch what your AI agent is changing, in real-time, right next to your terminal.

## Install

```bash
brew install Astro-Han/tap/diffpane
```

Or with Go:

```bash
go install github.com/Astro-Han/diffpane@latest
```

Pre-built binaries are also available on the [Releases](https://github.com/Astro-Han/diffpane/releases) page.

## Usage

```bash
cd your-project
diffpane
```

Split your terminal. Left: run your AI agent. Right: `diffpane` shows what changed.

diffpane records your current git HEAD when it starts (the "baseline"). It watches for file changes and shows you a live unified diff of everything that changed since the baseline. When you `git commit`, the baseline auto-resets so you only see new changes.

## Keys

| Key | Action |
|-----|--------|
| `↑`/`↓` | Scroll diff |
| `←`/`→` | Next/prev file |
| `f` | Toggle follow mode |
| `Tab` | File list |
| `q` | Quit |

## Requirements

- macOS (darwin/amd64 or darwin/arm64)
- Git

## License

MIT
