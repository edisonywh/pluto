# pluto

A vim-style TUI for reviewing Claude Code plans before they execute.

Pluto hooks into Claude Code's `PreToolUse` permission system to intercept `ExitPlanMode` calls. When Claude presents a plan, pluto opens a new terminal window where you can read, annotate, and approve or reject it — before any code is written.

## How It Works

1. Claude Code calls `ExitPlanMode` and pipes a JSON payload to pluto via stdin.
2. Pluto writes the plan to a temp file and opens a new terminal window running itself in `--review` mode.
3. The review window shows the plan in a scrollable, vim-navigable TUI.
4. You approve or reject. If you reject, any annotations you added are passed back to Claude as feedback.
5. The hook process reads the result and returns the decision to Claude Code.

On repeated plan revisions, pluto diffs the new plan against the previous one — press `D` to toggle the diff view.

## Prerequisites

- Go 1.23+
- macOS (terminal spawning uses AppleScript / `open`)
- One of: [Ghostty](https://ghostty.org), [iTerm2](https://iterm2.com), or Terminal.app

Terminal preference is detected automatically (Ghostty → iTerm2 → Terminal.app).

## Installation

```sh
go install github.com/edisonywh/pluto@latest
```

Or build from source:

```sh
git clone https://github.com/edisonywh/pluto
cd pluto
go build -o pluto .
# Move the binary somewhere on your PATH, e.g.:
mv pluto /usr/local/bin/pluto
```

## Setup

Add pluto as a `PreToolUse` hook in your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "ExitPlanMode",
        "hooks": [
          {
            "type": "command",
            "command": "pluto"
          }
        ]
      }
    ]
  }
}
```

Claude Code will pipe the plan JSON to pluto on stdin whenever it tries to exit plan mode.

## Usage

When Claude presents a plan, a new terminal window opens automatically. Navigate and annotate with vim-style keys:

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Down one line |
| `k` / `↑` | Up one line |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `ctrl+d` | Half page down |
| `ctrl+u` | Half page up |
| `w` | Next non-blank line |
| `b` | Previous non-blank line |
| `}` | Next paragraph |
| `{` | Previous paragraph |

### Selection & Annotation

| Key | Action |
|-----|--------|
| `V` | Visual line select (toggle) |
| `v` | Visual char select |
| `h` / `l` | Char left / right (in visual char mode) |
| `c` | Add a comment to selection |
| `x` | Mark selection as deleted |
| `r` | Mark selection as replaced |
| `esc` | Cancel selection |
| `enter` | Confirm annotation |

### Decisions

| Key | Action |
|-----|--------|
| `A` | Approve plan (Claude proceeds) |
| `R` | Reject plan (Claude revises, annotations sent as feedback) |
| `D` | Toggle diff view (shows changes from previous plan revision) |
| `?` | Toggle full help |

## Architecture

```
pluto
├── main.go                  # Entry point; dispatches to hook or review mode
├── internal/
│   ├── hook/                # Claude Code hook JSON encoding (allow/deny)
│   ├── tui/                 # Bubbletea TUI model and keymap
│   ├── annotation/          # Annotation formatting for deny feedback
│   ├── diff/                # Plan diffing (go-udiff)
│   ├── handoff/             # Temp-file payload passing between modes
│   ├── history/             # Per-session plan history for diffing
│   └── spawn/               # Terminal detection and window spawning
```

The two modes share no global state — they communicate only through temp files, making the inter-process handoff simple and debuggable.
