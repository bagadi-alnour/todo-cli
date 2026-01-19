package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/ui"
)

var (
	uiPort int
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the web UI server",
	Long: `Start a local web server for managing todos with a graphical interface.

The web UI provides:
  - A modern dark-themed interface
  - Add, edit, and delete todos
  - Filter by status
  - Keyboard navigation`,
	Example: `  todo ui            # Start on default port 8080
  todo ui --port 3000 # Start on custom port`,
	RunE: runUI,
}

func init() {
	rootCmd.AddCommand(uiCmd)

	uiCmd.Flags().IntVarP(&uiPort, "port", "p", 8080, "Port to run the server on")
}

func runUI(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	// Create server
	server := ui.NewServer(projectRoot, uiPort)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", uiPort),
		Handler: server.Handler(),
	}

	// Start server in goroutine
	go func() {
		terminal.PrintHeader("TODO UI SERVER", "🚀")
		fmt.Printf("  %s●%s Running at %s%shttp://localhost:%d%s\n",
			terminal.Green, terminal.Reset,
			terminal.Bold+terminal.Underline, terminal.BrightCyan, uiPort, terminal.Reset)
		fmt.Printf("  %s●%s Press %sCtrl+C%s to stop\n\n",
			terminal.Yellow, terminal.Reset,
			terminal.Bold, terminal.Reset)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("%sServer error: %v%s\n", terminal.Red, err, terminal.Reset)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Printf("\n%sShutting down server...%s\n", terminal.Yellow, terminal.Reset)
	return httpServer.Close()
}
