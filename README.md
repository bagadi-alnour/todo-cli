# Todo CLI

<p align="center">
  <strong>Project-embedded interactive todo system for developers</strong><br>
  <em>Todos that understand your code</em>
</p>

<p align="center">
  <a href="https://github.com/bagadi-alnour/todo-cli/actions/workflows/ci.yml"><img src="https://github.com/bagadi-alnour/todo-cli/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/bagadi-alnour/todo-cli/releases/latest"><img src="https://img.shields.io/github/v/release/bagadi-alnour/todo-cli" alt="Latest Release"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/bagadi-alnour/todo-cli" alt="Go Version">
  <a href="LICENSE"><img src="https://img.shields.io/github/license/bagadi-alnour/todo-cli" alt="License"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#installation">Installation</a> •
  <a href="#commands">Commands</a> •
  <a href="#scripting---json-output">JSON</a> •
  <a href="#web-ui">Web UI</a> •
  <a href="#storage-format">Storage</a> •
  <a href="#roadmap">Roadmap</a> •
  <a href="CHANGELOG.md">Changelog</a>
</p>

---

## Why not GitHub Issues, Linear, or Todoist?

Those tools are great for team-level work. `todo` is for **project-local developer memory**:

- Tasks tied to **files and folders** — know exactly where work lives.
- **Branch-aware context** — `todo context` shows what matters right now.
- **Survives context switching** — come back tomorrow and run `todo next`.
- **Works offline** — no cloud, no account, no sync.
- **Stores data beside the code** — `.todos/` is plain JSON you can grep, inspect, backup, or commit.
- **Scriptable** — `--json` everywhere, pipe into `jq`, integrate with editors and CI.

## Quick Start

Get going in 30 seconds:

```bash
todo init
todo add "Fix auth bug" --path src/auth --priority high --tag backend
todo list
todo next
todo done 1
```

## Features

- **Project-local storage** — Todos live in `.todos/` inside each repo (optionally committed for the team).
- **Per-creator files** — Each author’s todos go in `.todos/users/<firstname-lastname>.json` (from `git user.name`), so teammates rarely edit the same file in Git.
- **Assignees** — Tag work with `--assign` (git contributor email); filter with `list --assignee`, stats with `stats --by-assignee`, and pick assignees in the Web UI.
- **Git contributors cache** — `todo contributors` lists repo authors for assignee completion and blame-based suggestions on `add`.
- **Context-aware** — Attach file paths; git branch and commit captured automatically.
- **Branch view** — `todo context` shows todos for the current branch. `todo here` shows todos for the current directory.
- **Tags and due dates** — Filter with `--tag`, `--overdue`, `--due-before`, `--due-after`.
- **Notes** — Longer descriptions via `--notes` on `add` / `edit`.
- **Smart next task** — `todo next` ranks by overdue, due date, priority, then age, and tells you *why*.
- **Task dependencies** — `--blocked-by` and `--blocks` link todos; `todo show` displays the graph.
- **Recurring tasks** — `--recur daily|weekly|monthly`; completing auto-creates the next occurrence.
- **Source scan** — `todo scan` parses `TODO`/`FIXME` comments from source files and imports them.
- **Interactive list** — Keyboard-driven TUI (`todo list`); `--static` for pipes/CI.
- **Focus mode** — `todo focus` surfaces work for the current branch.
- **Stats dashboard** — `todo stats` for counts, tags, completion rate, overdue, and time-to-done.
- **Search** — `todo search "<query>"` across text, notes, tags, and paths.
- **Archive** — Move completed items to `.todos/archive.json`.
- **Import / Export** — `todo export --format markdown` or `todo import backup.json`.
- **Watch mode** — `todo watch` polls for changes and emits JSON events for editor integrations.
- **Health checks** — `todo doctor` (optional `--fix`).
- **File locking** — Safe when multiple terminals run `todo add` simultaneously.
- **Atomic writes** — Data files are written via temp file + fsync + rename.
- **Web UI** — Local server (default port **17887**).
- **Scripting** — `--json` on every read command; `--verbose` for diagnostics.

## Screenshots

### Interactive Todo List

![Todo List](docs/images/todo-list.png)

### Focus Mode

![Focus Mode](docs/images/todo-focus.png)

### Web UI

![Web UI](docs/images/todo-ui.png)

## Installation

Requires **Go 1.22+** to build from source.

### macOS

```bash
# Homebrew (recommended)
brew install bagadi-alnour/tap/todo

# From source
git clone https://github.com/bagadi-alnour/todo-cli.git
cd todo-cli && ./scripts/install.sh
```

### Linux

```bash
git clone https://github.com/bagadi-alnour/todo-cli.git
cd todo-cli && ./scripts/install.sh

# Or install with Go
go install github.com/bagadi-alnour/todo-cli/cmd/todo@latest
```

### GitHub Releases

Pre-built binaries for Linux, macOS, and Windows are available on the [Releases](https://github.com/bagadi-alnour/todo-cli/releases) page.

### Cross-build binaries

```bash
make build-all
# todo-darwin-{amd64,arm64}, todo-linux-{amd64,arm64}, todo-windows-amd64.exe
```

### Shell completions

The install script installs completions for your current shell. To install manually:

```bash
# Bash
todo completion bash > /etc/bash_completion.d/todo

# Zsh
mkdir -p ~/.zsh/completions
todo completion zsh > ~/.zsh/completions/_todo
# Then ensure fpath and compinit are set in ~/.zshrc:
#   fpath=(~/.zsh/completions $fpath)
#   autoload -Uz compinit && compinit

# Fish
mkdir -p ~/.config/fish/completions
todo completion fish > ~/.config/fish/completions/todo.fish

# PowerShell
todo completion powershell | Out-String | Invoke-Expression
# Persist by adding to $PROFILE
```

`--path` / `-p` on `add`, `edit`, `list`, `next`, and `search` completes paths relative to the project root.

## Global flags

| Flag | Meaning |
|------|---------|
| `-h`, `--help` | Help for the command |
| `--version` | Print version, commit, and build date |
| `-v`, `--verbose` | Log project root, config, and todo counts to stderr |

## Commands

### Shorthand `-p` (path vs port)

- **`todo ui`:** `-p` / `--port` is the **TCP port** (default **17887**).
- **All other commands** that expose `-p`: it is a **path** (same as `--path`).

---

### `todo init`

Create `.todos/`, `users/`, and `config.json` in the current directory. Your first todo file appears when you run `todo add` (requires `git config user.name`, or set `TODO_USER_NAME`).

Legacy projects with a single `.todos/todos.json` are migrated automatically into `users/` on first load.

```bash
todo init
todo init --force   # Reinitialize
```

---

### `todo add`

```bash
todo add "Fix login bug"
todo add "Refactor auth" --path src/auth -p src/types
todo add "Quick fix" --no-git
todo add "Important" --priority high
todo add "Launch" --tag release --tag qa --due tomorrow
todo add "Spec" --notes "See doc/design.md"
todo add "API" --json --no-git
todo add "Weekly audit" --recur weekly --due 2026-06-01
todo add "DB migration" --blocked-by abc123
todo add "Ship feature" --blocks def456
todo add "Review PR" --assign me
todo add "Ops runbook" --assign alice@example.com
```

`--assign` accepts a contributor name, email prefix, or `me` (your `git config user.email`). With `--path`, `todo add` may suggest an assignee from `git blame` when you omit `--assign`.

Due date supports: `YYYY-MM-DD`, `YYYY-MM-DDTHH:MM`, RFC3339, `today`, `tomorrow`, `+2d`.

---

### `todo list` (`todo ls`)

Default: **interactive TUI** when stdout is a TTY.

```bash
todo list --static
todo list --static --details
todo list -s open
todo list --status done
todo list -p src/
todo list --priority high
todo list -t backend -t frontend
todo list --overdue
todo list --due-before 2026-03-01
todo list --assignee me
todo list --assignee alice
todo list --json
```

**Interactive keys**

| Key | Action |
|-----|--------|
| `↑` `↓` or `j` `k` | Move selection |
| `Space` / `Enter` | Toggle status (confirm `Y` when marking done; re-open is instant) |
| `i` or `→` / `←` | Expand / collapse full details for the selected todo |
| `d` `x` | Delete (confirm `Y` / cancel `N` `q` `Esc`) |
| `g` / `G` | Jump to first / last |
| `?` `h` `H` | Help overlay |
| `q` / `Esc` | Quit |

---

### `todo show`

Display full details of a single todo.

```bash
todo show 1
todo show abc123
todo show 1 --json
```

---

### `todo edit`

```bash
todo edit 1 --text "New title"
todo edit 1 --status blocked
todo edit 1 --priority low
todo edit 1 -p cmd/foo.go --path cmd/bar.go
todo edit 1 --clear-paths
todo edit 1 -t backend --tag security
todo edit 1 --add-tag ops --remove-tag backend
todo edit 1 --clear-tags
todo edit 1 --due +3d
todo edit 1 --clear-due
todo edit 1 --notes "Longer description"
todo edit 1 --clear-notes
todo edit 1 --blocked-by abc123
todo edit 1 --blocks def456
todo edit 1 --clear-blocked-by
todo edit 1 --recur weekly
todo edit 1 --clear-recur
todo edit 1 --assign bob
todo edit 1 --clear-assignee
```

---

### `todo contributors`

List git contributors for the repo (cached in `.todos/contributors.json`). Used for `--assign` / `--assignee` tab completion.

```bash
todo contributors
todo contributors --refresh
todo contributors --json
```

---

### `todo done`

Mark one or more items done (by index or ID).

```bash
todo done 1
todo done 1 2 3
todo done a3f9c2d1
```

---

### `todo delete` (`del`, `rm`)

```bash
todo delete 2
todo delete 1 3 5
```

---

### `todo status` (`set-status`)

Last argument is the new status. All preceding are IDs or indices.

```bash
todo status 1 blocked
todo status 1 2 3 done
```

Statuses: `open`, `done`, `blocked`, `waiting`, `tech-debt`.

---

### `todo focus`

```bash
todo focus              # open todos on current branch
todo focus --all        # all open todos
todo focus --priority high
todo focus --json
```

---

### `todo next`

```bash
todo next
todo next --all
todo next -t backend
todo next -p src/auth --priority high
todo next --json
```

---

### `todo context`

Show todos for the current Git branch.

```bash
todo context
todo context --json
```

---

### `todo here`

Show todos related to the current directory.

```bash
cd src/auth && todo here
todo here --json
```

---

### `todo search`

Case-insensitive match on text, notes, tags, and paths.

```bash
todo search "auth"
todo search "billing" -s open
todo search "api" -t backend -p src/
todo search "fix" --json
```

---

### `todo stats`

```bash
todo stats
todo stats --by-assignee
todo stats --json
```

---

### `todo archive`

Move **done** items from all user files into `.todos/archive.json`.

```bash
todo archive
todo archive --before 2025-12-31
todo archive --json
```

---

### `todo export`

```bash
todo export                        # JSON to stdout
todo export --format markdown      # Markdown checklist
todo export --format json > backup.json
```

---

### `todo import`

Import todos from a previously exported JSON file. Duplicate IDs are skipped.

```bash
todo import backup.json
todo import ../other-project/.todos/users/alice-smith.json
```

---

### `todo doctor`

```bash
todo doctor
todo doctor --fix
todo doctor --json
```

Checks: project init, `users/` storage, config file, git repo, write access.

---

### `todo ui`

```bash
todo ui                    # default port 17887
todo ui --port 3000
todo ui -p 9000
```

Open `http://localhost:17887` (or your chosen port).

---

### `todo scan`

Parse `TODO`, `FIXME`, `HACK`, and `XXX` comments from source files and import them as todos.

```bash
todo scan                         # Scan current directory
todo scan src/                    # Scan specific directory
todo scan --dry-run               # Preview without importing
todo scan --tag code-review       # Tag all imported todos
todo scan --json
```

---

### `todo watch`

Watch `.todos/` for changes and emit JSON events to stdout. Useful for editor integrations or status bars.

```bash
todo watch                  # Poll every 2 seconds
todo watch --interval 5     # Poll every 5 seconds
```

---

### `todo config`

```bash
todo config
todo config --auto-git false
todo config --default-branch main
todo config --reset
```

---

### `todo completion`

```bash
todo completion bash|zsh|fish|powershell
```

## Scripting — `--json` output

Commands that support `--json`:

| Command | Output shape |
|---------|-------------|
| `todo add --json` | Single todo object |
| `todo list --json` | `{ "todos", "count", "stats" }` |
| `todo show --json` | Single todo object |
| `todo next --json` | `{ "todo", "reason", "count" }` |
| `todo focus --json` | `{ "todos", "count", "branch" }` |
| `todo context --json` | `{ "branch", "todos", "count" }` |
| `todo here --json` | `{ "directory", "todos", "count" }` |
| `todo doctor --json` | Health check summary |
| `todo stats --json` | Full statistics report |
| `todo archive --json` | `{ "archived", "count" }` |
| `todo search --json` | `{ "query", "results", "count" }` |
| `todo scan --json` | `{ "found", "count" }` |
| `todo export` | TodoFile object or Markdown |

Examples:

```bash
# High-priority open todos
todo list --json | jq '.todos[] | select(.priority == "high" and .status == "open")'

# Count by status
todo stats --json | jq '.byStatus'

# Todos for a specific path
todo here --json | jq '.todos[].text'
```

## Web UI

- Dashboard-style overview with filters, assignee dropdown, keyboard shortcuts, and live updates.
- Reads the same `.todos/` files as the CLI — merges all `users/*.json` — no separate database.
- **All** view hides completed todos (use the **done** filter to see them); list is sorted newest-first.
- **No cloud sync. No account. No background daemon.**
- Default port: **17887**. Override with `--port`.

## Workflow examples

```bash
# During code review
todo add "Refactor auth middleware after review" -p internal/auth --tag review

# Before switching branches
todo add "Finish migration cleanup" --priority high --tag migration

# Coming back to a project
todo next

# See what's relevant in this directory
cd src/billing && todo here

# See what's on this branch
todo context

# Project handoff — show all billing-related work
todo list --json -p src/billing | jq '.todos[] | {text, status, priority}'

# Export for a teammate
todo export --format markdown > TODO.md

# Set up a recurring reminder
todo add "Dependency audit" --recur weekly --due 2026-06-01 --tag maintenance

# Link related tasks
todo add "Write API endpoints" --blocks abc123
todo edit 1 --blocked-by def456

# Assign and filter by owner
todo add "Fix billing webhook" -p src/billing --assign me
todo list --static --assignee me

# Scan codebase for TODO comments
todo scan --dry-run           # preview first
todo scan --tag tech-debt     # import with tag
```

## Storage format

### `.todos/users/<firstname-lastname>.json`

Each file holds one creator’s todos. The filename comes from `git config user.name` (slugified, e.g. `Jane Doe` → `jane-doe.json`). The CLI and Web UI **read all** `users/*.json` and merge them for `list`, `next`, `stats`, and the UI.

```json
{
  "version": 1,
  "todos": [
    {
      "id": "a3f9c2d1e4b5…",
      "text": "Refactor authentication",
      "notes": "See ADR-12",
      "status": "open",
      "priority": "high",
      "tags": ["backend", "security"],
      "assignee": "alice@example.com",
      "createdBy": "jane-doe",
      "dueAt": "2026-01-25T23:59:59Z",
      "recur": "weekly",
      "blockedBy": ["b1c2d3e4"],
      "blocks": ["f5a6b7c8"],
      "createdAt": "2026-01-19T10:00:00Z",
      "updatedAt": "2026-01-19T10:00:00Z",
      "context": {
        "paths": ["src/auth/"],
        "branch": "feature/auth-refactor",
        "commit": "abc1234"
      },
      "meta": { "source": "cli" }
    }
  ]
}
```

- **`createdBy`** — slug of who added the todo (which file owns it). Not the same as **assignee** (who should do the work).
- **`assignee`** — git author email (resolved from names via `todo contributors`).

### Legacy `.todos/todos.json`

Older projects used a single `todos.json`. On first load it is migrated into `users/legacy.json` (or per-todo `createdBy` when present). New projects do not create `todos.json`.

### `.todos/contributors.json`

Cached output of `git shortlog` for assignee resolution and shell completion. Refreshed with `todo contributors --refresh`.

### `.todos/archive.json`

Same JSON shape as a user file. Written by `todo archive`, appended over time.

### `.todos/config.json`

```json
{
  "version": 1,
  "autoGit": true,
  "defaultBranch": "main"
}
```

Your data is plain JSON. Grep it, commit it, back it up, import it elsewhere.

## Sharing `.todos/` via Git

Committing `.todos/` lets your team share todos across branches. Everyone sees the full list via CLI/UI; each person’s new todos land in **their own** `users/<slug>.json`, which greatly reduces merge conflicts compared to one shared file.

### Handling merge conflicts

Conflicts are most common when two branches edit the **same** `users/*.json` (same author) or `archive.json`. Resolution options:

1. **Accept one side** — pick the file with more complete data.
2. **Re-import** — export the other side (`todo export > /tmp/theirs.json`), resolve, then `todo import /tmp/theirs.json` (duplicate IDs skipped).
3. **Merge manually** — copy missing todo objects into the `"todos"` array with unique IDs.

**Recommended `.gitattributes`** (also in this repo as [`.gitattributes`](.gitattributes)):

```gitattributes
.todos/users/*.json merge=union
.todos/archive.json merge=union
.todos/contributors.json merge=union
```

After a union merge, run `todo doctor --fix` to validate and deduplicate.

**Tips:**

- Use short-lived branches and merge often.
- Assignees can differ from file owner — hand off work with `--assign` without moving files.
- Use `todo export` / `todo import` when you need to move todos between machines without committing `.todos/`.

## Status types

| Status | Icon | Meaning |
|--------|------|---------|
| `open` | ○ | Active |
| `done` | ✓ | Completed |
| `blocked` | ⊘ | Blocked on dependency |
| `waiting` | ◷ | Paused |
| `tech-debt` | △ | Tech debt |

## Philosophy

> **The tool is global. The data is local.**

1. One binary, any repo.
2. Todos live next to the code they describe.
3. CLI-first; JSON and flags for automation.
4. Human-readable JSON on disk — no lock-in.
5. Git-aware defaults (`autoGit`, branch on add, `todo focus`, `todo context`).

## Development

```bash
make build          # embeds version + commit via ldflags
make test
make install        # copies binary to ~/bin
make install-global # copies to /usr/local/bin (sudo)
make fmt
make lint           # requires golangci-lint
make completions
```

### Git hooks

After clone, set up the commit-msg hook:

```bash
git config core.hooksPath githooks
```

### Releases

Releases are automated via [GoReleaser](https://goreleaser.com/). Push a tag to create a release:

```bash
git tag v0.6.0
git push origin v0.6.0
```

See [RELEASE.md](RELEASE.md) and [CHANGELOG.md](CHANGELOG.md) for the full release checklist.

## Roadmap

- [ ] Editor integrations (VS Code, Neovim) — sidebar panel and inline diagnostics
- [ ] Markdown import (parse `- [ ]` lists)
- [ ] `todo context` auto-filter by changed files
- [x] Web UI — assignee picker/filter, layout polish, newest-first sort
- [ ] Web UI — drag-and-drop, dark mode, live SSE updates via `todo watch`
- [ ] Subtask nesting (parent/child relationships)

## Contributing

Contributions are welcome via Pull Request.

## License

MIT — see [LICENSE](LICENSE).
