package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/storage"
	"github.com/spf13/cobra"
)

// registerPathFlagCompletion wires up project-aware file/folder completion for a flag.
func registerPathFlagCompletion(command *cobra.Command, flagName string) {
	_ = command.RegisterFlagCompletionFunc(flagName, completePath)
}

// completePath suggests files and directories relative to the todo project root (when found)
// so completions stay consistent even when running commands from subdirectories.
func completePath(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	baseDir := findProjectRootOrWD()
	input := expandHome(toComplete)

	dirPart, prefix := splitPathInput(input)
	searchDir := baseDir
	if filepath.IsAbs(input) {
		searchDir = dirPart
		if searchDir == "" {
			searchDir = string(os.PathSeparator)
		}
	} else if dirPart != "" {
		searchDir = filepath.Join(baseDir, dirPart)
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		// Fall back to shell defaults if we can't read the directory
		return nil, cobra.ShellCompDirectiveDefault
	}

	completions := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}

		candidate := name
		if dirPart != "" {
			candidate = filepath.Join(dirPart, name)
		}

		if entry.IsDir() {
			candidate += string(os.PathSeparator)
		}

		completions = append(completions, candidate)
	}

	sort.Strings(completions)
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func findProjectRootOrWD() string {
	if root, err := storage.FindProjectRoot("."); err == nil {
		return root
	}

	if wd, err := os.Getwd(); err == nil {
		return wd
	}

	return "."
}

func expandHome(input string) string {
	if strings.HasPrefix(input, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(input, "~"))
		}
	}
	return input
}

func splitPathInput(input string) (dir, prefix string) {
	if input == "" {
		return "", ""
	}

	// Accept either OS-specific separators or forward slashes (useful on Windows).
	idx := strings.LastIndexAny(input, "/\\")
	if idx == -1 {
		return "", input
	}

	return input[:idx], input[idx+1:]
}
