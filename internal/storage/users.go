package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/types"
)

const (
	UsersDir         = "users"
	legacyOwnerSlug  = "legacy"
	unknownOwnerSlug = "unknown"
)

// SlugFromGitName converts a git user.name to firstname-lastname.json style slug.
func SlugFromGitName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return unknownOwnerSlug
	}
	var parts []string
	for _, field := range strings.Fields(name) {
		var b strings.Builder
		for _, r := range field {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			}
		}
		if s := strings.ToLower(b.String()); s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return unknownOwnerSlug
	}
	return strings.Join(parts, "-")
}

// CurrentUserSlug returns the slug for the configured git user.name.
// Set TODO_USER_NAME to override (useful in CI/tests).
func CurrentUserSlug() (string, error) {
	if override := strings.TrimSpace(os.Getenv("TODO_USER_NAME")); override != "" {
		return slugFromNameOrError(override)
	}
	name, err := git.GetUserName()
	if err != nil {
		return "", fmt.Errorf("git user.name is required for per-user todos (or set TODO_USER_NAME): %w", err)
	}
	return slugFromNameOrError(name)
}

func slugFromNameOrError(name string) (string, error) {
	slug := SlugFromGitName(name)
	if slug == unknownOwnerSlug {
		return "", fmt.Errorf("name %q could not be converted to a user slug (expected firstname-lastname)", name)
	}
	return slug, nil
}

// ApplyCreator sets CreatedBy on a new todo from the current user slug.
func ApplyCreator(todo *types.Todo) error {
	slug, err := CurrentUserSlug()
	if err != nil {
		return err
	}
	todo.CreatedBy = slug
	return nil
}

func usersDir(projectRoot string) string {
	return filepath.Join(projectRoot, TodosDir, UsersDir)
}

func userTodosPath(projectRoot, slug string) string {
	slug = normalizeOwnerSlug(slug)
	return filepath.Join(usersDir(projectRoot), slug+".json")
}

func normalizeOwnerSlug(slug string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return unknownOwnerSlug
	}
	var b strings.Builder
	prevHyphen := false
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
			continue
		}
		if r == '-' || r == '_' || r == ' ' {
			if !prevHyphen && b.Len() > 0 {
				b.WriteRune('-')
				prevHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return unknownOwnerSlug
	}
	return out
}

func ownerSlugFromFilename(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return normalizeOwnerSlug(base)
}

func loadTodosFile(path string) ([]types.Todo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Todo{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var todoFile types.TodoFile
	if err := json.Unmarshal(data, &todoFile); err != nil {
		var todos []types.Todo
		if err := json.Unmarshal(data, &todos); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		normalizeTodos(todos)
		return todos, nil
	}
	normalizeTodos(todoFile.Todos)
	return todoFile.Todos, nil
}

func saveTodosFile(path string, todos []types.Todo) error {
	todoFile := &types.TodoFile{Version: 1, Todos: todos}
	data, err := json.MarshalIndent(todoFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal todos: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := atomicWriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func ensureUsersDir(projectRoot string) error {
	return os.MkdirAll(usersDir(projectRoot), 0755)
}

// migrateLegacyTodos moves monolithic todos.json into users/<slug>.json by createdBy.
func migrateLegacyTodos(projectRoot string) error {
	legacyPath := GetTodosPath(projectRoot)
	todos, err := loadTodosFile(legacyPath)
	if err != nil {
		return err
	}
	if len(todos) == 0 {
		return nil
	}

	if err := ensureUsersDir(projectRoot); err != nil {
		return err
	}

	byOwner := map[string][]types.Todo{}
	for i := range todos {
		slug := normalizeOwnerSlug(todos[i].CreatedBy)
		if slug == unknownOwnerSlug {
			slug = legacyOwnerSlug
			todos[i].CreatedBy = legacyOwnerSlug
		}
		byOwner[slug] = append(byOwner[slug], todos[i])
	}

	for slug, group := range byOwner {
		path := userTodosPath(projectRoot, slug)
		existing, err := loadTodosFile(path)
		if err != nil {
			return err
		}
		merged := mergeTodosByID(existing, group)
		if err := saveTodosFile(path, merged); err != nil {
			return err
		}
	}

	// Clear legacy file after successful migration.
	if err := saveTodosFile(legacyPath, []types.Todo{}); err != nil {
		return err
	}
	return nil
}

func mergeTodosByID(existing, incoming []types.Todo) []types.Todo {
	seen := make(map[string]struct{}, len(existing))
	out := make([]types.Todo, 0, len(existing)+len(incoming))
	for _, t := range existing {
		seen[t.ID] = struct{}{}
		out = append(out, t)
	}
	for _, t := range incoming {
		if _, ok := seen[t.ID]; ok {
			continue
		}
		out = append(out, t)
	}
	return out
}

func loadAllUserTodos(projectRoot string) ([]types.Todo, error) {
	if err := ensureUsersDir(projectRoot); err != nil {
		return nil, err
	}

	dir := usersDir(projectRoot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Todo{}, nil
		}
		return nil, err
	}

	byID := make(map[string]types.Todo)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		slug := ownerSlugFromFilename(entry.Name())
		path := filepath.Join(dir, entry.Name())
		todos, err := loadTodosFile(path)
		if err != nil {
			return nil, err
		}
		for _, t := range todos {
			if t.CreatedBy == "" {
				t.CreatedBy = slug
			}
			byID[t.ID] = t
		}
	}

	out := make([]types.Todo, 0, len(byID))
	for _, t := range byID {
		out = append(out, t)
	}
	normalizeTodos(out)
	return out, nil
}

func saveTodosByOwner(projectRoot string, todos []types.Todo) error {
	if err := ensureUsersDir(projectRoot); err != nil {
		return err
	}

	byOwner := make(map[string][]types.Todo)
	for _, t := range todos {
		slug := normalizeOwnerSlug(t.CreatedBy)
		if slug == unknownOwnerSlug {
			slug = legacyOwnerSlug
		}
		byOwner[slug] = append(byOwner[slug], t)
	}

	// Write each owner file present in the save set.
	written := make(map[string]struct{}, len(byOwner))
	for slug, group := range byOwner {
		if err := saveTodosFile(userTodosPath(projectRoot, slug), group); err != nil {
			return err
		}
		written[slug] = struct{}{}
	}

	// Clear user files that no longer have any todos.
	entries, err := os.ReadDir(usersDir(projectRoot))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		slug := ownerSlugFromFilename(entry.Name())
		if _, ok := written[slug]; ok {
			continue
		}
		if err := saveTodosFile(filepath.Join(usersDir(projectRoot), entry.Name()), []types.Todo{}); err != nil {
			return err
		}
	}
	return nil
}
