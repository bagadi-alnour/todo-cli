package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var watchInterval int

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for todo changes and print updates",
	Long: `Watch the .todos/ directory for changes and print updates to stdout.

Useful for editor integrations, status bars, or piping into other tools.
Outputs JSON events on each change.`,
	Example: `  todo watch                  # Poll every 2 seconds
  todo watch --interval 5     # Poll every 5 seconds`,
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().IntVar(&watchInterval, "interval", 2, "Poll interval in seconds")
}

func runWatch(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	todosPath := filepath.Join(projectRoot, storage.TodosDir, storage.TodosFile)
	ticker := time.NewTicker(time.Duration(watchInterval) * time.Second)
	defer ticker.Stop()

	var lastMod time.Time
	var lastCount int

	emit := func(todos []types.Todo, event string) {
		payload := map[string]any{
			"event":     event,
			"count":     len(todos),
			"timestamp": time.Now().Format(time.RFC3339),
		}
		open := 0
		done := 0
		for _, t := range todos {
			switch t.Status {
			case types.StatusOpen:
				open++
			case types.StatusDone:
				done++
			}
		}
		payload["open"] = open
		payload["done"] = done

		data, _ := json.Marshal(payload)
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	}

	// Initial read
	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}
	lastCount = len(todos)
	if info, err := os.Stat(todosPath); err == nil {
		lastMod = info.ModTime()
	}
	emit(todos, "init")
	terminal.PrintInfo(fmt.Sprintf("Watching %s (every %ds, Ctrl+C to stop)", todosPath, watchInterval))

	for range ticker.C {
		info, err := os.Stat(todosPath)
		if err != nil {
			continue
		}
		if !info.ModTime().After(lastMod) {
			continue
		}
		lastMod = info.ModTime()

		todos, err := storage.LoadTodos(projectRoot)
		if err != nil {
			continue
		}

		event := "change"
		if len(todos) > lastCount {
			event = "added"
		} else if len(todos) < lastCount {
			event = "removed"
		}
		lastCount = len(todos)

		emit(todos, event)
	}

	return nil
}
