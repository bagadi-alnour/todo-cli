package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	searchStatus string
	searchPath   string
	searchTags   []string
	searchJSON   bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search todos by text, tags, or paths",
	Long: `Full-text search across todo text, tags, and file paths.

The query is matched case-insensitively against the todo text, all tags,
and all associated paths. Additional filters can narrow the results.`,
	Example: `  todo search "auth"                 # Search for "auth" in text/tags/paths
  todo search "bug" --status open    # Search open todos only
  todo search "api" --tag backend    # Search within tagged todos
  todo search "fix" --json           # Machine-readable output`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVarP(&searchStatus, "status", "s", "", "Filter by status")
	searchCmd.Flags().StringVarP(&searchPath, "path", "p", "", "Filter by path prefix")
	searchCmd.Flags().StringArrayVarP(&searchTags, "tag", "t", []string{}, "Filter by tag(s)")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")

	registerPathFlagCompletion(searchCmd, "path")
}

func matchesQuery(todo types.Todo, query string) bool {
	q := strings.ToLower(query)

	if strings.Contains(strings.ToLower(todo.Text), q) {
		return true
	}
	if todo.Notes != "" && strings.Contains(strings.ToLower(todo.Notes), q) {
		return true
	}
	for _, tag := range todo.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	for _, p := range todo.Context.Paths {
		if strings.Contains(strings.ToLower(p), q) {
			return true
		}
	}
	return false
}

func runSearch(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}
	Verbosef("project root: %s", projectRoot)

	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}
	Verbosef("loaded %d todo(s)", len(todos))

	query := args[0]

	// Apply text search
	var results []types.Todo
	for _, t := range todos {
		if matchesQuery(t, query) {
			results = append(results, t)
		}
	}

	// Apply additional filters
	if searchStatus != "" {
		status := types.Status(strings.ToLower(searchStatus))
		if !status.IsValid() {
			return &types.InvalidStatusError{Status: searchStatus}
		}
		results = storage.FilterTodosByStatus(results, status)
	}
	if searchPath != "" {
		results = storage.FilterTodosByPath(results, searchPath)
	}
	if len(searchTags) > 0 {
		results = storage.FilterTodosByTags(results, normalizeTags(searchTags))
	}

	storage.SortTodosByPriority(results)

	if searchJSON {
		payload := map[string]any{
			"query":   query,
			"results": results,
			"count":   len(results),
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	if len(results) == 0 {
		terminal.PrintInfo(fmt.Sprintf("No todos matching %q", query))
		fmt.Println()
		return nil
	}

	fmt.Printf("\n  %s%s🔍 Search: %q%s  %s(%d result(s))%s\n", terminal.Bold, terminal.BrightCyan, query, terminal.Reset, terminal.Dim, len(results), terminal.Reset)
	fmt.Printf("  %s─────────────────────────────────────────%s\n\n", terminal.Dim, terminal.Reset)

	for i, todo := range results {
		statusColor := terminal.StatusColor(string(todo.Status))
		checkbox := terminal.StatusIcon(string(todo.Status))
		priorityLabel, priorityColor := priorityVisual(todo.Priority)

		textStyle := ""
		if todo.Status == types.StatusDone {
			textStyle = terminal.Dim
		}

		fmt.Printf("  %s%d.%s %s%s%s %s%s%s %s%s%s\n",
			terminal.Dim, i+1, terminal.Reset,
			statusColor, checkbox, terminal.Reset,
			priorityColor, priorityLabel, terminal.Reset,
			textStyle, todo.Text, terminal.Reset)

		if todo.Notes != "" {
			fmt.Printf("     %s📝 %s%s\n", terminal.Dim, terminal.Truncate(todo.Notes, 60), terminal.Reset)
		}
		if len(todo.Context.Paths) > 0 {
			fmt.Printf("     %s📁 %s%s\n", terminal.Dim, strings.Join(todo.Context.Paths, ", "), terminal.Reset)
		}
		if len(todo.Tags) > 0 {
			fmt.Printf("     %s🏷️ %s%s\n", terminal.Dim, strings.Join(todo.Tags, ", "), terminal.Reset)
		}
	}
	fmt.Println()

	return nil
}
