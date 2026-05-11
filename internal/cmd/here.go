package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var hereJSON bool

var hereCmd = &cobra.Command{
	Use:   "here",
	Short: "Show todos related to the current directory",
	Long: `Display todos whose paths overlap with the current working directory.

Useful when you cd into a subdirectory and want to see what's relevant.`,
	Example: `  cd src/auth && todo here
  todo here --json`,
	RunE: runHere,
}

func init() {
	rootCmd.AddCommand(hereCmd)
	hereCmd.Flags().BoolVar(&hereJSON, "json", false, "Output as JSON")
}

func runHere(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	cwd, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}

	relDir, err := filepath.Rel(projectRoot, cwd)
	if err != nil {
		relDir = "."
	}
	if relDir == "." {
		relDir = ""
	}

	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	var matched []types.Todo
	for _, t := range todos {
		if t.Status == types.StatusDone {
			continue
		}
		for _, p := range t.Context.Paths {
			if relDir == "" || p == relDir || strings.HasPrefix(p, relDir+"/") || strings.HasPrefix(relDir, p+"/") {
				matched = append(matched, t)
				break
			}
		}
	}

	displayDir := relDir
	if displayDir == "" {
		displayDir = "."
	}

	if hereJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{"directory": displayDir, "todos": matched, "count": len(matched)})
	}

	terminal.PrintHeader(fmt.Sprintf("DIRECTORY: %s", displayDir), "📂")

	if len(matched) == 0 {
		fmt.Printf("  %sNo open todos for this directory%s\n\n", terminal.Dim, terminal.Reset)
		return nil
	}

	storage.SortTodosByPriority(matched)
	for i, t := range matched {
		priorityLabel, priorityColor := priorityVisual(t.Priority)
		paths := strings.Join(t.Context.Paths, ", ")
		fmt.Printf("  %d. %s%s%s %s%s%s %s%s%s\n",
			i+1, priorityColor, priorityLabel, terminal.Reset,
			terminal.Bold, t.Text, terminal.Reset,
			terminal.Dim, paths, terminal.Reset)
	}
	fmt.Println()

	return nil
}
