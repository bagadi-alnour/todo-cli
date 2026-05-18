package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bagadi-alnour/todo-cli/internal/contributors"
	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/spf13/cobra"
)

var (
	contributorsRefresh bool
	contributorsJSON    bool
)

var contributorsCmd = &cobra.Command{
	Use:   "contributors",
	Short: "List git contributors for assignee completion",
	Long: `List people who have committed to this repository (from git shortlog).

The list is cached in .todos/contributors.json and used for --assign completion.`,
	Example: `  todo contributors
  todo contributors --refresh
  todo contributors --json`,
	RunE: runContributors,
}

func init() {
	rootCmd.AddCommand(contributorsCmd)
	contributorsCmd.Flags().BoolVar(&contributorsRefresh, "refresh", false, "Rebuild contributor cache from git")
	contributorsCmd.Flags().BoolVar(&contributorsJSON, "json", false, "Output as JSON")
}

func runContributors(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	var f *contributors.File
	if contributorsRefresh {
		if !git.IsGitRepo() {
			return fmt.Errorf("not a git repository")
		}
		f, err = contributors.RefreshFromGit(projectRoot)
		if err != nil {
			return err
		}
	} else {
		f, err = contributors.EnsureLoaded(projectRoot)
		if err != nil {
			return err
		}
	}

	if contributorsJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(f)
	}

	if len(f.Contributors) == 0 {
		if !git.IsGitRepo() {
			terminal.PrintInfo("No git repository — contributors are sourced from git history")
		} else {
			terminal.PrintInfo("No contributors found. Try: todo contributors --refresh")
		}
		fmt.Println()
		return nil
	}

	terminal.PrintHeader("GIT CONTRIBUTORS", "👥")
	for _, c := range f.Contributors {
		label := contributors.DisplayName(c)
		if c.Commits > 0 {
			fmt.Printf("  %s%s%s %s(%d commits)%s\n", terminal.Cyan, label, terminal.Reset, terminal.Dim, c.Commits, terminal.Reset)
		} else {
			fmt.Printf("  %s%s%s\n", terminal.Cyan, label, terminal.Reset)
		}
		fmt.Printf("    %s%s%s\n", terminal.Dim, c.Email, terminal.Reset)
	}
	fmt.Println()
	return nil
}
