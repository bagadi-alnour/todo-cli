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

var statusCmd = &cobra.Command{
	Use:     "status <id|index> <status>",
	Aliases: []string{"set-status"},
	Short:   "Update the status of a todo",
	Long: `Set the status of a todo without opening the interactive list.

Valid statuses: open, done, blocked, waiting, tech-debt.`,
	Args: cobra.ExactArgs(2),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	newStatus := types.Status(strings.ToLower(args[1]))
	if !newStatus.IsValid() {
		return &types.InvalidStatusError{Status: args[1]}
	}

	if target.Status == newStatus {
		terminal.PrintInfo("Status unchanged")
		fmt.Println()
		return nil
	}

	todos[idx].Status = newStatus
	todos[idx].UpdatedAt = time.Now()

	if err := storage.SaveTodos(projectRoot, todos); err != nil {
		return fmt.Errorf("failed to save todos: %w", err)
	}

	terminal.PrintSuccess(fmt.Sprintf("Status set to %s: %s", newStatus, target.Text))
	fmt.Println()
	return nil
}
