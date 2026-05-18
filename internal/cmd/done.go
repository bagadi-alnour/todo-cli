package cmd

import (
	"fmt"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

func spawnRecurrence(completed types.Todo) (*types.Todo, error) {
	id, err := storage.GenerateID()
	if err != nil {
		return nil, err
	}
	next := types.NewTodo(id, completed.Text)
	next.Priority = completed.Priority
	next.Tags = completed.Tags
	next.Notes = completed.Notes
	next.Context = completed.Context
	next.Recur = completed.Recur
	next.BlockedBy = completed.BlockedBy
	next.Blocks = completed.Blocks
	next.CreatedBy = completed.CreatedBy

	base := time.Now()
	if completed.DueAt != nil {
		base = *completed.DueAt
	}
	due := completed.Recur.NextDue(base)
	next.DueAt = &due

	return next, nil
}

var doneCmd = &cobra.Command{
	Use:   "done <id|index> [id|index...]",
	Short: "Mark one or more todos as done",
	Long: `Mark todos as completed.

You can specify todos by ID (or partial ID) or by index number
as shown in 'todo list'. Multiple arguments are supported.`,
	Example: `  todo done 1           # Mark todo #1 as done
  todo done 1 2 3       # Mark multiple todos as done
  todo done abc123      # Mark todo with ID starting with abc123`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDone,
}

func init() {
	rootCmd.AddCommand(doneCmd)
}

func runDone(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	return storage.WithLock(projectRoot, func() error {
		todos, err := storage.LoadTodos(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load todos: %w", err)
		}

		completed := 0
		var recurring []types.Todo
		for _, idOrIndex := range args {
			todo, idx := storage.FindTodoByIDOrIndex(todos, idOrIndex)
			if todo == nil {
				terminal.PrintWarning(fmt.Sprintf("Not found: %s", idOrIndex))
				continue
			}
			if todo.Status == types.StatusDone {
				terminal.PrintWarning(fmt.Sprintf("Already done: %s", todo.Text))
				continue
			}
			todos[idx].MarkDone()
			terminal.PrintSuccess(fmt.Sprintf("Completed: %s", todo.Text))
			completed++

			if todo.Recur.IsValid() {
				next, err := spawnRecurrence(todos[idx])
				if err != nil {
					terminal.PrintWarning(fmt.Sprintf("Failed to create recurring copy: %v", err))
					continue
				}
				recurring = append(recurring, *next)
				terminal.PrintInfo(fmt.Sprintf("Recurring: created next %s occurrence", todo.Recur))
			}
		}

		if completed == 0 {
			fmt.Println()
			return nil
		}

		todos = append(todos, recurring...)

		if err := storage.SaveTodos(projectRoot, todos); err != nil {
			return fmt.Errorf("failed to save todos: %w", err)
		}

		openCount := 0
		for _, t := range todos {
			if t.Status == types.StatusOpen {
				openCount++
			}
		}

		fmt.Println()
		if openCount == 0 {
			fmt.Printf("  %s🎉 All todos complete! Great job!%s\n\n", terminal.BrightGreen, terminal.Reset)
		} else {
			fmt.Printf("  %s%d todo(s) remaining%s\n\n", terminal.Dim, openCount, terminal.Reset)
		}

		return nil
	})
}
