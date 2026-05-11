package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var exportFormat string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export todos to JSON or Markdown",
	Long: `Export the current project's todos in a portable format.

Supported formats: json (default), markdown.`,
	Example: `  todo export
  todo export --format markdown
  todo export --format json > backup.json`,
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Output format: json, markdown")
}

func runExport(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	switch strings.ToLower(exportFormat) {
	case "json":
		return exportJSON(cmd, todos)
	case "markdown", "md":
		return exportMarkdown(cmd, todos)
	default:
		return fmt.Errorf("unsupported format: %s. Use: json, markdown", exportFormat)
	}
}

func exportJSON(cmd *cobra.Command, todos []types.Todo) error {
	out := &types.TodoFile{Version: 1, Todos: todos}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func exportMarkdown(cmd *cobra.Command, todos []types.Todo) error {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "# Todos")

	groups := map[types.Priority][]types.Todo{}
	order := []types.Priority{types.PriorityHigh, types.PriorityMedium, types.PriorityLow}
	for _, t := range todos {
		groups[t.Priority] = append(groups[t.Priority], t)
	}

	for _, p := range order {
		items := groups[p]
		if len(items) == 0 {
			continue
		}
		label := string(p)
		if len(label) > 0 {
			label = strings.ToUpper(label[:1]) + label[1:]
		}
		fmt.Fprintf(w, "\n## %s priority\n\n", label)
		for _, t := range items {
			check := " "
			if t.Status == types.StatusDone {
				check = "x"
			}
			line := fmt.Sprintf("- [%s] %s", check, t.Text)
			if len(t.Context.Paths) > 0 {
				line += " `" + strings.Join(t.Context.Paths, "`, `") + "`"
			}
			for _, tag := range t.Tags {
				line += " #" + tag
			}
			fmt.Fprintln(w, line)
		}
	}

	return nil
}
