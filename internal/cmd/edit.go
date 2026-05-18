package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	editText           string
	editPaths          []string
	editClearPaths     bool
	editPriority       string
	editStatus         string
	editTags           []string
	editAddTags        []string
	editRemoveTags     []string
	editClearTags      bool
	editDue            string
	editClearDue       bool
	editNotes          string
	editClearNotes     bool
	editBlockedBy      []string
	editBlocks         []string
	editClearBlockedBy bool
	editClearBlocks    bool
	editRecur          string
	editClearRecur     bool
	editAssign         string
	editClearAssignee  bool
)

var editCmd = &cobra.Command{
	Use:   "edit <id|index>",
	Short: "Edit a todo's text, status, priority, or paths",
	Long: `Update an existing todo without opening the interactive list.

You can change the text, status, priority, or replace/clear any paths.`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVar(&editText, "text", "", "New todo text")
	editCmd.Flags().StringArrayVarP(&editPaths, "path", "p", []string{}, "Replace paths (can be provided multiple times)")
	editCmd.Flags().BoolVar(&editClearPaths, "clear-paths", false, "Remove all associated paths")
	editCmd.Flags().StringVar(&editPriority, "priority", "", "Set priority: low, medium, high")
	editCmd.Flags().StringVar(&editStatus, "status", "", "Set status: open, done, blocked, waiting, tech-debt")
	editCmd.Flags().StringArrayVarP(&editTags, "tag", "t", []string{}, "Replace tags (repeat or comma-separate)")
	editCmd.Flags().StringArrayVar(&editAddTags, "add-tag", []string{}, "Add tag(s) without replacing existing tags")
	editCmd.Flags().StringArrayVar(&editRemoveTags, "remove-tag", []string{}, "Remove tag(s)")
	editCmd.Flags().BoolVar(&editClearTags, "clear-tags", false, "Remove all tags")
	editCmd.Flags().StringVar(&editDue, "due", "", "Set due date/time (YYYY-MM-DD, YYYY-MM-DDTHH:MM, RFC3339, today, tomorrow, +2d)")
	editCmd.Flags().BoolVar(&editClearDue, "clear-due", false, "Remove due date")
	editCmd.Flags().StringVar(&editNotes, "notes", "", "Set notes/description")
	editCmd.Flags().BoolVar(&editClearNotes, "clear-notes", false, "Remove notes")
	editCmd.Flags().StringArrayVar(&editBlockedBy, "blocked-by", []string{}, "Set blocker IDs (replaces existing)")
	editCmd.Flags().StringArrayVar(&editBlocks, "blocks", []string{}, "Set IDs this todo blocks (replaces existing)")
	editCmd.Flags().BoolVar(&editClearBlockedBy, "clear-blocked-by", false, "Remove all blockers")
	editCmd.Flags().BoolVar(&editClearBlocks, "clear-blocks", false, "Remove all blocks")
	editCmd.Flags().StringVar(&editRecur, "recur", "", "Set recurrence: daily, weekly, monthly")
	editCmd.Flags().BoolVar(&editClearRecur, "clear-recur", false, "Remove recurrence")
	editCmd.Flags().StringVar(&editAssign, "assign", "", "Assign to a git contributor (name, email prefix, or me)")
	editCmd.Flags().BoolVar(&editClearAssignee, "clear-assignee", false, "Remove assignee")

	registerPathFlagCompletion(editCmd, "path")
	registerAssigneeFlagCompletion(editCmd, "assign")
}

func runEdit(cmd *cobra.Command, args []string) error {
	if editClearDue && cmd.Flags().Changed("due") {
		return fmt.Errorf("cannot use --due with --clear-due")
	}
	if editClearNotes && cmd.Flags().Changed("notes") {
		return fmt.Errorf("cannot use --notes with --clear-notes")
	}
	if editClearAssignee && cmd.Flags().Changed("assign") {
		return fmt.Errorf("cannot use --assign with --clear-assignee")
	}

	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	return storage.WithLock(projectRoot, func() error {
		todos, err := storage.LoadTodos(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load todos: %w", err)
		}

		todo, idx := storage.FindTodoByIDOrIndex(todos, args[0])
		if todo == nil {
			return &types.TodoNotFoundError{ID: args[0]}
		}

		updated := false

		if cmd.Flags().Changed("text") {
			text := strings.TrimSpace(editText)
			if text == "" {
				return fmt.Errorf("todo text cannot be empty")
			}
			todos[idx].Text = text
			updated = true
		}

		if cmd.Flags().Changed("priority") {
			p := types.Priority(strings.ToLower(editPriority))
			if !p.IsValid() {
				return fmt.Errorf("invalid priority: %s. Use: low, medium, high", editPriority)
			}
			todos[idx].Priority = p
			updated = true
		}

		if cmd.Flags().Changed("status") {
			status := types.Status(strings.ToLower(editStatus))
			if !status.IsValid() {
				return &types.InvalidStatusError{Status: editStatus}
			}
			switch status {
			case types.StatusDone:
				todos[idx].MarkDone()
			case types.StatusOpen:
				todos[idx].MarkOpen()
			default:
				todos[idx].Status = status
				todos[idx].CompletedAt = nil
			}
			updated = true
		}

		if editClearPaths {
			todos[idx].Context.Paths = []string{}
			updated = true
		} else if cmd.Flags().Changed("path") {
			todos[idx].Context.Paths = normalizePaths(editPaths)
			updated = true
		}

		if editClearTags {
			todos[idx].Tags = nil
			updated = true
		}
		if cmd.Flags().Changed("tag") {
			todos[idx].Tags = normalizeTags(editTags)
			updated = true
		}
		if cmd.Flags().Changed("add-tag") {
			todos[idx].Tags = mergeTags(todos[idx].Tags, editAddTags)
			updated = true
		}
		if cmd.Flags().Changed("remove-tag") {
			todos[idx].Tags = removeTags(todos[idx].Tags, editRemoveTags)
			updated = true
		}

		if editClearDue {
			todos[idx].DueAt = nil
			updated = true
		} else if cmd.Flags().Changed("due") {
			dueAt, err := parseDueDateInput(editDue, time.Now())
			if err != nil {
				return err
			}
			todos[idx].DueAt = dueAt
			updated = true
		}

		if editClearNotes {
			todos[idx].Notes = ""
			updated = true
		} else if cmd.Flags().Changed("notes") {
			todos[idx].Notes = editNotes
			updated = true
		}

		if editClearBlockedBy {
			todos[idx].BlockedBy = nil
			updated = true
		} else if cmd.Flags().Changed("blocked-by") {
			todos[idx].BlockedBy = editBlockedBy
			updated = true
		}
		if editClearBlocks {
			todos[idx].Blocks = nil
			updated = true
		} else if cmd.Flags().Changed("blocks") {
			todos[idx].Blocks = editBlocks
			updated = true
		}

		if editClearRecur {
			todos[idx].Recur = ""
			updated = true
		} else if cmd.Flags().Changed("recur") {
			r := types.Recurrence(strings.ToLower(editRecur))
			if !r.IsValid() {
				return fmt.Errorf("invalid recurrence: %s. Use: daily, weekly, monthly", editRecur)
			}
			todos[idx].Recur = r
			updated = true
		}

		if editClearAssignee {
			todos[idx].Assignee = ""
			updated = true
		} else if cmd.Flags().Changed("assign") {
			email, err := resolveAssignee(projectRoot, editAssign)
			if err != nil {
				return err
			}
			todos[idx].Assignee = email
			updated = true
		}

		if !updated {
			return fmt.Errorf("no updates provided; use --text, --status, --priority, --path, --tag, --due, --notes, --blocked-by, --blocks, --recur, --assign, or clear flags")
		}

		todos[idx].UpdatedAt = time.Now()

		if err := storage.SaveTodos(projectRoot, todos); err != nil {
			return fmt.Errorf("failed to save todos: %w", err)
		}

		terminal.PrintSuccess("Todo updated")
		fmt.Printf("  %s%s%s\n\n", terminal.Dim, todos[idx].Text, terminal.Reset)
		return nil
	})
}
