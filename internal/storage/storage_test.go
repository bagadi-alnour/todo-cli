package storage

import (
	"testing"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/types"
)

func TestSaveAndLoadTodos(t *testing.T) {
	dir := t.TempDir()

	if _, err := InitProject(dir, true); err != nil {
		t.Fatalf("init project: %v", err)
	}

	todos := []types.Todo{
		*types.NewTodo("id1", "first task"),
		*types.NewTodo("id2", "second task"),
	}
	todos[1].Priority = types.PriorityHigh
	todos[1].Context.Paths = []string{"src"}

	if err := SaveTodos(dir, todos); err != nil {
		t.Fatalf("save todos: %v", err)
	}

	loaded, err := LoadTodos(dir)
	if err != nil {
		t.Fatalf("load todos: %v", err)
	}

	if len(loaded) != len(todos) {
		t.Fatalf("expected %d todos, got %d", len(todos), len(loaded))
	}

	if loaded[1].Priority != types.PriorityHigh {
		t.Fatalf("expected priority %s, got %s", types.PriorityHigh, loaded[1].Priority)
	}
	if loaded[1].Context.Paths[0] != "src" {
		t.Fatalf("expected path src, got %v", loaded[1].Context.Paths)
	}
}

func TestFiltersAndFinders(t *testing.T) {
	todos := []types.Todo{
		{ID: "a1", Text: "open item", Status: types.StatusOpen, Priority: types.PriorityHigh, Context: types.Context{Paths: []string{"src/pkg"}}},
		{ID: "a2", Text: "done item", Status: types.StatusDone, Priority: types.PriorityLow, Context: types.Context{Paths: []string{"docs"}}},
		{ID: "a3", Text: "blocked", Status: types.StatusBlocked, Priority: types.PriorityMedium, Context: types.Context{Paths: []string{"src/ui"}}},
	}

	if got := FilterTodosByStatus(todos, types.StatusOpen); len(got) != 1 {
		t.Fatalf("expected 1 open todo, got %d", len(got))
	}

	if got := FilterTodosByPath(todos, "src"); len(got) != 2 {
		t.Fatalf("expected 2 todos under src, got %d", len(got))
	}

	if got := FilterTodosByPriority(todos, types.PriorityHigh); len(got) != 1 || got[0].ID != "a1" {
		t.Fatalf("priority filter returned %+v", got)
	}

	if todo, idx := FindTodoByIDOrIndex(todos, "a2"); todo == nil || idx != 1 {
		t.Fatalf("find by id failed, got %v at %d", todo, idx)
	}

	if todo, idx := FindTodoByIDOrIndex(todos, "2"); todo == nil || todo.ID != "a2" || idx != 1 {
		t.Fatalf("find by index failed, got %v at %d", todo, idx)
	}

	if todo, _ := FindTodoByIDOrIndex(todos, "zzz"); todo != nil {
		t.Fatalf("expected nil todo for unknown id")
	}
}

func TestSortTodosByPriority(t *testing.T) {
	now := time.Now()
	todos := []types.Todo{
		{ID: "low", Priority: types.PriorityLow, CreatedAt: now},
		{ID: "high1", Priority: types.PriorityHigh, CreatedAt: now.Add(time.Minute)},
		{ID: "high0", Priority: types.PriorityHigh, CreatedAt: now.Add(-time.Minute)},
		{ID: "medium", Priority: types.PriorityMedium, CreatedAt: now},
	}

	SortTodosByPriority(todos)

	expectedOrder := []string{"high0", "high1", "medium", "low"}
	for i, id := range expectedOrder {
		if todos[i].ID != id {
			t.Fatalf("expected %s at position %d, got %s", id, i, todos[i].ID)
		}
	}
}
