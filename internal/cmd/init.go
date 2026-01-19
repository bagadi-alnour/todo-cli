package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

var (
	forceInit bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new todo project",
	Long: `Initialize a new todo project in the current directory.

This creates a .todos/ directory containing:
  - todos.json: The todo list storage file
  - config.json: Project-specific configuration

The .todos/ directory can be committed to version control
to share todos with your team.`,
	Example: `  todo init          # Initialize in current directory
  todo init --force  # Reinitialize existing project`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Force reinitialize even if already initialized")
}

func runInit(cmd *cobra.Command, args []string) error {
	terminal.PrintHeader("INITIALIZE PROJECT", "📦")

	projectPath, err := storage.InitProject(".", forceInit)
	if err != nil {
		if _, ok := err.(*types.AlreadyInitializedError); ok {
			terminal.PrintWarning("Project already initialized")
			fmt.Printf("  %sUse --force to reinitialize%s\n\n", terminal.Dim, terminal.Reset)
			return nil
		}
		return fmt.Errorf("failed to initialize project: %w", err)
	}

	terminal.PrintSuccess("Todo project initialized!")
	fmt.Println()
	fmt.Printf("  %sCreated:%s\n", terminal.Dim, terminal.Reset)
	fmt.Printf("    %s.todos/todos.json%s  - Todo storage\n", terminal.BrightCyan, terminal.Reset)
	fmt.Printf("    %s.todos/config.json%s - Configuration\n", terminal.BrightCyan, terminal.Reset)
	fmt.Println()
	fmt.Printf("  %sLocation:%s %s\n", terminal.Dim, terminal.Reset, projectPath)
	fmt.Println()
	fmt.Printf("  %s💡 Next steps:%s\n", terminal.Dim, terminal.Reset)
	fmt.Printf("    %stodo add \"Your first todo\"%s\n", terminal.BrightCyan, terminal.Reset)
	fmt.Printf("    %stodo list%s\n", terminal.BrightCyan, terminal.Reset)
	fmt.Println()

	return nil
}
