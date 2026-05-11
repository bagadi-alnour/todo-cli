package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	nextAll      bool
	nextPriority string
	nextPath     string
	nextTags     []string
	nextJSON     bool
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Recommend the next todo to work on",
	Long: `Pick the best next todo based on urgency and priority.

Ranking rules:
  1. Overdue items first
  2. Then soonest due date
  3. Then priority (high > medium > low)
  4. Then oldest task`,
	Example: `  todo next
  todo next --tag backend
  todo next --path src/auth --priority high
  todo next --json`,
	RunE: runNext,
}

func init() {
	rootCmd.AddCommand(nextCmd)

	nextCmd.Flags().BoolVarP(&nextAll, "all", "a", false, "Include non-done todos (open + blocked + waiting + tech-debt)")
	nextCmd.Flags().StringVar(&nextPriority, "priority", "", "Filter by priority: low, medium, high")
	nextCmd.Flags().StringVarP(&nextPath, "path", "p", "", "Filter by path prefix")
	nextCmd.Flags().StringArrayVarP(&nextTags, "tag", "t", []string{}, "Filter by tag(s), OR matching (repeat or comma-separate)")
	nextCmd.Flags().BoolVar(&nextJSON, "json", false, "Output result as JSON")

	registerPathFlagCompletion(nextCmd, "path")
}

func runNext(cmd *cobra.Command, args []string) error {
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

	candidates := make([]types.Todo, 0, len(todos))
	for _, t := range todos {
		if nextAll {
			if t.Status != types.StatusDone {
				candidates = append(candidates, t)
			}
		} else if t.Status == types.StatusOpen {
			candidates = append(candidates, t)
		}
	}

	if nextPath != "" {
		candidates = storage.FilterTodosByPath(candidates, nextPath)
	}
	if len(nextTags) > 0 {
		candidates = storage.FilterTodosByTags(candidates, normalizeTags(nextTags))
	}
	if nextPriority != "" {
		p := types.Priority(strings.ToLower(nextPriority))
		if !p.IsValid() {
			return fmt.Errorf("invalid priority: %s. Use: low, medium, high", nextPriority)
		}
		candidates = storage.FilterTodosByPriority(candidates, p)
	}

	if len(candidates) == 0 {
		if nextJSON {
			payload := map[string]any{"todo": nil, "message": "No matching todo found"}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(payload)
		}
		terminal.PrintInfo("No matching todo found")
		fmt.Println()
		return nil
	}

	now := time.Now()
	sortTodosForExecution(candidates, now)
	selected := candidates[0]

	if nextJSON {
		payload := map[string]any{
			"todo":   selected,
			"reason": nextReason(selected, now),
			"count":  len(candidates),
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	terminal.PrintHeader("NEXT TODO", "👉")
	priorityLabel, priorityColor := priorityVisual(selected.Priority)
	fmt.Printf("  %s%s%s %s%s%s %s%s%s\n\n",
		terminal.StatusColor(string(selected.Status)), terminal.StatusIcon(string(selected.Status)), terminal.Reset,
		priorityColor, priorityLabel, terminal.Reset,
		terminal.Bold, selected.Text, terminal.Reset)

	fmt.Printf("  %sReason:%s %s\n", terminal.Dim, terminal.Reset, nextReason(selected, now))
	shortID := selected.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	fmt.Printf("  %sID:%s %s\n", terminal.Dim, terminal.Reset, shortID)
	if selected.DueAt != nil {
		color := terminal.Cyan
		if isOverdueDueDate(selected.DueAt, now) {
			color = terminal.BrightRed
		}
		fmt.Printf("  %sDue:%s %s%s%s\n", terminal.Dim, terminal.Reset, color, formatDueLabel(selected.DueAt, now), terminal.Reset)
	}
	if selected.Notes != "" {
		fmt.Printf("  %sNotes:%s %s\n", terminal.Dim, terminal.Reset, selected.Notes)
	}
	if len(selected.Tags) > 0 {
		fmt.Printf("  %sTags:%s %s\n", terminal.Dim, terminal.Reset, strings.Join(selected.Tags, ", "))
	}
	if len(selected.Context.Paths) > 0 {
		fmt.Printf("  %sPaths:%s %s\n", terminal.Dim, terminal.Reset, strings.Join(selected.Context.Paths, ", "))
	}
	if selected.Context.Branch != "" {
		fmt.Printf("  %sBranch:%s %s\n", terminal.Dim, terminal.Reset, selected.Context.Branch)
	}
	fmt.Println()
	fmt.Printf("  %s💡 Run %stodo done %s%s %swhen finished%s\n\n",
		terminal.Dim, terminal.BrightCyan, shortID, terminal.Reset+terminal.Dim, terminal.Dim, terminal.Reset)

	return nil
}

func nextReason(todo types.Todo, now time.Time) string {
	if isOverdueDueDate(todo.DueAt, now) {
		overdue := now.Sub(*todo.DueAt)
		days := int(overdue.Hours() / 24)
		if days >= 2 {
			return fmt.Sprintf("overdue by %d days, %s priority", days, todo.Priority)
		} else if days == 1 {
			return fmt.Sprintf("overdue by 1 day, %s priority", todo.Priority)
		}
		hours := int(overdue.Hours())
		if hours > 0 {
			return fmt.Sprintf("overdue by %d hours, %s priority", hours, todo.Priority)
		}
		return fmt.Sprintf("overdue, %s priority", todo.Priority)
	}
	if todo.DueAt != nil {
		diff := todo.DueAt.Sub(now)
		hours := int(diff.Hours())
		if hours < 1 {
			return fmt.Sprintf("due in less than an hour, %s priority", todo.Priority)
		} else if hours < 24 {
			return fmt.Sprintf("due in %d hours, %s priority", hours, todo.Priority)
		}
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("due in %d days, %s priority", days, todo.Priority)
	}
	if priorityWeight(todo.Priority) >= priorityWeight(types.PriorityHigh) {
		age := int(now.Sub(todo.CreatedAt).Hours() / 24)
		if age > 0 {
			return fmt.Sprintf("high priority, open for %d days", age)
		}
		return "high priority, created today"
	}
	age := int(now.Sub(todo.CreatedAt).Hours() / 24)
	if age > 0 {
		return fmt.Sprintf("%s priority, oldest open (%d days)", todo.Priority, age)
	}
	return fmt.Sprintf("%s priority, created today", todo.Priority)
}
