# diffpane

Real-time TUI diff viewer for AI coding agents.

Split your terminal. Left: your AI agent. Right: `diffpane` showing every change as it happens.

## Features

- **Live diff** — watches your git worktree and shows changes in real time
- **Session baseline** — records HEAD at startup, auto-resets on commit
- **Follow mode** — auto-jumps to the latest changed file (on by default)
- **Zero dependencies** — single binary, `brew install` and go

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

## Keys

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll diff |
| `n` / `p` | Next / previous file |
| `f` | Toggle follow mode |
| `Tab` | File list |
| `q` | Quit |

## How it works

`diffpane` records your current `git HEAD` when it starts (the "baseline"). It watches for file changes and shows a live unified diff of everything that changed since the baseline. When you `git commit`, the baseline auto-resets so you only see new changes.

## Status

Under development. Not yet functional.

## License

MIT
