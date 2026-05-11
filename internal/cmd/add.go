package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	addPaths     []string
	addPriority  string
	addNoGit     bool
	addTags      []string
	addDue       string
	addJSON      bool
	addNotes     string
	addBlockedBy []string
	addBlocks    []string
	addRecur     string
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
  todo add "Important task" --priority high
  todo add "Ship billing flow" --tag billing --tag backend --due 2026-03-01`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringArrayVarP(&addPaths, "path", "p", []string{}, "Associate with file/folder paths (can be used multiple times)")
	addCmd.Flags().StringVar(&addPriority, "priority", "medium", "Priority level: low, medium, high")
	addCmd.Flags().BoolVar(&addNoGit, "no-git", false, "Don't capture git context (branch/commit)")
	addCmd.Flags().StringArrayVarP(&addTags, "tag", "t", []string{}, "Tag(s) for organizing and filtering (repeat or comma-separate)")
	addCmd.Flags().StringVar(&addDue, "due", "", "Due date/time (YYYY-MM-DD, YYYY-MM-DDTHH:MM, RFC3339, today, tomorrow, +2d)")
	addCmd.Flags().StringVar(&addNotes, "notes", "", "Additional notes or description")
	addCmd.Flags().StringArrayVar(&addBlockedBy, "blocked-by", []string{}, "IDs of todos that block this one")
	addCmd.Flags().StringArrayVar(&addBlocks, "blocks", []string{}, "IDs of todos that this one blocks")
	addCmd.Flags().StringVar(&addRecur, "recur", "", "Recurrence when completed: daily, weekly, monthly")
	addCmd.Flags().BoolVar(&addJSON, "json", false, "Output the created todo as JSON")

	// Project-aware path completion
	registerPathFlagCompletion(addCmd, "path")
}

func runAdd(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}
	Verbosef("project root: %s", projectRoot)

	pathFlagUsed := cmd.Flags().Changed("path")

	config, err := storage.LoadConfig(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	Verbosef("config: autoGit=%v, defaultBranch=%q", config.AutoGit, config.DefaultBranch)

	text := strings.Join(args, " ")
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("todo text cannot be empty")
	}
	if pathFlagUsed || len(addPaths) > 0 {
		switch {
		case len(args) > 1:
			text = strings.TrimSpace(args[0])
			addPaths = append(addPaths, args[1:]...)
		case len(args) == 1:
			text, addPaths = splitTrailingPaths(text, addPaths)
		}
	}

	priority := types.Priority(addPriority)
	if priority != types.PriorityLow && priority != types.PriorityMedium && priority != types.PriorityHigh {
		return fmt.Errorf("invalid priority: %s. Use: low, medium, high", addPriority)
	}

	var dueAt *time.Time
	if cmd.Flags().Changed("due") {
		d, err := parseDueDateInput(addDue, time.Now())
		if err != nil {
			return err
		}
		dueAt = d
	}

	if addRecur != "" {
		r := types.Recurrence(strings.ToLower(addRecur))
		if !r.IsValid() {
			return fmt.Errorf("invalid recurrence: %s. Use: daily, weekly, monthly", addRecur)
		}
	}

	var todo *types.Todo
	err = storage.WithLock(projectRoot, func() error {
		todos, err := storage.LoadTodos(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load todos: %w", err)
		}

		id, err := storage.GenerateID()
		if err != nil {
			return fmt.Errorf("failed to generate ID: %w", err)
		}

		todo = types.NewTodo(id, text)
		todo.Priority = priority

		normalizedPaths := normalizePaths(addPaths)
		if len(normalizedPaths) > 0 {
			todo.SetPaths(normalizedPaths)
		}
		todo.Tags = normalizeTags(addTags)
		if addNotes != "" {
			todo.Notes = addNotes
		}
		todo.DueAt = dueAt

		if addRecur != "" {
			todo.Recur = types.Recurrence(strings.ToLower(addRecur))
		}
		if len(addBlockedBy) > 0 {
			todo.BlockedBy = addBlockedBy
		}
		if len(addBlocks) > 0 {
			todo.Blocks = addBlocks
		}

		if !addNoGit && config.AutoGit && git.IsGitRepo() {
			branch, commit, err := git.GetGitContext()
			if err == nil && branch != "" {
				todo.SetGitContext(branch, commit)
			}
		} else if !addNoGit && config.AutoGit && config.DefaultBranch != "" {
			todo.SetGitContext(config.DefaultBranch, "")
		}

		todos = append(todos, *todo)
		return storage.SaveTodos(projectRoot, todos)
	})
	if err != nil {
		return err
	}

	if addJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(todo)
	}

	terminal.PrintSuccess(fmt.Sprintf("Added: %s", text))

	if len(todo.Context.Paths) > 0 {
		fmt.Printf("  %s📁 Paths: %s%s\n", terminal.Dim, strings.Join(todo.Context.Paths, ", "), terminal.Reset)
	}
	if len(todo.Tags) > 0 {
		fmt.Printf("  %s🏷️ Tags: %s%s\n", terminal.Dim, strings.Join(todo.Tags, ", "), terminal.Reset)
	}
	if todo.DueAt != nil {
		fmt.Printf("  %s⏳ %s%s\n", terminal.Dim, formatDueLabel(todo.DueAt, time.Now()), terminal.Reset)
	}
	if todo.Context.Branch != "" {
		fmt.Printf("  %s🌿 Branch: %s%s\n", terminal.Dim, todo.Context.Branch, terminal.Reset)
	}
	if todo.Context.Commit != "" {
		fmt.Printf("  %s📝 Commit: %s%s\n", terminal.Dim, todo.Context.Commit, terminal.Reset)
	}
	fmt.Printf("  %s🆔 ID: %s%s\n", terminal.Dim, todo.ID[:8], terminal.Reset)
	fmt.Println()

	return nil
}
