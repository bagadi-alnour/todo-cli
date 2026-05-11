package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	archiveBefore string
	archiveJSON   bool
)

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Move completed todos to an archive file",
	Long: `Move todos with status "done" from the active list into .todos/archive.json.

This keeps your main todo list clean while preserving history.`,
	Example: `  todo archive                         # Archive all done todos
  todo archive --before 2025-12-31     # Archive done todos completed before date
  todo archive --json                  # Output archived items as JSON`,
	RunE: runArchive,
}

func init() {
	rootCmd.AddCommand(archiveCmd)
	archiveCmd.Flags().StringVar(&archiveBefore, "before", "", "Only archive todos completed before this date (YYYY-MM-DD)")
	archiveCmd.Flags().BoolVar(&archiveJSON, "json", false, "Output archived items as JSON")
}

func runArchive(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}
	Verbosef("project root: %s", projectRoot)

	var cutoff *time.Time
	if archiveBefore != "" {
		t, err := time.Parse("2006-01-02", archiveBefore)
		if err != nil {
			return fmt.Errorf("invalid --before date (expected YYYY-MM-DD): %w", err)
		}
		eod := t.Add(24*time.Hour - time.Nanosecond)
		cutoff = &eod
	}

	return storage.WithLock(projectRoot, func() error {
		todos, err := storage.LoadTodos(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load todos: %w", err)
		}

		var remaining []types.Todo
		var archived []types.Todo

		for _, t := range todos {
			if t.Status != types.StatusDone {
				remaining = append(remaining, t)
				continue
			}
			if cutoff != nil && t.CompletedAt != nil && t.CompletedAt.After(*cutoff) {
				remaining = append(remaining, t)
				continue
			}
			archived = append(archived, t)
		}

		if len(archived) == 0 {
			if archiveJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{"archived": []types.Todo{}, "count": 0})
			}
			terminal.PrintInfo("No completed todos to archive")
			fmt.Println()
			return nil
		}

		existingArchive, err := storage.LoadArchive(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load archive: %w", err)
		}

		existingArchive = append(existingArchive, archived...)
		if err := storage.SaveArchive(projectRoot, existingArchive); err != nil {
			return fmt.Errorf("failed to save archive: %w", err)
		}

		if err := storage.SaveTodos(projectRoot, remaining); err != nil {
			return fmt.Errorf("failed to save todos: %w", err)
		}

		if archiveJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{"archived": archived, "count": len(archived)})
		}

		terminal.PrintSuccess(fmt.Sprintf("Archived %d completed todo(s)", len(archived)))
		fmt.Printf("  %s%d remaining in active list%s\n\n", terminal.Dim, len(remaining), terminal.Reset)

		return nil
	})
}
