package cmd

import (
	"fmt"
	"strconv"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	configAutoGit       string
	configDefaultBranch string
	configReset         bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or update project configuration",
	Long: `View or update the todo project's configuration.

When no flags are provided, the current configuration is shown.
Use --auto-git and --default-branch to update values, or --reset to
restore defaults.`,
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().StringVar(&configAutoGit, "auto-git", "", "Enable/disable automatic git context capture (true/false)")
	configCmd.Flags().StringVar(&configDefaultBranch, "default-branch", "", "Set the default branch used when git context is unavailable")
	configCmd.Flags().BoolVar(&configReset, "reset", false, "Reset configuration to defaults")
}

func runConfig(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	cfg, err := storage.LoadConfig(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	modified := false
	if configReset {
		cfg = types.DefaultConfig()
		modified = true
	}

	if cmd.Flags().Changed("auto-git") {
		value, err := strconv.ParseBool(configAutoGit)
		if err != nil {
			return fmt.Errorf("invalid value for --auto-git: %s (use true/false)", configAutoGit)
		}
		cfg.AutoGit = value
		modified = true
	}

	if cmd.Flags().Changed("default-branch") {
		cfg.DefaultBranch = configDefaultBranch
		modified = true
	}

	if modified {
		if err := storage.SaveConfig(projectRoot, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		terminal.PrintSuccess("Configuration updated")
		fmt.Println()
	}

	fmt.Printf("  %sConfig:%s\n", terminal.Dim, terminal.Reset)
	fmt.Printf("    %sautoGit:%s       %v\n", terminal.BrightCyan, terminal.Reset, cfg.AutoGit)
	defaultBranch := cfg.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "(not set)"
	}
	fmt.Printf("    %sdefaultBranch:%s %s\n\n", terminal.BrightCyan, terminal.Reset, defaultBranch)

	return nil
}
