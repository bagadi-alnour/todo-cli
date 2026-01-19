package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bagadi-alnour/todo-cli/internal/types"
)

const (
	TodosDir     = ".todos"
	TodosFile    = "todos.json"
	ConfigFile   = "config.json"
)

// GenerateID creates a unique ID for a new todo
func GenerateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

// FindProjectRoot walks up the directory tree to find a .todos directory
func FindProjectRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	current := absPath
	for {
		todosPath := filepath.Join(current, TodosDir)
		if info, err := os.Stat(todosPath); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			return "", &types.ProjectNotFoundError{SearchPath: startPath}
		}
		current = parent
	}
}

// EnsureProjectRoot ensures a .todos directory exists, creating it if necessary
func EnsureProjectRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	todosPath := filepath.Join(absPath, TodosDir)
	if err := os.MkdirAll(todosPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create .todos directory: %w", err)
	}

	return absPath, nil
}

// InitProject initializes a new todo project in the given directory
func InitProject(path string, force bool) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	todosPath := filepath.Join(absPath, TodosDir)

	// Check if already initialized
	if !force {
		if _, err := os.Stat(todosPath); err == nil {
			return "", &types.AlreadyInitializedError{Path: todosPath}
		}
	}

	// Create .todos directory
	if err := os.MkdirAll(todosPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create .todos directory: %w", err)
	}

	// Create empty todos.json
	todoFile := types.NewTodoFile()
	if err := saveTodoFile(absPath, todoFile); err != nil {
		return "", err
	}

	// Create default config.json
	config := types.DefaultConfig()
	if err := SaveConfig(absPath, config); err != nil {
		return "", err
	}

	return absPath, nil
}

// GetTodosPath returns the full path to the todos.json file
func GetTodosPath(projectRoot string) string {
	return filepath.Join(projectRoot, TodosDir, TodosFile)
}

// GetConfigPath returns the full path to the config.json file
func GetConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, TodosDir, ConfigFile)
}

// LoadTodos loads todos from the project's todos.json file
func LoadTodos(projectRoot string) ([]types.Todo, error) {
	todosPath := GetTodosPath(projectRoot)

	data, err := os.ReadFile(todosPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Todo{}, nil
		}
		return nil, fmt.Errorf("failed to read todos file: %w", err)
	}

	var todoFile types.TodoFile
	if err := json.Unmarshal(data, &todoFile); err != nil {
		// Try legacy format (just array of todos)
		var todos []types.Todo
		if err := json.Unmarshal(data, &todos); err != nil {
			return nil, fmt.Errorf("failed to parse todos file: %w", err)
		}
		return todos, nil
	}

	return todoFile.Todos, nil
}

// SaveTodos saves todos to the project's todos.json file
func SaveTodos(projectRoot string, todos []types.Todo) error {
	todoFile := &types.TodoFile{
		Version: 1,
		Todos:   todos,
	}
	return saveTodoFile(projectRoot, todoFile)
}

// saveTodoFile saves the todo file structure to disk
func saveTodoFile(projectRoot string, todoFile *types.TodoFile) error {
	todosPath := GetTodosPath(projectRoot)

	data, err := json.MarshalIndent(todoFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal todos: %w", err)
	}

	if err := os.WriteFile(todosPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write todos file: %w", err)
	}

	return nil
}

// LoadConfig loads the project configuration
func LoadConfig(projectRoot string) (*types.Config, error) {
	configPath := GetConfigPath(projectRoot)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return types.DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the project configuration
func SaveConfig(projectRoot string, config *types.Config) error {
	configPath := GetConfigPath(projectRoot)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// FindTodoByID finds a todo by its ID
func FindTodoByID(todos []types.Todo, id string) (*types.Todo, int) {
	for i := range todos {
		if todos[i].ID == id {
			return &todos[i], i
		}
	}
	return nil, -1
}

// FindTodoByIndex finds a todo by its 1-based index
func FindTodoByIndex(todos []types.Todo, index int) (*types.Todo, int) {
	idx := index - 1 // Convert to 0-based
	if idx >= 0 && idx < len(todos) {
		return &todos[idx], idx
	}
	return nil, -1
}

// FindTodoByIDOrIndex finds a todo by ID or 1-based index
func FindTodoByIDOrIndex(todos []types.Todo, idOrIndex string) (*types.Todo, int) {
	// First try as index
	var index int
	if _, err := fmt.Sscanf(idOrIndex, "%d", &index); err == nil {
		if todo, idx := FindTodoByIndex(todos, index); todo != nil {
			return todo, idx
		}
	}

	// Then try as ID (partial match)
	for i := range todos {
		if todos[i].ID == idOrIndex || (len(idOrIndex) >= 4 && len(todos[i].ID) >= len(idOrIndex) && todos[i].ID[:len(idOrIndex)] == idOrIndex) {
			return &todos[i], i
		}
	}

	return nil, -1
}

// DeleteTodo removes a todo by index and returns the updated slice
func DeleteTodo(todos []types.Todo, index int) []types.Todo {
	if index < 0 || index >= len(todos) {
		return todos
	}
	return append(todos[:index], todos[index+1:]...)
}

// FilterTodosByStatus filters todos by status
func FilterTodosByStatus(todos []types.Todo, status types.Status) []types.Todo {
	var filtered []types.Todo
	for _, t := range todos {
		if t.Status == status {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// FilterTodosByPath filters todos that have paths matching the given prefix
func FilterTodosByPath(todos []types.Todo, pathPrefix string) []types.Todo {
	var filtered []types.Todo
	for _, t := range todos {
		for _, p := range t.Context.Paths {
			if len(p) >= len(pathPrefix) && p[:len(pathPrefix)] == pathPrefix {
				filtered = append(filtered, t)
				break
			}
		}
	}
	return filtered
}

// FilterTodosByBranch filters todos by git branch
func FilterTodosByBranch(todos []types.Todo, branch string) []types.Todo {
	var filtered []types.Todo
	for _, t := range todos {
		if t.Context.Branch == branch {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
