package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

var (
	doctorFix bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check for issues with your todo list",
	Long: `Run health checks on your todo list.

Checks for:
  - Orphaned paths (todos pointing to non-existent files)
  - Empty todos
  - Duplicate todos
  - Stale todos (open for more than 30 days)`,
	Example: `  todo doctor        # Run all checks
  todo doctor --fix  # Auto-fix issues (remove orphans)`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)

	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Auto-fix issues where possible")
}

func runDoctor(cmd *cobra.Command, args []string) error {
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

	terminal.PrintHeader("TODO DOCTOR", "🩺")

	// Project info
	projectName := filepath.Base(projectRoot)
	fmt.Printf("  %s📁 Project:%s %s%s%s\n", terminal.Dim, terminal.Reset, terminal.BrightCyan, projectName, terminal.Reset)
	fmt.Printf("  %s📋 Todos:%s   %s%d total%s\n", terminal.Dim, terminal.Reset, terminal.BrightWhite+terminal.Bold, len(todos), terminal.Reset)

	// Git info
	if git.IsGitRepo() {
		branch, _ := git.GetCurrentBranch()
		fmt.Printf("  %s🌿 Branch:%s  %s%s%s\n", terminal.Dim, terminal.Reset, terminal.Green, branch, terminal.Reset)
	}
	fmt.Println()

	if len(todos) == 0 {
		terminal.PrintSuccess("No todos to check.")
		fmt.Println()
		return nil
	}

	fmt.Printf("  %s%s─── HEALTH CHECKS ───%s\n\n", terminal.BrightCyan, terminal.Dim, terminal.Reset)

	issues := 0
	modified := false

	// Check 1: Orphaned paths
	fmt.Printf("  %s🔍 Checking for orphaned paths...%s\n", terminal.Dim, terminal.Reset)
	orphanedTodos, orphanedPaths, totalPaths := checkOrphanedPaths(todos, projectRoot)
	if len(orphanedTodos) > 0 {
		fmt.Printf("     %s⚠  %d orphaned path(s) found in %d todo(s)%s\n", terminal.BrightYellow+terminal.Bold, orphanedPaths, len(orphanedTodos), terminal.Reset)
		issues += len(orphanedTodos)
	} else if totalPaths > 0 {
		fmt.Printf("     %s✓  All %d path(s) are valid%s\n", terminal.Green, totalPaths, terminal.Reset)
	} else {
		fmt.Printf("     %s○  No paths to check%s\n", terminal.Dim, terminal.Reset)
	}

	// Check 2: Empty todos
	fmt.Printf("  %s🔍 Checking for empty todos...%s\n", terminal.Dim, terminal.Reset)
	emptyTodos := checkEmptyTodos(todos)
	if len(emptyTodos) > 0 {
		fmt.Printf("     %s⚠  %d empty todo(s) found%s\n", terminal.BrightYellow+terminal.Bold, len(emptyTodos), terminal.Reset)
		issues += len(emptyTodos)
	} else {
		fmt.Printf("     %s✓  No empty todos%s\n", terminal.Green, terminal.Reset)
	}

	// Check 3: Duplicate todos
	fmt.Printf("  %s🔍 Checking for duplicate todos...%s\n", terminal.Dim, terminal.Reset)
	duplicates := checkDuplicateTodos(todos)
	if len(duplicates) > 0 {
		fmt.Printf("     %s⚠  %d potential duplicate(s) found%s\n", terminal.BrightYellow+terminal.Bold, len(duplicates), terminal.Reset)
		issues += len(duplicates)
	} else {
		fmt.Printf("     %s✓  No duplicates detected%s\n", terminal.Green, terminal.Reset)
	}

	// Check 4: Stale todos
	fmt.Printf("  %s🔍 Checking for stale todos...%s\n", terminal.Dim, terminal.Reset)
	staleTodos := checkStaleTodos(todos)
	if len(staleTodos) > 0 {
		fmt.Printf("     %s⚠  %d stale todo(s) (open > 30 days)%s\n", terminal.BrightYellow+terminal.Bold, len(staleTodos), terminal.Reset)
		issues += len(staleTodos)
	} else {
		fmt.Printf("     %s✓  No stale todos%s\n", terminal.Green, terminal.Reset)
	}

	fmt.Println()

	// Summary
	fmt.Printf("  %s%s─── SUMMARY ───%s\n\n", terminal.BrightCyan, terminal.Dim, terminal.Reset)

	// Stats table
	stats := countTodoStats(todos)
	fmt.Printf("  %s┌──────────────────────────────────────┐%s\n", terminal.Dim, terminal.Reset)
	fmt.Printf("  %s│%s  %-12s %s%3d%s  %s│%s  %-12s %s%3d%s  %s│%s\n",
		terminal.Dim, terminal.Reset, "Open", terminal.Blue+terminal.Bold, stats["open"], terminal.Reset,
		terminal.Dim, terminal.Reset, "Done", terminal.Green+terminal.Bold, stats["done"], terminal.Reset, terminal.Dim, terminal.Reset)
	fmt.Printf("  %s│%s  %-12s %s%3d%s  %s│%s  %-12s %s%3d%s  %s│%s\n",
		terminal.Dim, terminal.Reset, "Blocked", terminal.Red+terminal.Bold, stats["blocked"], terminal.Reset,
		terminal.Dim, terminal.Reset, "Waiting", terminal.Magenta+terminal.Bold, stats["waiting"], terminal.Reset, terminal.Dim, terminal.Reset)
	fmt.Printf("  %s│%s  %-12s %s%3d%s  %s│%s  %-12s %s%3d%s  %s│%s\n",
		terminal.Dim, terminal.Reset, "Tech Debt", terminal.Yellow+terminal.Bold, stats["tech-debt"], terminal.Reset,
		terminal.Dim, terminal.Reset, "Total", terminal.BrightWhite+terminal.Bold, len(todos), terminal.Reset, terminal.Dim, terminal.Reset)
	fmt.Printf("  %s└──────────────────────────────────────┘%s\n", terminal.Dim, terminal.Reset)
	fmt.Println()

	// Health status
	if issues == 0 {
		terminal.PrintSuccess("Your todo list is healthy!")
		fmt.Println()
	} else {
		fmt.Printf("  %s%s⚠  Found %d issue(s) to review%s\n\n", terminal.BrightYellow, terminal.Bold, issues, terminal.Reset)

		// Show detailed issues
		if len(orphanedTodos) > 0 {
			fmt.Printf("  %s%sOrphaned Paths:%s\n", terminal.Yellow, terminal.Bold, terminal.Reset)
			for _, todo := range orphanedTodos {
				fmt.Printf("  %s  •%s %s\n", terminal.Dim, terminal.Reset, terminal.Truncate(todo.Text, 50))
				for _, path := range todo.Context.Paths {
					absPath := filepath.Join(projectRoot, path)
					if _, err := os.Stat(absPath); os.IsNotExist(err) {
						fmt.Printf("      %s❌ %s%s\n", terminal.Red, path, terminal.Reset)
					}
				}
			}
			fmt.Println()
		}

		if len(staleTodos) > 0 {
			fmt.Printf("  %s%sStale Todos (consider updating or completing):%s\n", terminal.Yellow, terminal.Bold, terminal.Reset)
			for _, todo := range staleTodos {
				age := formatTimeAgo(todo.CreatedAt)
				fmt.Printf("  %s  •%s %s %s(%s)%s\n", terminal.Dim, terminal.Reset, terminal.Truncate(todo.Text, 40), terminal.Dim, age, terminal.Reset)
			}
			fmt.Println()
		}
	}

	// Save if modified
	if modified {
		if err := storage.SaveTodos(projectRoot, todos); err != nil {
			return fmt.Errorf("failed to save todos: %w", err)
		}
		terminal.PrintSuccess("Changes saved!")
		fmt.Println()
	}

	// Tips
	fmt.Printf("  %s💡 Tips:%s\n", terminal.Dim, terminal.Reset)
	fmt.Printf("  %s   • Use %stodo list%s %sto manage your todos interactively%s\n", terminal.Dim, terminal.BrightCyan, terminal.Reset, terminal.Dim, terminal.Reset)
	fmt.Printf("  %s   • Use %stodo ui%s %sfor a web-based interface%s\n", terminal.Dim, terminal.BrightCyan, terminal.Reset, terminal.Dim, terminal.Reset)
	fmt.Printf("  %s   • Use %stodo focus%s %sto see your current priorities%s\n\n", terminal.Dim, terminal.BrightCyan, terminal.Reset, terminal.Dim, terminal.Reset)

	return nil
}

func checkOrphanedPaths(todos []types.Todo, projectRoot string) ([]types.Todo, int, int) {
	var orphaned []types.Todo
	orphanedCount := 0
	totalPaths := 0

	for _, todo := range todos {
		if len(todo.Context.Paths) == 0 {
			continue
		}

		hasOrphan := false
		for _, path := range todo.Context.Paths {
			totalPaths++
			absPath := filepath.Join(projectRoot, path)
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				orphanedCount++
				hasOrphan = true
			}
		}
		if hasOrphan {
			orphaned = append(orphaned, todo)
		}
	}

	return orphaned, orphanedCount, totalPaths
}

func checkEmptyTodos(todos []types.Todo) []types.Todo {
	var empty []types.Todo
	for _, todo := range todos {
		if len(todo.Text) == 0 {
			empty = append(empty, todo)
		}
	}
	return empty
}

func checkDuplicateTodos(todos []types.Todo) []types.Todo {
	seen := make(map[string]bool)
	var duplicates []types.Todo

	for _, todo := range todos {
		if seen[todo.Text] {
			duplicates = append(duplicates, todo)
		}
		seen[todo.Text] = true
	}

	return duplicates
}

func checkStaleTodos(todos []types.Todo) []types.Todo {
	var stale []types.Todo
	now := time.Now()

	for _, todo := range todos {
		if todo.Status != types.StatusOpen {
			continue
		}
		age := now.Sub(todo.CreatedAt)
		if age.Hours() > 30*24 { // 30 days
			stale = append(stale, todo)
		}
	}

	return stale
}

func countTodoStats(todos []types.Todo) map[string]int {
	stats := map[string]int{
		"open":      0,
		"done":      0,
		"blocked":   0,
		"waiting":   0,
		"tech-debt": 0,
	}

	for _, todo := range todos {
		stats[string(todo.Status)]++
	}

	return stats
}
