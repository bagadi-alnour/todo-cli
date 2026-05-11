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

var showJSON bool

var showCmd = &cobra.Command{
	Use:   "show <id|index>",
	Short: "Show full details of a todo",
	Long:  `Display all fields of a single todo item, including notes, context, and timestamps.`,
	Example: `  todo show 1
  todo show abc123
  todo show 1 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
}

func runShow(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	todo, _ := storage.FindTodoByIDOrIndex(todos, args[0])
	if todo == nil {
		return &types.TodoNotFoundError{ID: args[0]}
	}

	if showJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(todo)
	}

	now := time.Now()
	shortID := todo.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	priorityLabel, priorityColor := priorityVisual(todo.Priority)
	fmt.Printf("\n  %s%s%s %s%s%s %s%s%s\n",
		terminal.StatusColor(string(todo.Status)), terminal.StatusIcon(string(todo.Status)), terminal.Reset,
		priorityColor, priorityLabel, terminal.Reset,
		terminal.Bold, todo.Text, terminal.Reset)
	fmt.Println()

	fmt.Printf("  %sID:%s       %s\n", terminal.Dim, terminal.Reset, shortID)
	fmt.Printf("  %sStatus:%s   %s\n", terminal.Dim, terminal.Reset, todo.Status)
	fmt.Printf("  %sPriority:%s %s\n", terminal.Dim, terminal.Reset, todo.Priority)

	if todo.Notes != "" {
		fmt.Printf("  %sNotes:%s    %s\n", terminal.Dim, terminal.Reset, todo.Notes)
	}
	if len(todo.Tags) > 0 {
		fmt.Printf("  %sTags:%s     %s\n", terminal.Dim, terminal.Reset, strings.Join(todo.Tags, ", "))
	}
	if todo.DueAt != nil {
		color := terminal.Cyan
		if isOverdueDueDate(todo.DueAt, now) {
			color = terminal.BrightRed
		}
		fmt.Printf("  %sDue:%s      %s%s%s\n", terminal.Dim, terminal.Reset, color, formatDueLabel(todo.DueAt, now), terminal.Reset)
	}
	if len(todo.Context.Paths) > 0 {
		fmt.Printf("  %sPaths:%s    %s\n", terminal.Dim, terminal.Reset, strings.Join(todo.Context.Paths, ", "))
	}
	if todo.Context.Branch != "" {
		fmt.Printf("  %sBranch:%s   %s\n", terminal.Dim, terminal.Reset, todo.Context.Branch)
	}
	if todo.Context.Commit != "" {
		fmt.Printf("  %sCommit:%s   %s\n", terminal.Dim, terminal.Reset, todo.Context.Commit)
	}
	if len(todo.BlockedBy) > 0 {
		fmt.Printf("  %sBlocked by:%s %s\n", terminal.Dim, terminal.Reset, strings.Join(todo.BlockedBy, ", "))
	}
	if len(todo.Blocks) > 0 {
		fmt.Printf("  %sBlocks:%s   %s\n", terminal.Dim, terminal.Reset, strings.Join(todo.Blocks, ", "))
	}
	if todo.Recur != "" {
		fmt.Printf("  %sRecur:%s    %s\n", terminal.Dim, terminal.Reset, todo.Recur)
	}

	fmt.Printf("  %sCreated:%s  %s\n", terminal.Dim, terminal.Reset, todo.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  %sUpdated:%s  %s\n", terminal.Dim, terminal.Reset, todo.UpdatedAt.Format(time.RFC3339))
	if todo.CompletedAt != nil {
		fmt.Printf("  %sDone:%s     %s\n", terminal.Dim, terminal.Reset, todo.CompletedAt.Format(time.RFC3339))
	}
	fmt.Println()

	return nil
}
