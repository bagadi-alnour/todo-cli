package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

var (
	listStatic bool
	listStatus string
	listPath   string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List todos with interactive navigation",
	Long: `List all todos in the project.

By default, opens an interactive view where you can:
  - Navigate with arrow keys or j/k
  - Toggle status with Space or Enter
  - Delete with d or x
  - Press ? for help
  - Press q to quit

Use --static for non-interactive output.`,
	Example: `  todo list                  # Interactive mode
  todo list --static         # Non-interactive output
  todo list --status open    # Filter by status
  todo list --path src/      # Filter by path`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVar(&listStatic, "static", false, "Non-interactive output")
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status: open, done, blocked, waiting, tech-debt")
	listCmd.Flags().StringVarP(&listPath, "path", "p", "", "Filter by path prefix")
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Apply filters
	if listStatus != "" {
		status := types.Status(listStatus)
		if !status.IsValid() {
			return &types.InvalidStatusError{Status: listStatus}
		}
		todos = storage.FilterTodosByStatus(todos, status)
	}

	if listPath != "" {
		todos = storage.FilterTodosByPath(todos, listPath)
	}

	if len(todos) == 0 {
		terminal.PrintInfo("No todos found")
		if listStatus != "" || listPath != "" {
			terminal.PrintDim("Try removing filters or add a new todo with: todo add \"Your task\"")
		} else {
			terminal.PrintDim("Add your first todo with: todo add \"Your task\"")
		}
		fmt.Println()
		return nil
	}

	// Check for interactive mode
	if listStatic || !terminal.IsInteractiveTerminal() {
		return displayStaticList(todos)
	}

	return runInteractiveList(todos, projectRoot)
}

func runInteractiveList(todos []types.Todo, projectRoot string) error {
	selectedIndex := 0
	showDeleteConfirm := false

	// Set terminal to raw mode
	termState, err := terminal.MakeRaw()
	if err != nil {
		return displayStaticList(todos)
	}
	defer termState.Restore()

	// Switch to alternate screen
	terminal.Write(terminal.AltScreenOn + terminal.HideCursor)
	defer terminal.Write(terminal.ShowCursor + terminal.AltScreenOff)

	for {
		if showDeleteConfirm {
			displayDeleteConfirm(todos, selectedIndex)
		} else {
			displayInteractiveTodos(todos, selectedIndex)
		}

		key := terminal.ReadKey()

		if showDeleteConfirm {
			switch key {
			case "y", "Y":
				if selectedIndex >= 0 && selectedIndex < len(todos) {
					todos = storage.DeleteTodo(todos, selectedIndex)
					storage.SaveTodos(projectRoot, todos)
					if selectedIndex >= len(todos) && selectedIndex > 0 {
						selectedIndex--
					}
					if len(todos) == 0 {
						return nil
					}
				}
				showDeleteConfirm = false
			case "n", "N", "ESC", "q":
				showDeleteConfirm = false
			}
			continue
		}

		switch key {
		case "q", "Q", "ESC":
			return nil

		case "DOWN", "j":
			if selectedIndex < len(todos)-1 {
				selectedIndex++
			}

		case "UP", "k":
			if selectedIndex > 0 {
				selectedIndex--
			}

		case "SPACE", "ENTER":
			if selectedIndex >= 0 && selectedIndex < len(todos) {
				todos[selectedIndex].Toggle()
				storage.SaveTodos(projectRoot, todos)
			}

		case "d", "D", "x", "X":
			if selectedIndex >= 0 && selectedIndex < len(todos) {
				showDeleteConfirm = true
			}

		case "g":
			selectedIndex = 0

		case "G":
			selectedIndex = len(todos) - 1

		case "?", "h", "H":
			displayHelp()
			terminal.ReadKey()
		}
	}
}

func displayInteractiveTodos(todos []types.Todo, selectedIndex int) {
	terminal.Write(terminal.CursorHome + terminal.ClearScreen)

	terminal.WriteLine("")
	terminal.WriteLine(fmt.Sprintf("  %s%s╭─────────────────────────────────────────────────────╮%s", terminal.Bold, terminal.BrightCyan, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s%s│  📋  TODO LIST                                       │%s", terminal.Bold, terminal.BrightCyan, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s%s╰─────────────────────────────────────────────────────╯%s", terminal.Bold, terminal.BrightCyan, terminal.Reset))
	terminal.WriteLine("")

	terminal.WriteLine(fmt.Sprintf("  %s↑↓%s navigate  %s␣%s toggle  %sd%s delete  %sq%s quit  %s?%s help",
		terminal.Yellow+terminal.Bold, terminal.Reset+terminal.Dim,
		terminal.Green+terminal.Bold, terminal.Reset+terminal.Dim,
		terminal.Red+terminal.Bold, terminal.Reset+terminal.Dim,
		terminal.BrightRed+terminal.Bold, terminal.Reset+terminal.Dim,
		terminal.Cyan+terminal.Bold, terminal.Reset))
	terminal.WriteLine("")

	for i, todo := range todos {
		isSelected := i == selectedIndex
		var line string

		if isSelected {
			line = fmt.Sprintf("  %s%s▸ ", terminal.Bold, terminal.BrightCyan)
		} else {
			line = fmt.Sprintf("  %s  ", terminal.Dim)
		}

		statusColor := terminal.StatusColor(string(todo.Status))
		checkbox := terminal.StatusIcon(string(todo.Status))

		if isSelected {
			line += fmt.Sprintf("%s%s%s ", statusColor+terminal.Bold, checkbox, terminal.Reset+terminal.Bold+terminal.BrightWhite)
		} else {
			if todo.Status == types.StatusDone {
				line += fmt.Sprintf("%s%s %s", statusColor, checkbox, terminal.Dim)
			} else {
				line += fmt.Sprintf("%s%s %s", statusColor, checkbox, terminal.Reset)
			}
		}

		text := terminal.Truncate(todo.Text, 50)
		line += text + terminal.Reset

		terminal.WriteLine(line)

		if isSelected && len(todo.Context.Paths) > 0 {
			terminal.WriteLine(fmt.Sprintf("      %s📁 %s%s", terminal.Dim, strings.Join(todo.Context.Paths, ", "), terminal.Reset))
		}
		if isSelected && todo.Context.Branch != "" {
			terminal.WriteLine(fmt.Sprintf("      %s🌿 %s%s", terminal.Dim, todo.Context.Branch, terminal.Reset))
		}
	}

	terminal.WriteLine("")

	progress := float64(selectedIndex+1) / float64(len(todos))
	barWidth := 30
	filled := int(progress * float64(barWidth))

	progressBar := "  " + terminal.Dim
	for i := 0; i < barWidth; i++ {
		if i < filled {
			progressBar += "█"
		} else {
			progressBar += "░"
		}
	}
	progressBar += fmt.Sprintf(" %d/%d%s", selectedIndex+1, len(todos), terminal.Reset)
	terminal.WriteLine(progressBar)

	// Stats
	stats := countByStatus(todos)
	terminal.WriteLine(fmt.Sprintf("  %s%s●%s %d open  %s●%s %d done%s",
		terminal.Dim, terminal.Blue, terminal.Dim, stats["open"], terminal.Green, terminal.Dim, stats["done"], terminal.Reset))
}

func displayDeleteConfirm(todos []types.Todo, selectedIndex int) {
	terminal.Write(terminal.CursorHome + terminal.ClearScreen)

	terminal.WriteLine("")
	terminal.WriteLine(fmt.Sprintf("  %s%s╭─────────────────────────────────────────────────────╮%s", terminal.Bold, terminal.BrightRed, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s%s│  🗑️   DELETE TODO                                    │%s", terminal.Bold, terminal.BrightRed, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s%s╰─────────────────────────────────────────────────────╯%s", terminal.Bold, terminal.BrightRed, terminal.Reset))
	terminal.WriteLine("")

	if selectedIndex >= 0 && selectedIndex < len(todos) {
		todo := todos[selectedIndex]
		text := terminal.Truncate(todo.Text, 45)
		terminal.WriteLine(fmt.Sprintf("  %sAre you sure you want to delete:%s", terminal.Dim, terminal.Reset))
		terminal.WriteLine("")
		terminal.WriteLine(fmt.Sprintf("  %s%s\"%s\"%s", terminal.Bold, terminal.BrightWhite, text, terminal.Reset))
		terminal.WriteLine("")
	}

	terminal.WriteLine(fmt.Sprintf("  %sThis action cannot be undone.%s", terminal.Red, terminal.Reset))
	terminal.WriteLine("")
	terminal.WriteLine(fmt.Sprintf("  Press %s%sY%s to confirm, %s%sN%s to cancel", terminal.Green+terminal.Bold, "", terminal.Reset, terminal.Red+terminal.Bold, "", terminal.Reset))
}

func displayHelp() {
	terminal.Write(terminal.CursorHome + terminal.ClearScreen)

	terminal.WriteLine("")
	terminal.WriteLine(fmt.Sprintf("  %s%s╭─────────────────────────────────────────────────────╮%s", terminal.Bold, terminal.BrightCyan, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s%s│  📚  KEYBOARD SHORTCUTS                              │%s", terminal.Bold, terminal.BrightCyan, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s%s╰─────────────────────────────────────────────────────╯%s", terminal.Bold, terminal.BrightCyan, terminal.Reset))
	terminal.WriteLine("")

	terminal.WriteLine(fmt.Sprintf("  %sNavigation%s", terminal.Bold+terminal.Yellow, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s↑%s %sk%s    Move up", terminal.Yellow+terminal.Bold, terminal.Reset, terminal.Dim, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s↓%s %sj%s    Move down", terminal.Yellow+terminal.Bold, terminal.Reset, terminal.Dim, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %sg%s      Jump to top", terminal.Yellow+terminal.Bold, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %sG%s      Jump to bottom", terminal.Yellow+terminal.Bold, terminal.Reset))
	terminal.WriteLine("")

	terminal.WriteLine(fmt.Sprintf("  %sActions%s", terminal.Bold+terminal.Green, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s␣%s      Toggle todo status", terminal.Green+terminal.Bold, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %sEnter%s  Toggle todo status", terminal.Green+terminal.Bold, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %sd%s/%sx%s   Delete selected todo", terminal.Red+terminal.Bold, terminal.Reset, terminal.Red+terminal.Bold, terminal.Reset))
	terminal.WriteLine("")

	terminal.WriteLine(fmt.Sprintf("  %sOther%s", terminal.Bold+terminal.Cyan, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %sq%s      Quit", terminal.Red+terminal.Bold, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s?%s      Show this help", terminal.Cyan+terminal.Bold, terminal.Reset))
	terminal.WriteLine("")

	terminal.WriteLine(fmt.Sprintf("  %sStatus Icons%s", terminal.Bold+terminal.Magenta, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s✓%s  Done     %s○%s  Open", terminal.Green, terminal.Reset, terminal.Blue, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s✗%s  Blocked  %s◔%s  Waiting", terminal.Red, terminal.Reset, terminal.Yellow, terminal.Reset))
	terminal.WriteLine(fmt.Sprintf("  %s⚠%s  Tech Debt", terminal.Magenta, terminal.Reset))
	terminal.WriteLine("")

	terminal.WriteLine(fmt.Sprintf("  %sPress any key to continue...%s", terminal.Dim, terminal.Reset))
}

func displayStaticList(todos []types.Todo) error {
	fmt.Printf("\n  %s%s📋 TODO LIST%s\n", terminal.Bold, terminal.BrightCyan, terminal.Reset)
	fmt.Printf("  %s─────────────────────────────────────────%s\n\n", terminal.Dim, terminal.Reset)

	for i, todo := range todos {
		statusColor := terminal.StatusColor(string(todo.Status))
		checkbox := terminal.StatusIcon(string(todo.Status))

		textStyle := ""
		if todo.Status == types.StatusDone {
			textStyle = terminal.Dim
		}

		fmt.Printf("  %s%d.%s %s%s%s %s%s%s\n",
			terminal.Dim, i+1, terminal.Reset,
			statusColor, checkbox, terminal.Reset,
			textStyle, todo.Text, terminal.Reset)

		if len(todo.Context.Paths) > 0 {
			fmt.Printf("     %s📁 %s%s\n", terminal.Dim, strings.Join(todo.Context.Paths, ", "), terminal.Reset)
		}
		if todo.Context.Branch != "" {
			fmt.Printf("     %s🌿 %s%s\n", terminal.Dim, todo.Context.Branch, terminal.Reset)
		}
	}

	stats := countByStatus(todos)
	fmt.Println()
	fmt.Printf("  %s%s●%s %d open  %s●%s %d done%s\n",
		terminal.Dim, terminal.Blue, terminal.Dim, stats["open"], terminal.Green, terminal.Dim, stats["done"], terminal.Reset)
	fmt.Printf("\n  %s💡 Run 'todo list' in a terminal for interactive mode%s\n", terminal.Dim, terminal.Reset)
	fmt.Printf("  %s💡 Run 'todo ui' for web interface%s\n\n", terminal.Dim, terminal.Reset)

	return nil
}

func countByStatus(todos []types.Todo) map[string]int {
	counts := map[string]int{
		"open":      0,
		"done":      0,
		"blocked":   0,
		"waiting":   0,
		"tech-debt": 0,
	}
	for _, t := range todos {
		counts[string(t.Status)]++
	}
	return counts
}
