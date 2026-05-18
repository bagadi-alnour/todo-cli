package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import todos from a JSON file",
	Long: `Import todos from a previously exported JSON file.

Imported todos are merged into the current project. Duplicate IDs are skipped.`,
	Example: `  todo import backup.json
  todo import ../other-project/.todos/todos.json`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var incoming []types.Todo

	var todoFile types.TodoFile
	if err := json.Unmarshal(data, &todoFile); err == nil && todoFile.Version > 0 {
		incoming = todoFile.Todos
	} else {
		if err := json.Unmarshal(data, &incoming); err != nil {
			return fmt.Errorf("failed to parse import file (expected JSON array or {version, todos}): %w", err)
		}
	}

	if len(incoming) == 0 {
		terminal.PrintInfo("Import file contains no todos")
		fmt.Println()
		return nil
	}

	return storage.WithLock(projectRoot, func() error {
		existing, err := storage.LoadTodos(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load todos: %w", err)
		}

		idSet := make(map[string]struct{}, len(existing))
		for _, t := range existing {
			idSet[t.ID] = struct{}{}
		}

		creator, err := storage.CurrentUserSlug()
		if err != nil {
			return err
		}

		added := 0
		skipped := 0
		for _, t := range incoming {
			if _, dup := idSet[t.ID]; dup {
				skipped++
				continue
			}
			if strings.TrimSpace(t.CreatedBy) == "" {
				t.CreatedBy = creator
			}
			existing = append(existing, t)
			idSet[t.ID] = struct{}{}
			added++
		}

		if added > 0 {
			if err := storage.SaveTodos(projectRoot, existing); err != nil {
				return fmt.Errorf("failed to save todos: %w", err)
			}
		}

		terminal.PrintSuccess(fmt.Sprintf("Imported %d todo(s)", added))
		if skipped > 0 {
			fmt.Printf("  %s%d duplicate(s) skipped%s\n", terminal.Dim, skipped, terminal.Reset)
		}
		fmt.Println()
		return nil
	})
}
