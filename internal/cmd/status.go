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
	Use:     "status <id|index> [id|index...] <status>",
	Aliases: []string{"set-status"},
	Short:   "Update the status of one or more todos",
	Long: `Set the status of todos without opening the interactive list.
The last argument is the target status. All preceding arguments are todo IDs or indices.

Valid statuses: open, done, blocked, waiting, tech-debt.`,
	Example: `  todo status 1 blocked       # Set todo #1 to blocked
  todo status 1 2 3 done      # Set multiple todos to done`,
	Args: cobra.MinimumNArgs(2),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	newStatus := types.Status(strings.ToLower(args[len(args)-1]))
	if !newStatus.IsValid() {
		return &types.InvalidStatusError{Status: args[len(args)-1]}
	}

	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	return storage.WithLock(projectRoot, func() error {
		todos, err := storage.LoadTodos(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load todos: %w", err)
		}

		targets := args[:len(args)-1]
		updated := 0

		for _, idOrIndex := range targets {
			target, idx := storage.FindTodoByIDOrIndex(todos, idOrIndex)
			if target == nil {
				terminal.PrintWarning(fmt.Sprintf("Not found: %s", idOrIndex))
				continue
			}
			if target.Status == newStatus {
				terminal.PrintInfo(fmt.Sprintf("Already %s: %s", newStatus, target.Text))
				continue
			}

			switch newStatus {
			case types.StatusDone:
				todos[idx].MarkDone()
			case types.StatusOpen:
				todos[idx].MarkOpen()
			default:
				todos[idx].Status = newStatus
				todos[idx].CompletedAt = nil
				todos[idx].UpdatedAt = time.Now()
			}

			terminal.PrintSuccess(fmt.Sprintf("Status set to %s: %s", newStatus, target.Text))
			updated++
		}

		if updated == 0 {
			fmt.Println()
			return nil
		}

		if err := storage.SaveTodos(projectRoot, todos); err != nil {
			return fmt.Errorf("failed to save todos: %w", err)
		}

		fmt.Println()
		return nil
	})
}
