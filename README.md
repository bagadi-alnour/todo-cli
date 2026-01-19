# Todo CLI

A project-embedded interactive todo system for developers. Todos that live with your code.

## Features

- **Project-local storage** - Todos stored in `.todos/` directory within your project
- **Context-aware** - Attach todos to file paths, folders, or patterns
- **Git integration** - Automatically captures branch and commit context
- **Interactive CLI** - Navigate, toggle, and delete todos with keyboard shortcuts
- **Web UI** - Optional modern dark-themed web interface
- **Orphan detection** - Find todos linked to non-existent files

## Installation

### Using Go Install (Recommended)

```bash
go install github.com/bagadi-alnour/todo-cli/cmd/todo@latest
```

This will install the `todo` binary to your `$GOPATH/bin` (usually `~/go/bin`).

### From Source

```bash
# Clone the repository
git clone https://github.com/bagadi-alnour/todo-cli.git
cd todo-cli

# Build
make build

# Install to ~/bin
make install

# Or install globally (requires sudo)
make install-global
```

### Pre-built Binaries

Download from the [releases page](https://github.com/bagadi-alnour/todo-cli/releases).

## Quick Start

```bash
# Initialize a todo project in your current directory
todo init

# Add your first todo
todo add "Setup authentication system"

# Add a todo with context
todo add "Refactor middleware" --path src/middleware/

# List todos interactively
todo list

# Mark a todo as done
todo done 1
```

## Commands

### `todo init`
Initialize a new todo project in the current directory.

```bash
todo init           # Create .todos/ directory
todo init --force   # Reinitialize existing project
```

### `todo add`
Add a new todo item.

```bash
todo add "Fix login bug"
todo add "Refactor auth" --path src/auth
todo add "Update tests" --path src/tests --path src/utils
todo add "Quick fix" --no-git      # Skip git context capture
todo add "Important" --priority high
```

### `todo list`
List todos with interactive navigation.

```bash
todo list                 # Interactive mode (default)
todo list --static        # Non-interactive output
todo list --status open   # Filter by status
todo list --path src/     # Filter by path
```

**Keyboard Shortcuts:**
- `↑`/`↓` or `j`/`k` - Navigate
- `Space` or `Enter` - Toggle status
- `d` or `x` - Delete todo
- `g` - Jump to top
- `G` - Jump to bottom
- `?` - Show help
- `q` - Quit

### `todo done`
Mark a todo as complete.

```bash
todo done 1         # By list index
todo done abc123    # By ID (partial match supported)
```

### `todo focus`
Show todos relevant to your current context.

```bash
todo focus         # Branch-relevant todos
todo focus --all   # All open todos
```

### `todo doctor`
Run health checks on your todo list.

```bash
todo doctor        # Check for issues
todo doctor --fix  # Auto-fix where possible
```

**Checks:**
- Orphaned paths (todos pointing to deleted files)
- Empty todos
- Duplicate todos
- Stale todos (open > 30 days)

### `todo ui`
Start the web interface.

```bash
todo ui              # Start on port 8080
todo ui --port 3000  # Custom port
```

## Storage Format

Todos are stored in `.todos/todos.json`:

```json
{
  "version": 1,
  "todos": [
    {
      "id": "a3f9c2d1e4b5...",
      "text": "Refactor authentication",
      "status": "open",
      "priority": "high",
      "createdAt": "2026-01-19T10:00:00Z",
      "updatedAt": "2026-01-19T10:00:00Z",
      "context": {
        "paths": ["src/auth/"],
        "branch": "feature/auth-refactor",
        "commit": "abc1234"
      },
      "meta": {
        "source": "cli"
      }
    }
  ]
}
```

## Status Types

| Status | Description |
|--------|-------------|
| `open` | Active, needs to be done |
| `done` | Completed |
| `blocked` | Waiting on external dependency |
| `waiting` | Paused, will resume later |
| `tech-debt` | Known technical debt to address |

## Philosophy

1. **The tool is global. The data is local.** - Install once, use everywhere.
2. **Todos should live with the code.** - Context matters.
3. **CLI-first.** - Fast and keyboard-driven.
4. **Human-readable storage.** - Easy to inspect and edit manually.
5. **Git-aware.** - Understand your development workflow.

## Configuration

Configuration is stored in `.todos/config.json`:

```json
{
  "version": 1,
  "autoGit": true,
  "defaultBranch": "main"
}
```

## Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Development mode with hot reload
make dev
```

## License

MIT
