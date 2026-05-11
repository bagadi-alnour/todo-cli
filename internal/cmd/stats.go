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

var statsJSON bool

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show todo statistics and summary dashboard",
	Long: `Display a summary of your todo list including counts by status,
priority, tag breakdown, completion rate, and average age of open items.`,
	Example: `  todo stats         # Show dashboard
  todo stats --json  # Machine-readable output`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output as JSON")
}

type statsReport struct {
	Total              int            `json:"total"`
	ByStatus           map[string]int `json:"byStatus"`
	ByPriority         map[string]int `json:"byPriority"`
	ByTag              map[string]int `json:"byTag"`
	CompletionRate     float64        `json:"completionRate"`
	AvgAgeDays         float64        `json:"avgAgeDaysOpen"`
	AvgCompletionHours float64        `json:"avgCompletionHours"`
	Overdue            int            `json:"overdue"`
}

func computeStats(todos []types.Todo, now time.Time) statsReport {
	r := statsReport{
		Total:      len(todos),
		ByStatus:   countByStatus(todos),
		ByPriority: map[string]int{"high": 0, "medium": 0, "low": 0},
		ByTag:      map[string]int{},
	}

	var openAgeSum float64
	openCount := 0
	var completionSum float64
	doneCount := 0
	for _, t := range todos {
		r.ByPriority[string(t.Priority)]++
		for _, tag := range t.Tags {
			r.ByTag[strings.ToLower(tag)]++
		}
		if t.Status == types.StatusOpen {
			openCount++
			openAgeSum += now.Sub(t.CreatedAt).Hours() / 24.0
		}
		if t.Status == types.StatusDone && t.CompletedAt != nil {
			doneCount++
			completionSum += t.CompletedAt.Sub(t.CreatedAt).Hours()
		}
		if t.Status == types.StatusOpen && t.DueAt != nil && t.DueAt.Before(now) {
			r.Overdue++
		}
	}

	if r.Total > 0 {
		r.CompletionRate = float64(r.ByStatus["done"]) / float64(r.Total) * 100
	}
	if openCount > 0 {
		r.AvgAgeDays = openAgeSum / float64(openCount)
	}
	if doneCount > 0 {
		r.AvgCompletionHours = completionSum / float64(doneCount)
	}

	return r
}

func runStats(cmd *cobra.Command, args []string) error {
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

	now := time.Now()
	report := computeStats(todos, now)

	if statsJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	terminal.PrintHeader("TODO STATS", "📊")

	if report.Total == 0 {
		terminal.PrintInfo("No todos yet. Add one with: todo add \"Your task\"")
		fmt.Println()
		return nil
	}

	// Status breakdown
	fmt.Printf("  %sStatus%s\n", terminal.Bold+terminal.BrightCyan, terminal.Reset)
	fmt.Printf("    %s●%s Open       %s%d%s\n", terminal.Blue, terminal.Reset, terminal.Bold, report.ByStatus["open"], terminal.Reset)
	fmt.Printf("    %s●%s Done       %s%d%s\n", terminal.Green, terminal.Reset, terminal.Bold, report.ByStatus["done"], terminal.Reset)
	fmt.Printf("    %s●%s Blocked    %s%d%s\n", terminal.Red, terminal.Reset, terminal.Bold, report.ByStatus["blocked"], terminal.Reset)
	fmt.Printf("    %s●%s Waiting    %s%d%s\n", terminal.Magenta, terminal.Reset, terminal.Bold, report.ByStatus["waiting"], terminal.Reset)
	fmt.Printf("    %s●%s Tech Debt  %s%d%s\n", terminal.Yellow, terminal.Reset, terminal.Bold, report.ByStatus["tech-debt"], terminal.Reset)
	fmt.Println()

	// Priority breakdown
	fmt.Printf("  %sPriority%s\n", terminal.Bold+terminal.BrightCyan, terminal.Reset)
	fmt.Printf("    %s▲%s High    %s%d%s\n", terminal.BrightRed, terminal.Reset, terminal.Bold, report.ByPriority["high"], terminal.Reset)
	fmt.Printf("    %s-%s Medium  %s%d%s\n", terminal.Yellow, terminal.Reset, terminal.Bold, report.ByPriority["medium"], terminal.Reset)
	fmt.Printf("    %s▼%s Low     %s%d%s\n", terminal.Dim, terminal.Reset, terminal.Bold, report.ByPriority["low"], terminal.Reset)
	fmt.Println()

	// Tags
	if len(report.ByTag) > 0 {
		fmt.Printf("  %sTags%s\n", terminal.Bold+terminal.BrightCyan, terminal.Reset)
		for tag, count := range report.ByTag {
			fmt.Printf("    %s#%s%s %d\n", terminal.Cyan, tag, terminal.Reset, count)
		}
		fmt.Println()
	}

	// Metrics
	fmt.Printf("  %sMetrics%s\n", terminal.Bold+terminal.BrightCyan, terminal.Reset)
	fmt.Printf("    Completion rate:   %s%.0f%%%s\n", terminal.Bold, report.CompletionRate, terminal.Reset)
	fmt.Printf("    Avg open age:      %s%.1f days%s\n", terminal.Bold, report.AvgAgeDays, terminal.Reset)
	if report.AvgCompletionHours > 0 {
		if report.AvgCompletionHours >= 24 {
			fmt.Printf("    Avg time to done:  %s%.1f days%s\n", terminal.Bold, report.AvgCompletionHours/24, terminal.Reset)
		} else {
			fmt.Printf("    Avg time to done:  %s%.1f hours%s\n", terminal.Bold, report.AvgCompletionHours, terminal.Reset)
		}
	}
	if report.Overdue > 0 {
		fmt.Printf("    Overdue:           %s%s%d%s\n", terminal.BrightRed, terminal.Bold, report.Overdue, terminal.Reset)
	} else {
		fmt.Printf("    Overdue:           %s0%s\n", terminal.Bold, terminal.Reset)
	}
	fmt.Printf("    Total:             %s%d%s\n", terminal.Bold, report.Total, terminal.Reset)
	fmt.Println()

	return nil
}
