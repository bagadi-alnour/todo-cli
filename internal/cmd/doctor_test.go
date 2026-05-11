package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/types"
)

func TestApplyDoctorFixes(t *testing.T) {
	projectRoot := t.TempDir()
	validPath := filepath.Join(projectRoot, "keep.txt")
	if err := os.WriteFile(validPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("setup file: %v", err)
	}

	now := time.Now()
	todos := []types.Todo{
		{ID: "1", Text: "orphaned", CreatedAt: now, UpdatedAt: now, Context: types.Context{Paths: []string{"missing.txt"}}},
		{ID: "2", Text: "duplicate", CreatedAt: now, UpdatedAt: now, Context: types.Context{Paths: []string{"keep.txt"}}},
		{ID: "3", Text: "duplicate", CreatedAt: now, UpdatedAt: now},
		{ID: "4", Text: "   ", CreatedAt: now, UpdatedAt: now},
	}

	cleaned, report := applyDoctorFixes(todos, projectRoot)

	if report.removedEmpty != 1 {
		t.Fatalf("expected 1 empty removal, got %d", report.removedEmpty)
	}
	if report.removedDuplicates != 1 {
		t.Fatalf("expected 1 duplicate removal, got %d", report.removedDuplicates)
	}
	if report.removedOrphanedPaths != 1 {
		t.Fatalf("expected 1 orphaned path removal, got %d", report.removedOrphanedPaths)
	}

	if len(cleaned) != 2 {
		t.Fatalf("expected 2 todos after cleanup, got %d", len(cleaned))
	}

	for _, todo := range cleaned {
		if todo.ID == "1" && len(todo.Context.Paths) != 0 {
			t.Fatalf("expected orphaned paths removed, got %v", todo.Context.Paths)
		}
	}
}
