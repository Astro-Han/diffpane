# diffpane

Real-time TUI diff viewer for AI coding agents.

Watch what your AI agent is changing, in real-time, right next to your terminal.

## Install

```bash
brew install Astro-Han/tap/diffpane
```

Or download from [Releases](https://github.com/Astro-Han/diffpane/releases).

## Usage

```bash
cd your-project
diffpane
```

Split your terminal. Left: run your AI agent. Right: `diffpane` shows what changed.

## Keys

| Key | Action |
|-----|--------|
| `j`/`k` | Scroll diff |
| `n`/`p` | Next/prev file |
| `f` | Toggle follow mode |
| `Tab` | File list |
| `q` | Quit |

## How it works

diffpane records your current git HEAD when it starts (the "baseline"). It watches for file changes and shows you a live diff of everything that changed since the baseline. When you `git commit`, the baseline auto-resets so you only see new changes.

## License

MIT
