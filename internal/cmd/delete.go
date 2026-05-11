package cmd

import (
	"fmt"
	"sort"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <id|index> [id|index...]",
	Aliases: []string{"del", "rm"},
	Short:   "Delete one or more todos",
	Long:    "Remove todos by list index or ID. Multiple arguments are supported.",
	Args:    cobra.MinimumNArgs(1),
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

	return storage.WithLock(projectRoot, func() error {
		todos, err := storage.LoadTodos(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load todos: %w", err)
		}

		var toDelete []int
		for _, idOrIndex := range args {
			target, idx := storage.FindTodoByIDOrIndex(todos, idOrIndex)
			if target == nil {
				terminal.PrintWarning(fmt.Sprintf("Not found: %s", idOrIndex))
				continue
			}
			toDelete = append(toDelete, idx)
			terminal.PrintSuccess(fmt.Sprintf("Deleted: %s", target.Text))
		}

		if len(toDelete) == 0 {
			fmt.Println()
			return nil
		}

		seen := make(map[int]bool, len(toDelete))
		var unique []int
		for _, idx := range toDelete {
			if !seen[idx] {
				seen[idx] = true
				unique = append(unique, idx)
			}
		}
		sort.Sort(sort.Reverse(sort.IntSlice(unique)))
		for _, idx := range unique {
			todos = storage.DeleteTodo(todos, idx)
		}

		if err := storage.SaveTodos(projectRoot, todos); err != nil {
			return fmt.Errorf("failed to save todos: %w", err)
		}

		fmt.Println()
		return nil
	})
}
