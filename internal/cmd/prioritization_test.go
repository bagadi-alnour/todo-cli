package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/types"
)

func TestSortTodosForExecution(t *testing.T) {
	now := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)
	overdue := now.Add(-time.Hour)
	soon := now.Add(2 * time.Hour)
	later := now.Add(24 * time.Hour)

	todos := []types.Todo{
		{ID: "low-no-due", Priority: types.PriorityLow, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "high-no-due", Priority: types.PriorityHigh, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "due-later", Priority: types.PriorityLow, DueAt: &later, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "due-soon", Priority: types.PriorityLow, DueAt: &soon, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "overdue", Priority: types.PriorityLow, DueAt: &overdue, CreatedAt: now.Add(-24 * time.Hour)},
	}

	sortTodosForExecution(todos, now)

	expected := []string{"overdue", "due-soon", "due-later", "high-no-due", "low-no-due"}
	for i := range expected {
		if todos[i].ID != expected[i] {
			t.Fatalf("unexpected order at %d: got %s want %s", i, todos[i].ID, expected[i])
		}
	}
}

func TestNextReason(t *testing.T) {
	now := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)
	overdue := now.Add(-time.Hour)
	soon := now.Add(4 * time.Hour)

	reason := nextReason(types.Todo{DueAt: &overdue, Priority: types.PriorityMedium}, now)
	if reason == "" {
		t.Fatal("expected non-empty overdue reason")
	}
	if !strings.Contains(reason, "overdue") {
		t.Fatalf("expected overdue in reason, got: %q", reason)
	}

	reason = nextReason(types.Todo{DueAt: &soon, Priority: types.PriorityMedium}, now)
	if reason == "" {
		t.Fatal("expected non-empty due reason")
	}
	if !strings.Contains(reason, "due in") {
		t.Fatalf("expected 'due in' in reason, got: %q", reason)
	}

	reason = nextReason(types.Todo{Priority: types.PriorityHigh, CreatedAt: now.Add(-48 * time.Hour)}, now)
	if reason == "" {
		t.Fatal("expected non-empty priority reason")
	}
	if !strings.Contains(reason, "high priority") {
		t.Fatalf("expected 'high priority' in reason, got: %q", reason)
	}
}
