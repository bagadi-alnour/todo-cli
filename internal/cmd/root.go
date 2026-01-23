package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version   = "0.1.2"
	BuildDate = "unknown"
)

// Global flags
var (
	verbose    bool
	configPath string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "todo",
	Short: "Project-embedded interactive todo system",
	Long: `A CLI-first todo system that stores project-local data,
providing deep contextual awareness through file, folder,
and path attachments — with an optional lightweight web UI.

The tool is global. The data is local.`,
	Version: Version,
}

// Minimal fallbacks so bash completion works even without bash-completion installed
const bashCompletionFallback = `
_todo_bash_fallback_get_comp_words_by_ref() {
  # Ignore any option flags (e.g., -n "=") — we only need positionals.
  while [[ "$1" == -* ]]; do
    case "$1" in
      -n|-c|-o) shift 2 ;;
      *) shift ;;
    esac
  done
  local curvar=$1 prevvar=$2 wordsvar=$3 cwordvar=$4

  local cur=${COMP_WORDS[COMP_CWORD]}
  local prev=${COMP_WORDS[COMP_CWORD-1]}
  local cword=${COMP_CWORD}

  printf -v "$curvar" '%s' "$cur"
  printf -v "$prevvar" '%s' "$prev"
  printf -v "$cwordvar" '%s' "$cword"
  eval "$wordsvar=(\"${COMP_WORDS[@]}\")"
}

if ! declare -F _get_comp_words_by_ref >/dev/null; then
  _get_comp_words_by_ref() { _todo_bash_fallback_get_comp_words_by_ref "$@"; }
fi
`

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Config file path (default: .todos/config.json)")

	// Disable completion command by default (can be enabled later)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Ensure bash completions work without external bash-completion helpers
	rootCmd.BashCompletionFunction = bashCompletionFallback
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}

// GetConfigPath returns the config path
func GetConfigPath() string {
	return configPath
}
