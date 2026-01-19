# Project-Embedded Interactive Todo System (CLI-first, Cobra-based)

A **globally installed**, CLI-first todo system that stores **project-local data**, providing deep contextual awareness through file, folder, and path attachments — with an optional lightweight web UI.

---

## Core Architecture (Locked)

### Tool
- Global Go binary (`todo`)
- Separate repository (tool evolves independently)
- Installed via:
  ```bash
  go install github.com/you/todo@latest
  # later: brew install todo
  ```
- Built with **Cobra** for command structure

### Data
- Stored **inside each project**, never mixed with tool code
- AI-readable and git-friendly

```
.todos/
  todos.json     # durable source of truth (committable)
  config.json    # local config (optional)
```

### UI
- Optional, explicit (`todo ui`)
- Go HTTP server + **HTMX**
- No React, no build pipeline
- UI never owns state (CLI + JSON do)

---

## Features (MVP)

- ✅ **Project-local storage** (`.todos/`)
- ✅ **CLI-first UX** (fast, scriptable)
- ✅ **Context-aware todos** (files, folders, globs)
- ✅ **Automatic project detection** (git-like traversal)
- ✅ **Human-readable JSON schema** (AI-friendly)
- ✅ **Multiple statuses** (open, done, blocked, waiting, tech-debt)
- ✅ **Optional web UI** (`todo ui`)

---

## Quick Start

### Install

```bash
go install github.com/you/todo@latest
```

### Initialize a project

```bash
todo init
```

Creates:
```
.todos/
  todos.json
  config.json
```

---

## Core Commands (Cobra-based)

### `todo add <text> [paths...]`
Add a todo with optional contextual attachments.

```bash
todo add "Refactor auth middleware" src/auth/
todo add "Clean utils" src/utils/*.ts
```

---

### `todo list`
List todos.

```bash
todo list
todo list --status open
todo list --path src/auth
```

- Defaults to clean, readable output
- Interactive TUI is **out of scope for MVP** (kept simple & scriptable)

---

### `todo focus`
Show only actionable todos related to:
- current directory
- current git branch (if available)

```bash
todo focus
```

---

### `todo done <id>`
Mark a todo as completed.

```bash
todo done 1
todo done a3f9c2
```

---

### `todo ui [port]`
Launch optional local web UI.

```bash
todo ui        # random free port
todo ui 3000   # custom port
```

- HTMX-powered
- Server-rendered HTML
- Uses same JSON + logic as CLI

---

## Storage Structure

```
.todos/
  todos.json    # source of truth (recommended to commit)
  config.json   # per-project settings
```

---

## Todo JSON Schema (v1 – future-proof)

```json
{
  "version": 1,
  "todos": [
    {
      "id": "uuid",
      "title": "Refactor auth middleware",
      "status": "open",
      "priority": "medium",
      "createdAt": "2026-01-19T10:12:00Z",
      "updatedAt": "2026-01-19T10:12:00Z",
      "context": {
        "paths": ["src/auth", "src/shared/logger.ts"],
        "branch": "auth-refactor",
        "commit": null
      },
      "meta": {
        "source": "cli",
        "aiHint": "Touches authentication flow"
      },
      "history": [
        {
          "at": "2026-01-19T10:12:00Z",
          "event": "created"
        }
      ]
    }
  ]
}
```

### Schema Design Goals
- Flat, readable JSON
- Stable IDs
- Path-based context first-class
- Explicit metadata for future AI tooling
- Append-only history for auditability

---

## Project Detection

Works like `git`:

1. Start in current directory
2. Walk upward until `.todos/` is found
3. If none exists, `todo init` creates one

Allows running `todo` from any subdirectory.

---

## Roadmap

### MVP
- Cobra-based CLI
- `.todos/` storage
- add / list / done / focus
- Project detection
- Optional web UI (read + toggle)

### V1
- Git-aware context (branch + commit auto-capture)
- Smarter focus filtering
- UI filtering + status updates

### V2
- Orphan detection (`todo doctor`)
- TODO comment ingestion
- Migration tools

### V3
- AI-assisted suggestions
- Cross-project analytics (opt-in)

---

## Philosophy

> **"The tool is global. The data is local."**

- Project-native data
- No code pollution
- CLI-first, UI-assisted
- Optimized for developers and AI

---

## Build & Distribution

### Requirements
- Go 1.19+

### Local build
```bash
go build -o todo .
```

### Installation
- `go install` (primary)
- Homebrew formula (later)

---

## License

MIT License

