package types

import (
	"fmt"
	"time"
)

// Status represents the current state of a todo
type Status string

const (
	StatusOpen     Status = "open"
	StatusDone     Status = "done"
	StatusBlocked  Status = "blocked"
	StatusWaiting  Status = "waiting"
	StatusTechDebt Status = "tech-debt"
)

// ValidStatuses returns all valid status values
func ValidStatuses() []Status {
	return []Status{StatusOpen, StatusDone, StatusBlocked, StatusWaiting, StatusTechDebt}
}

// IsValid checks if a status is valid
func (s Status) IsValid() bool {
	for _, valid := range ValidStatuses() {
		if s == valid {
			return true
		}
	}
	return false
}

// Priority represents the priority level of a todo
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// IsValid checks if a priority is valid
func (p Priority) IsValid() bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

// PriorityWeight gives a numeric weight for sorting (high first)
func (p Priority) PriorityWeight() int {
	switch p {
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}

// Context holds contextual information about where the todo applies
type Context struct {
	Paths  []string `json:"paths,omitempty"`
	Branch string   `json:"branch,omitempty"`
	Commit string   `json:"commit,omitempty"`
}

// Meta holds metadata about the todo
type Meta struct {
	Source string `json:"source,omitempty"`
	AIHint string `json:"aiHint,omitempty"`
}

// Todo represents a single todo item
type Todo struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Status    Status    `json:"status"`
	Priority  Priority  `json:"priority,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Context   Context   `json:"context"`
	Meta      Meta      `json:"meta,omitempty"`
}

// NewTodo creates a new todo with default values
func NewTodo(id, text string) *Todo {
	now := time.Now()
	return &Todo{
		ID:        id,
		Text:      text,
		Status:    StatusOpen,
		Priority:  PriorityMedium,
		CreatedAt: now,
		UpdatedAt: now,
		Context:   Context{},
		Meta: Meta{
			Source: "cli",
		},
	}
}

// SetPaths sets the context paths for the todo
func (t *Todo) SetPaths(paths []string) {
	t.Context.Paths = paths
	t.UpdatedAt = time.Now()
}

// SetGitContext sets the git context (branch and commit)
func (t *Todo) SetGitContext(branch, commit string) {
	t.Context.Branch = branch
	t.Context.Commit = commit
	t.UpdatedAt = time.Now()
}

// MarkDone marks the todo as done
func (t *Todo) MarkDone() {
	t.Status = StatusDone
	t.UpdatedAt = time.Now()
}

// MarkOpen marks the todo as open
func (t *Todo) MarkOpen() {
	t.Status = StatusOpen
	t.UpdatedAt = time.Now()
}

// Toggle toggles between done and open status
func (t *Todo) Toggle() {
	if t.Status == StatusDone {
		t.MarkOpen()
	} else {
		t.MarkDone()
	}
}

// Config holds per-project configuration
type Config struct {
	Version       int    `json:"version"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	AutoGit       bool   `json:"autoGit"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		AutoGit: true,
	}
}

// TodoFile represents the structure of the todos.json file
type TodoFile struct {
	Version int    `json:"version"`
	Todos   []Todo `json:"todos"`
}

// NewTodoFile creates a new todo file with default values
func NewTodoFile() *TodoFile {
	return &TodoFile{
		Version: 1,
		Todos:   []Todo{},
	}
}

// Custom error types

// ProjectNotFoundError indicates no .todos directory was found
type ProjectNotFoundError struct {
	SearchPath string
}

func (e *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("no todo project found (searched from: %s). Run 'todo init' to create one", e.SearchPath)
}

// TodoNotFoundError indicates a todo with the given ID was not found
type TodoNotFoundError struct {
	ID string
}

func (e *TodoNotFoundError) Error() string {
	return fmt.Sprintf("todo not found: %s", e.ID)
}

// InvalidStatusError indicates an invalid status was provided
type InvalidStatusError struct {
	Status string
}

func (e *InvalidStatusError) Error() string {
	return fmt.Sprintf("invalid status: %s. Valid statuses: open, done, blocked, waiting, tech-debt", e.Status)
}

// AlreadyInitializedError indicates the project is already initialized
type AlreadyInitializedError struct {
	Path string
}

func (e *AlreadyInitializedError) Error() string {
	return fmt.Sprintf("todo project already initialized at: %s", e.Path)
}
