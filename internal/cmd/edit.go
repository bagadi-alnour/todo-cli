package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	editText       string
	editPaths      []string
	editClearPaths bool
	editPriority   string
	editStatus     string
)

var editCmd = &cobra.Command{
	Use:   "edit <id|index>",
	Short: "Edit a todo's text, status, priority, or paths",
	Long: `Update an existing todo without opening the interactive list.

You can change the text, status, priority, or replace/clear any paths.`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVar(&editText, "text", "", "New todo text")
	editCmd.Flags().StringArrayVarP(&editPaths, "path", "p", []string{}, "Replace paths (can be provided multiple times)")
	editCmd.Flags().BoolVar(&editClearPaths, "clear-paths", false, "Remove all associated paths")
	editCmd.Flags().StringVar(&editPriority, "priority", "", "Set priority: low, medium, high")
	editCmd.Flags().StringVar(&editStatus, "status", "", "Set status: open, done, blocked, waiting, tech-debt")

	// Project-aware path completion
	registerPathFlagCompletion(editCmd, "path")
}

func runEdit(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	todo, idx := storage.FindTodoByIDOrIndex(todos, args[0])
	if todo == nil {
		return &types.TodoNotFoundError{ID: args[0]}
	}

	updated := false

	if cmd.Flags().Changed("text") {
		text := strings.TrimSpace(editText)
		if text == "" {
			return fmt.Errorf("todo text cannot be empty")
		}
		todos[idx].Text = text
		updated = true
	}

	if cmd.Flags().Changed("priority") {
		p := types.Priority(strings.ToLower(editPriority))
		if !p.IsValid() {
			return fmt.Errorf("invalid priority: %s. Use: low, medium, high", editPriority)
		}
		todos[idx].Priority = p
		updated = true
	}

	if cmd.Flags().Changed("status") {
		status := types.Status(strings.ToLower(editStatus))
		if !status.IsValid() {
			return &types.InvalidStatusError{Status: editStatus}
		}
		todos[idx].Status = status
		updated = true
	}

	if editClearPaths {
		todos[idx].Context.Paths = []string{}
		updated = true
	} else if cmd.Flags().Changed("path") {
		todos[idx].Context.Paths = editPaths
		updated = true
	}

	if !updated {
		return fmt.Errorf("no updates provided; set --text, --status, --priority, or --path")
	}

	todos[idx].UpdatedAt = time.Now()

	if err := storage.SaveTodos(projectRoot, todos); err != nil {
		return fmt.Errorf("failed to save todos: %w", err)
	}

	terminal.PrintSuccess("Todo updated")
	fmt.Printf("  %s%s%s\n\n", terminal.Dim, todos[idx].Text, terminal.Reset)
	return nil
}
