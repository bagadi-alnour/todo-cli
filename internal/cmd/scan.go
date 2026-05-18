package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	scanDryRun bool
	scanJSON   bool
	scanTag    string
)

var scanCmd = &cobra.Command{
	Use:   "scan [path...]",
	Short: "Import TODO/FIXME comments from source files",
	Long: `Scan source files for TODO and FIXME comments and create todos from them.

Skips binary files, .git directories, node_modules, and vendor folders.
Each comment becomes a todo with the file path attached as context.
Duplicate text+path combinations are skipped.`,
	Example: `  todo scan                         # Scan current directory
  todo scan src/                    # Scan specific directory
  todo scan --dry-run               # Preview without importing
  todo scan --tag code-review       # Tag all imported todos
  todo scan --json                  # Output found comments as JSON`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().BoolVar(&scanDryRun, "dry-run", false, "Preview found comments without importing")
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "Output as JSON")
	scanCmd.Flags().StringVarP(&scanTag, "tag", "t", "", "Tag to apply to all imported todos")
}

var todoCommentRe = regexp.MustCompile(`(?i)(?://|#|/\*|\*)\s*(?:TODO|FIXME|HACK|XXX)[\s:]+(.+?)(?:\s*\*/)?$`)

type scanResult struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Text string `json:"text"`
	Kind string `json:"kind"`
}

func runScan(cmd *cobra.Command, args []string) error {
	projectRoot, err := storage.FindProjectRoot(".")
	if err != nil {
		return err
	}

	roots := args
	if len(roots) == 0 {
		roots = []string{"."}
	}

	var results []scanResult
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			base := info.Name()
			if info.IsDir() {
				switch base {
				case ".git", "node_modules", "vendor", ".gomodcache", "__pycache__", ".todos":
					return filepath.SkipDir
				}
				return nil
			}
			if !isSourceFile(base) {
				return nil
			}

			relPath, _ := filepath.Rel(projectRoot, path)
			if relPath == "" {
				relPath = path
			}

			found, err := scanFile(path, relPath)
			if err != nil {
				return nil
			}
			results = append(results, found...)
			return nil
		})
	}

	if scanJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{"found": results, "count": len(results)})
	}

	if len(results) == 0 {
		terminal.PrintInfo("No TODO/FIXME comments found")
		fmt.Println()
		return nil
	}

	if scanDryRun {
		terminal.PrintHeader("SCAN PREVIEW (dry run)", "🔍")
		for _, r := range results {
			fmt.Printf("  %s%s:%d%s %s\n", terminal.Dim, r.File, r.Line, terminal.Reset, r.Text)
		}
		fmt.Printf("\n  %s%d comment(s) found. Run without --dry-run to import.%s\n\n", terminal.Dim, len(results), terminal.Reset)
		return nil
	}

	return storage.WithLock(projectRoot, func() error {
		todos, err := storage.LoadTodos(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to load todos: %w", err)
		}

		existing := make(map[string]struct{})
		for _, t := range todos {
			for _, p := range t.Context.Paths {
				existing[t.Text+"\x00"+p] = struct{}{}
			}
		}

		added := 0
		skipped := 0
		for _, r := range results {
			key := r.Text + "\x00" + r.File
			if _, dup := existing[key]; dup {
				skipped++
				continue
			}
			existing[key] = struct{}{}

			id, err := storage.GenerateID()
			if err != nil {
				continue
			}
			todo := types.NewTodo(id, r.Text)
			if err := storage.ApplyCreator(todo); err != nil {
				return err
			}
			todo.SetPaths([]string{r.File})
			todo.Meta.Source = "scan"
			if scanTag != "" {
				todo.Tags = []string{strings.ToLower(strings.TrimSpace(scanTag))}
			}
			todos = append(todos, *todo)
			added++
		}

		if added > 0 {
			if err := storage.SaveTodos(projectRoot, todos); err != nil {
				return fmt.Errorf("failed to save todos: %w", err)
			}
		}

		terminal.PrintSuccess(fmt.Sprintf("Imported %d todo(s) from source comments", added))
		if skipped > 0 {
			fmt.Printf("  %s%d duplicate(s) skipped%s\n", terminal.Dim, skipped, terminal.Reset)
		}
		fmt.Println()
		return nil
	})
}

func scanFile(absPath, relPath string) ([]scanResult, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []scanResult
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		matches := todoCommentRe.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}
		text := strings.TrimSpace(matches[1])
		if text == "" {
			continue
		}
		kind := "TODO"
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "fixme"):
			kind = "FIXME"
		case strings.Contains(lower, "hack"):
			kind = "HACK"
		case strings.Contains(lower, "xxx"):
			kind = "XXX"
		}
		results = append(results, scanResult{File: relPath, Line: lineNum, Text: text, Kind: kind})
	}
	return results, nil
}

func isSourceFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rb", ".rs", ".java",
		".c", ".cpp", ".h", ".hpp", ".cs", ".swift", ".kt", ".scala",
		".sh", ".bash", ".zsh", ".fish", ".yaml", ".yml", ".toml",
		".php", ".lua", ".r", ".jl", ".ex", ".exs", ".erl", ".hs",
		".ml", ".mli", ".clj", ".cljs", ".vue", ".svelte", ".css",
		".scss", ".less", ".html", ".xml", ".sql", ".tf", ".proto",
		".graphql", ".gql", ".md", ".txt", ".cfg", ".ini", ".conf":
		return true
	}
	return false
}
