package cmd

import (
	"sort"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/types"
)

func sortTodosForExecution(todos []types.Todo, now time.Time) {
	sort.SliceStable(todos, func(i, j int) bool {
		left := todos[i]
		right := todos[j]

		leftOverdue := isOverdueDueDate(left.DueAt, now)
		rightOverdue := isOverdueDueDate(right.DueAt, now)
		if leftOverdue != rightOverdue {
			return leftOverdue
		}

		leftHasDue := left.DueAt != nil
		rightHasDue := right.DueAt != nil
		if leftHasDue != rightHasDue {
			return leftHasDue
		}

		if leftHasDue && rightHasDue && !left.DueAt.Equal(*right.DueAt) {
			return left.DueAt.Before(*right.DueAt)
		}

		leftPriority := priorityWeight(left.Priority)
		rightPriority := priorityWeight(right.Priority)
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}

		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.Before(right.CreatedAt)
		}

		return left.ID < right.ID
	})
}

func priorityWeight(p types.Priority) int {
	if !p.IsValid() {
		return types.PriorityMedium.PriorityWeight()
	}
	return p.PriorityWeight()
}
