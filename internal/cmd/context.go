package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var contextJSON bool

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show todos related to the current Git branch",
	Long: `Display todos whose Git context matches the current branch.

This helps you see what's relevant right now when switching between branches.`,
	Example: `  todo context
  todo context --json`,
	RunE: runContext,
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.Flags().BoolVar(&contextJSON, "json", false, "Output as JSON")
}

func runContext(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	branch := ""
	if git.IsGitRepo() {
		b, _, err := git.GetGitContext()
		if err == nil {
			branch = b
		}
	}

	if branch == "" {
		if contextJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{"branch": nil, "todos": []types.Todo{}, "message": "not a git repository"})
		}
		terminal.PrintWarning("Not inside a Git repository")
		fmt.Println()
		return nil
	}

	var open []types.Todo
	for _, t := range todos {
		if t.Status != types.StatusDone && t.Context.Branch == branch {
			open = append(open, t)
		}
	}

	if contextJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{"branch": branch, "todos": open, "count": len(open)})
	}

	terminal.PrintHeader(fmt.Sprintf("BRANCH: %s", branch), "🌿")

	if len(open) == 0 {
		fmt.Printf("  %sNo open todos on this branch%s\n\n", terminal.Dim, terminal.Reset)
		return nil
	}

	storage.SortTodosByPriority(open)
	for i, t := range open {
		priorityLabel, priorityColor := priorityVisual(t.Priority)
		paths := ""
		if len(t.Context.Paths) > 0 {
			paths = fmt.Sprintf(" %s%s%s", terminal.Dim, strings.Join(t.Context.Paths, ", "), terminal.Reset)
		}
		fmt.Printf("  %d. %s%s%s %s%s%s%s\n",
			i+1, priorityColor, priorityLabel, terminal.Reset,
			terminal.Bold, t.Text, terminal.Reset, paths)
	}
	fmt.Println()

	return nil
}
