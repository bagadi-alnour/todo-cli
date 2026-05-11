package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/types"
	"github.com/gofrs/flock"
)

const (
	TodosDir    = ".todos"
	TodosFile   = "todos.json"
	ConfigFile  = "config.json"
	ArchiveFile = "archive.json"
	LockFile    = ".lock"
)

// WithLock acquires an exclusive file lock on .todos/.lock, runs fn, then
// releases the lock. This prevents concurrent CLI invocations from
// corrupting the data files.
func WithLock(projectRoot string, fn func() error) error {
	lockPath := filepath.Join(projectRoot, TodosDir, LockFile)
	fl := flock.New(lockPath)
	if err := fl.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock %s: %w", lockPath, err)
	}
	defer fl.Unlock()
	return fn()
}

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
		normalizeTodos(todos)
		return todos, nil
	}

	normalizeTodos(todoFile.Todos)
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

// atomicWriteFile writes data to a temp file in the same directory, fsyncs
// it, then renames it to the target path. This prevents corruption if the
// process is interrupted mid-write.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}

func saveTodoFile(projectRoot string, todoFile *types.TodoFile) error {
	todosPath := GetTodosPath(projectRoot)

	data, err := json.MarshalIndent(todoFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal todos: %w", err)
	}

	if err := atomicWriteFile(todosPath, data, 0644); err != nil {
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

	if err := atomicWriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetArchivePath returns the full path to the archive.json file
func GetArchivePath(projectRoot string) string {
	return filepath.Join(projectRoot, TodosDir, ArchiveFile)
}

// LoadArchive loads archived todos from archive.json
func LoadArchive(projectRoot string) ([]types.Todo, error) {
	archivePath := GetArchivePath(projectRoot)
	data, err := os.ReadFile(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Todo{}, nil
		}
		return nil, fmt.Errorf("failed to read archive file: %w", err)
	}

	var todoFile types.TodoFile
	if err := json.Unmarshal(data, &todoFile); err != nil {
		var todos []types.Todo
		if err := json.Unmarshal(data, &todos); err != nil {
			return nil, fmt.Errorf("failed to parse archive file: %w", err)
		}
		return todos, nil
	}
	return todoFile.Todos, nil
}

// SaveArchive saves archived todos to archive.json
func SaveArchive(projectRoot string, todos []types.Todo) error {
	archivePath := GetArchivePath(projectRoot)
	todoFile := &types.TodoFile{
		Version: 1,
		Todos:   todos,
	}
	data, err := json.MarshalIndent(todoFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal archive: %w", err)
	}
	if err := atomicWriteFile(archivePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write archive file: %w", err)
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

// FilterTodosByPriority filters todos by priority
func FilterTodosByPriority(todos []types.Todo, priority types.Priority) []types.Todo {
	var filtered []types.Todo
	for _, t := range todos {
		if t.Priority == priority {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// FilterTodosByTag filters todos by a single tag (case-insensitive).
func FilterTodosByTag(todos []types.Todo, tag string) []types.Todo {
	var filtered []types.Todo
	needle := strings.ToLower(strings.TrimSpace(tag))
	if needle == "" {
		return filtered
	}
	for _, t := range todos {
		for _, candidate := range t.Tags {
			if strings.ToLower(candidate) == needle {
				filtered = append(filtered, t)
				break
			}
		}
	}
	return filtered
}

// FilterTodosByTags filters todos that match at least one provided tag (OR semantics).
func FilterTodosByTags(todos []types.Todo, tags []string) []types.Todo {
	tagSet := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag != "" {
			tagSet[tag] = struct{}{}
		}
	}
	if len(tagSet) == 0 {
		return []types.Todo{}
	}

	var filtered []types.Todo
	for _, t := range todos {
		for _, candidate := range t.Tags {
			if _, ok := tagSet[strings.ToLower(candidate)]; ok {
				filtered = append(filtered, t)
				break
			}
		}
	}
	return filtered
}

// FilterOverdueTodos filters open todos with a due date in the past.
func FilterOverdueTodos(todos []types.Todo, now time.Time) []types.Todo {
	var filtered []types.Todo
	for _, t := range todos {
		if t.Status != types.StatusOpen || t.DueAt == nil {
			continue
		}
		if t.DueAt.Before(now) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// FilterTodosDueBefore filters todos with dueAt <= cutoff.
func FilterTodosDueBefore(todos []types.Todo, cutoff time.Time) []types.Todo {
	var filtered []types.Todo
	for _, t := range todos {
		if t.DueAt != nil && (t.DueAt.Before(cutoff) || t.DueAt.Equal(cutoff)) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// FilterTodosDueAfter filters todos with dueAt >= cutoff.
func FilterTodosDueAfter(todos []types.Todo, cutoff time.Time) []types.Todo {
	var filtered []types.Todo
	for _, t := range todos {
		if t.DueAt != nil && (t.DueAt.After(cutoff) || t.DueAt.Equal(cutoff)) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// SortTodosByPriority sorts todos in-place with highest priority first, then by creation time
func SortTodosByPriority(todos []types.Todo) {
	sort.SliceStable(todos, func(i, j int) bool {
		left := todos[i].Priority.PriorityWeight()
		right := todos[j].Priority.PriorityWeight()
		if left == right {
			return todos[i].CreatedAt.Before(todos[j].CreatedAt)
		}
		return left > right
	})
}

func normalizeTodos(todos []types.Todo) {
	for i := range todos {
		if !todos[i].Priority.IsValid() {
			todos[i].Priority = types.PriorityMedium
		}
		todos[i].Tags = normalizeTags(todos[i].Tags)
		// Keep completion timestamp consistent with status for mixed historical data.
		if todos[i].Status == types.StatusDone && todos[i].CompletedAt == nil {
			completedAt := todos[i].UpdatedAt
			if completedAt.IsZero() {
				completedAt = todos[i].CreatedAt
			}
			if !completedAt.IsZero() {
				todos[i].CompletedAt = &completedAt
			}
		}
		if todos[i].Status != types.StatusDone {
			todos[i].CompletedAt = nil
		}
	}
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
