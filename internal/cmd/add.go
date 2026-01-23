package cmd

import (
	"fmt"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
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

	// Project-aware path completion
	registerPathFlagCompletion(addCmd, "path")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	// Track whether the user supplied --path/-p
	pathFlagUsed := cmd.Flags().Changed("path")

	// Load config
	config, err := storage.LoadConfig(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle text and any trailing args when --path is used
	text := strings.Join(args, " ")
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("todo text cannot be empty")
	}
	if (pathFlagUsed || len(addPaths) > 0) && len(args) > 1 {
		// Treat args after the first as implicit path entries when a path flag is present.
		text = strings.TrimSpace(args[0])
		addPaths = append(addPaths, args[1:]...)
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

	// Set paths if provided (supports comma-separated lists)
	normalizedPaths := normalizePaths(addPaths)
	if len(normalizedPaths) > 0 {
		todo.SetPaths(normalizedPaths)
	}

	// Capture git context unless disabled
	if !addNoGit && config.AutoGit && git.IsGitRepo() {
		branch, commit, err := git.GetGitContext()
		if err == nil && branch != "" {
			todo.SetGitContext(branch, commit)
		}
	} else if !addNoGit && config.AutoGit && config.DefaultBranch != "" {
		todo.SetGitContext(config.DefaultBranch, "")
	}

	// Add to todos
	todos = append(todos, *todo)

	// Save
	if err := storage.SaveTodos(projectRoot, todos); err != nil {
		return fmt.Errorf("failed to save todos: %w", err)
	}

	// Print success message
	terminal.PrintSuccess(fmt.Sprintf("Added: %s", text))

	if len(normalizedPaths) > 0 {
		fmt.Printf("  %s📁 Paths: %s%s\n", terminal.Dim, strings.Join(normalizedPaths, ", "), terminal.Reset)
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
