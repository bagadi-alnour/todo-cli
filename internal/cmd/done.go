package cmd

import (
	"fmt"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done <id|index>",
	Short: "Mark a todo as done",
	Long: `Mark a todo as completed.

You can specify a todo by its ID (or partial ID) or by its index number
as shown in 'todo list'.`,
	Example: `  todo done 1           # Mark todo #1 as done
  todo done abc123      # Mark todo with ID starting with abc123
  todo done a3f9        # Partial ID match`,
	Args: cobra.ExactArgs(1),
	RunE: runDone,
}

func init() {
	rootCmd.AddCommand(doneCmd)
}

func runDone(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	// Load todos
	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	// Find todo by ID or index
	idOrIndex := args[0]
	todo, idx := storage.FindTodoByIDOrIndex(todos, idOrIndex)
	if todo == nil {
		return &types.TodoNotFoundError{ID: idOrIndex}
	}

	// Check if already done
	if todo.Status == types.StatusDone {
		terminal.PrintWarning(fmt.Sprintf("Already done: %s", todo.Text))
		fmt.Println()
		return nil
	}

	// Mark as done
	todos[idx].MarkDone()

	// Save
	if err := storage.SaveTodos(projectRoot, todos); err != nil {
		return fmt.Errorf("failed to save todos: %w", err)
	}

	terminal.PrintSuccess(fmt.Sprintf("Completed: %s", todo.Text))
	fmt.Println()

	// Show remaining open todos count
	openCount := 0
	for _, t := range todos {
		if t.Status == types.StatusOpen {
			openCount++
		}
	}

	if openCount == 0 {
		fmt.Printf("  %s🎉 All todos complete! Great job!%s\n\n", terminal.BrightGreen, terminal.Reset)
	} else {
		fmt.Printf("  %s%d todo(s) remaining%s\n\n", terminal.Dim, openCount, terminal.Reset)
	}

	return nil
}
