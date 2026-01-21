package cmd

import (
	"fmt"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <id|index>",
	Aliases: []string{"del", "rm"},
	Short:   "Delete a todo",
	Long:    "Remove a todo by its list index or ID without opening the interactive list.",
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	target, idx := storage.FindTodoByIDOrIndex(todos, args[0])
	if target == nil {
		return &types.TodoNotFoundError{ID: args[0]}
	}

	todos = storage.DeleteTodo(todos, idx)

	if err := storage.SaveTodos(projectRoot, todos); err != nil {
		return fmt.Errorf("failed to save todos: %w", err)
	}

	terminal.PrintSuccess(fmt.Sprintf("Deleted: %s", target.Text))
	fmt.Println()
	return nil
}
