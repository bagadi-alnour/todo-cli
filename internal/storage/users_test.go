package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bagadi-alnour/todo-cli/internal/types"
)

func TestSlugFromGitName(t *testing.T) {
	tests := map[string]string{
		"Bagadi ALNOUR":  "bagadi-alnour",
		"  Jane   Doe  ": "jane-doe",
		"O'Brien Smith": "obrien-smith",
		"":              unknownOwnerSlug,
		"!!!":           unknownOwnerSlug,
	}
	for in, want := range tests {
		if got := SlugFromGitName(in); got != want {
			t.Fatalf("SlugFromGitName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSaveAndLoadPerUserTodos(t *testing.T) {
	t.Setenv("TODO_USER_NAME", "Alice Example")
	dir := t.TempDir()
	if _, err := InitProject(dir, true); err != nil {
		t.Fatalf("init: %v", err)
	}

	todos := []types.Todo{
		*types.NewTodo("a1", "alice task"),
		*types.NewTodo("b1", "bob task"),
	}
	todos[0].CreatedBy = "alice-example"
	todos[1].CreatedBy = "bob-builder"

	if err := SaveTodos(dir, todos); err != nil {
		t.Fatalf("save: %v", err)
	}

	alicePath := filepath.Join(dir, TodosDir, UsersDir, "alice-example.json")
	bobPath := filepath.Join(dir, TodosDir, UsersDir, "bob-builder.json")
	if _, err := os.Stat(alicePath); err != nil {
		t.Fatalf("missing alice file: %v", err)
	}
	if _, err := os.Stat(bobPath); err != nil {
		t.Fatalf("missing bob file: %v", err)
	}

	loaded, err := LoadTodos(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(loaded))
	}
}

func TestMigrateLegacyTodos(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitProject(dir, true); err != nil {
		t.Fatalf("init: %v", err)
	}

	legacy := []types.Todo{*types.NewTodo("legacy1", "from old file")}
	if err := saveTodoFile(dir, &types.TodoFile{Version: 1, Todos: legacy}); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	loaded, err := LoadTodos(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 migrated todo, got %d", len(loaded))
	}
	if loaded[0].CreatedBy != legacyOwnerSlug {
		t.Fatalf("expected createdBy %q, got %q", legacyOwnerSlug, loaded[0].CreatedBy)
	}

	legacyPath := GetTodosPath(dir)
	legacyTodos, err := loadTodosFile(legacyPath)
	if err != nil {
		t.Fatalf("read legacy: %v", err)
	}
	if len(legacyTodos) != 0 {
		t.Fatalf("expected legacy todos.json to be empty after migration, got %d", len(legacyTodos))
	}
}
