#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY_NAME="todo"
TARGET_DIR="${TARGET_DIR:-$HOME/.local/bin}"
GO_CMD="${GO_CMD:-go}"

if ! command -v "$GO_CMD" >/dev/null 2>&1; then
  echo "Go is required to install todo (https://go.dev/dl/)" >&2
  exit 1
fi

# Use a disposable Go build cache to avoid polluting global cache
if [[ -z "${GOCACHE:-}" || "${GOCACHE}" == "off" ]]; then
  GOCACHE="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/todo-install-go-cache")"
  CLEANUP_GOCACHE=1
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  arm64|aarch64) ARCH=arm64 ;;
  x86_64|amd64) ARCH=amd64 ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

if [[ "$OS" != "darwin" && "$OS" != "linux" ]]; then
  echo "Unsupported OS: $OS (supported: macOS, Linux)" >&2
  exit 1
fi

VERSION=${VERSION:-$($GO_CMD list -m -f '{{.Version}}' 2>/dev/null || echo "dev")}
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LD_FLAGS="-X=github.com/bagadi-alnour/todo-cli/internal/cmd.Version=$VERSION -X=github.com/bagadi-alnour/todo-cli/internal/cmd.BuildDate=$BUILD_DATE"

echo "Building for $OS/$ARCH..."
BIN_OUT="$ROOT_DIR/$BINARY_NAME"
GOOS=$OS GOARCH=$ARCH $GO_CMD build -ldflags "$LD_FLAGS" -o "$BIN_OUT" ./cmd/todo

mkdir -p "$TARGET_DIR"
cp "$BIN_OUT" "$TARGET_DIR/$BINARY_NAME"
rm -f "$BIN_OUT"

if [[ ":$PATH:" != *":$TARGET_DIR:"* ]]; then
  echo "Installed to $TARGET_DIR; add it to PATH if not already."
fi

echo "Installed todo -> $TARGET_DIR/$BINARY_NAME"

# Install shell completions for common shells
install_completions() {
  local shell_name="$1"
  case "$shell_name" in
    bash)
      local bash_dir="${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions"
      mkdir -p "$bash_dir"
      "$TARGET_DIR/$BINARY_NAME" completion bash > "$bash_dir/$BINARY_NAME" && \
        echo "Installed bash completion -> $bash_dir/$BINARY_NAME (requires bash-completion)"
      ;;
    zsh)
      local zsh_dir="${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions"
      mkdir -p "$zsh_dir"
      "$TARGET_DIR/$BINARY_NAME" completion zsh > "$zsh_dir/_$BINARY_NAME" && \
        echo "Installed zsh completion -> $zsh_dir/_$BINARY_NAME"
      local zshrc="${ZDOTDIR:-$HOME}/.zshrc"
      if ! grep -q "$zsh_dir" "$zshrc" 2>/dev/null; then
        {
          echo ""
          echo "# Added by todo install script for completions"
          echo "fpath+=('$zsh_dir')"
          echo "autoload -Uz compinit && compinit"
        } >> "$zshrc"
        echo "Updated $zshrc to load completions (restart shell)"
      fi
      ;;
    fish)
      local fish_dir="${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions"
      mkdir -p "$fish_dir"
      "$TARGET_DIR/$BINARY_NAME" completion fish > "$fish_dir/$BINARY_NAME.fish" && \
        echo "Installed fish completion -> $fish_dir/$BINARY_NAME.fish"
      ;;
    *)
      echo "Skipping completions: unsupported shell '$shell_name'" >&2
      ;;
  esac
}

current_shell="$(basename "${SHELL:-}")"
case "$current_shell" in
  bash|zsh|fish)
    install_completions "$current_shell"
    ;;
  *)
    # Attempt to install all known shells if current shell is unknown
    install_completions bash
    install_completions zsh
    install_completions fish
    ;;
esac

# Cleanup temporary cache if we created one
if [[ "${CLEANUP_GOCACHE:-0}" -eq 1 ]]; then
  rm -rf "$GOCACHE" >/dev/null 2>&1 || true
fi
