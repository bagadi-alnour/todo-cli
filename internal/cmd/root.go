package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version   = "0.1.1"
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
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}

// GetConfigPath returns the config path
func GetConfigPath() string {
	return configPath
}
