package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

var (
	addPaths    []string
	addPriority string
	addNoGit    bool
)

var addCmd = &cobra.Command{
	Use:   "add <text>",
	Short: "Add a new todo",
	Long: `Add a new todo item to the project.

Todos can be associated with file paths for context-aware tracking.
Git branch and commit information is automatically captured unless --no-git is specified.`,
	Example: `  todo add "Fix authentication bug"
  todo add "Refactor middleware" --path src/auth
  todo add "Update tests" -p src/tests -p src/utils
  todo add "Quick fix" --no-git
  todo add "Important task" --priority high`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringArrayVarP(&addPaths, "path", "p", []string{}, "Associate with file/folder paths (can be used multiple times)")
	addCmd.Flags().StringVar(&addPriority, "priority", "medium", "Priority level: low, medium, high")
	addCmd.Flags().BoolVar(&addNoGit, "no-git", false, "Don't capture git context (branch/commit)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	// Join all args as the todo text
	text := strings.Join(args, " ")
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("todo text cannot be empty")
	}

	// Load existing todos
	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	// Generate ID
	id, err := storage.GenerateID()
	if err != nil {
		return fmt.Errorf("failed to generate ID: %w", err)
	}

	// Create new todo
	todo := types.NewTodo(id, text)

	// Set priority
	priority := types.Priority(addPriority)
	if priority != types.PriorityLow && priority != types.PriorityMedium && priority != types.PriorityHigh {
		return fmt.Errorf("invalid priority: %s. Use: low, medium, high", addPriority)
	}
	todo.Priority = priority

	// Set paths if provided
	if len(addPaths) > 0 {
		todo.SetPaths(addPaths)
	}

	// Capture git context unless disabled
	if !addNoGit && git.IsGitRepo() {
		branch, commit, err := git.GetGitContext()
		if err == nil && branch != "" {
			todo.SetGitContext(branch, commit)
		}
	}

	// Add to todos
	todos = append(todos, *todo)

	// Save
	if err := storage.SaveTodos(projectRoot, todos); err != nil {
		return fmt.Errorf("failed to save todos: %w", err)
	}

	// Print success message
	terminal.PrintSuccess(fmt.Sprintf("Added: %s", text))

	if len(addPaths) > 0 {
		fmt.Printf("  %s📁 Paths: %s%s\n", terminal.Dim, strings.Join(addPaths, ", "), terminal.Reset)
	}

	if todo.Context.Branch != "" {
		fmt.Printf("  %s🌿 Branch: %s%s\n", terminal.Dim, todo.Context.Branch, terminal.Reset)
	}

	if todo.Context.Commit != "" {
		fmt.Printf("  %s📝 Commit: %s%s\n", terminal.Dim, todo.Context.Commit, terminal.Reset)
	}

	fmt.Printf("  %s🆔 ID: %s%s\n", terminal.Dim, id[:8], terminal.Reset)
	fmt.Println()

	return nil
}
