package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

var (
	focusAll bool
)

var focusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Show focused todos for current context",
	Long: `Show todos relevant to your current context.

By default, shows open todos that match the current git branch.
If not in a git repo, shows all open todos.`,
	Example: `  todo focus        # Show branch-relevant todos
  todo focus --all  # Show all open todos`,
	RunE: runFocus,
}

func init() {
	rootCmd.AddCommand(focusCmd)

	focusCmd.Flags().BoolVarP(&focusAll, "all", "a", false, "Show all open todos, not just branch-relevant")
}

func runFocus(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	// Load todos
	todos, err := storage.LoadTodos(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load todos: %w", err)
	}

	// Get open todos
	var openTodos []types.Todo
	for _, t := range todos {
		if t.Status == types.StatusOpen {
			openTodos = append(openTodos, t)
		}
	}

	// Get current branch for filtering
	currentBranch := ""
	if !focusAll && git.IsGitRepo() {
		currentBranch, _ = git.GetCurrentBranch()
	}

	// Filter by branch if applicable
	var focusedTodos []types.Todo
	if currentBranch != "" && !focusAll {
		// First, get todos matching current branch
		for _, t := range openTodos {
			if t.Context.Branch == currentBranch {
				focusedTodos = append(focusedTodos, t)
			}
		}
		// Also include todos with no branch (global todos)
		for _, t := range openTodos {
			if t.Context.Branch == "" {
				focusedTodos = append(focusedTodos, t)
			}
		}
	} else {
		focusedTodos = openTodos
	}

	// Count stats
	blockedCount := 0
	waitingCount := 0
	doneCount := 0
	for _, t := range todos {
		switch t.Status {
		case types.StatusBlocked:
			blockedCount++
		case types.StatusWaiting:
			waitingCount++
		case types.StatusDone:
			doneCount++
		}
	}

	terminal.PrintHeader("FOCUS MODE", "🎯")

	// Stats bar
	fmt.Printf("  %s%d open%s", terminal.Blue+terminal.Bold, len(focusedTodos), terminal.Reset)
	if blockedCount > 0 {
		fmt.Printf("  %s•%s  %s%d blocked%s", terminal.Dim, terminal.Reset, terminal.Yellow, blockedCount, terminal.Reset)
	}
	if waitingCount > 0 {
		fmt.Printf("  %s•%s  %s%d waiting%s", terminal.Dim, terminal.Reset, terminal.Magenta, waitingCount, terminal.Reset)
	}
	if doneCount > 0 {
		fmt.Printf("  %s•%s  %s%d done%s", terminal.Dim, terminal.Reset, terminal.Green, doneCount, terminal.Reset)
	}
	fmt.Println()

	if currentBranch != "" && !focusAll {
		fmt.Printf("  %s🌿 Branch: %s%s\n", terminal.Dim, currentBranch, terminal.Reset)
	}
	fmt.Println()

	if len(focusedTodos) == 0 {
		fmt.Printf("  %s✨ No open todos! You're all caught up! 🎉%s\n\n", terminal.BrightGreen+terminal.Bold, terminal.Reset)
		return nil
	}

	// Display todos
	for i, todo := range focusedTodos {
		var prefix string
		var textStyle string

		if i == 0 {
			// First todo - highlighted as current focus
			fmt.Printf("  %s%s─── CURRENT FOCUS ───%s\n", terminal.BrightCyan, terminal.Dim, terminal.Reset)
			prefix = fmt.Sprintf("%s%s▶ ", terminal.BrightCyan+terminal.Bold, terminal.BrightWhite)
			textStyle = terminal.Bold + terminal.BrightWhite
		} else {
			prefix = fmt.Sprintf("  %s%d.%s ", terminal.Dim, i+1, terminal.Reset)
			textStyle = ""
		}

		fmt.Printf("%s%s%s%s\n", prefix, textStyle, todo.Text, terminal.Reset)

		// Context paths
		if len(todo.Context.Paths) > 0 {
			pathColor := terminal.BrightCyan
			if i != 0 {
				pathColor = terminal.Dim
			}
			fmt.Printf("     %s📁 %s%s\n", pathColor, strings.Join(todo.Context.Paths, ", "), terminal.Reset)
		}

		// Time ago
		timeAgo := formatTimeAgo(todo.CreatedAt)
		fmt.Printf("     %s⏱  %s%s\n", terminal.Dim, timeAgo, terminal.Reset)

		if i == 0 {
			fmt.Printf("  %s%s───────────────────────%s\n", terminal.BrightCyan, terminal.Dim, terminal.Reset)
		}
		fmt.Println()
	}

	// Tips
	fmt.Printf("  %s💡 Tip: Run %stodo done <id>%s %sto mark your current focus as complete%s\n", terminal.Dim, terminal.BrightCyan, terminal.Reset+terminal.Dim, terminal.Dim, terminal.Reset)
	fmt.Printf("  %s💡 Tip: Run %stodo list%s %sfor interactive navigation%s\n\n", terminal.Dim, terminal.BrightCyan, terminal.Reset+terminal.Dim, terminal.Dim, terminal.Reset)

	return nil
}

func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}
