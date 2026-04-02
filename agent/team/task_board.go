package team

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// TaskStatus represents the lifecycle state of a task on the board.
type TaskStatus string

const (
	// TaskOpen indicates the task is available to be claimed.
	TaskOpen TaskStatus = "open"
	// TaskClaimed indicates the task has been assigned to a teammate.
	TaskClaimed TaskStatus = "claimed"
	// TaskCompleted indicates the task has been finished.
	TaskCompleted TaskStatus = "completed"
)

// Task represents a unit of work on the team's shared task board.
type Task struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Status    TaskStatus `json:"status"`
	Assignee  string     `json:"assignee,omitempty"`
	Result    string     `json:"result,omitempty"`
	CreatedBy string     `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TaskBoard is a shared, thread-safe task list for team coordination.
type TaskBoard struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	idGen atomic.Int64
}

// NewTaskBoard creates a new empty task board.
func NewTaskBoard() *TaskBoard {
	return &TaskBoard{
		tasks: make(map[string]*Task),
	}
}

// Create adds a new open task to the board.
func (tb *TaskBoard) Create(title, creator string) *Task {
	now := time.Now()
	t := &Task{
		ID:        fmt.Sprintf("board-%d", tb.idGen.Add(1)),
		Title:     title,
		Status:    TaskOpen,
		CreatedBy: creator,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tb.mu.Lock()
	tb.tasks[t.ID] = t
	tb.mu.Unlock()

	return t
}

// Claim assigns an open task to the given assignee.
func (tb *TaskBoard) Claim(taskID, assignee string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	t, ok := tb.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	if t.Status != TaskOpen {
		return fmt.Errorf("task %q is not open (status: %s)", taskID, t.Status)
	}

	t.Status = TaskClaimed
	t.Assignee = assignee
	t.UpdatedAt = time.Now()
	return nil
}

// Complete marks a claimed task as completed with the given result.
func (tb *TaskBoard) Complete(taskID, assignee, result string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	t, ok := tb.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	if t.Assignee != assignee {
		return fmt.Errorf(
			"task %q is assigned to %q, not %q",
			taskID,
			t.Assignee,
			assignee,
		)
	}

	t.Status = TaskCompleted
	t.Result = result
	t.UpdatedAt = time.Now()
	return nil
}

// List returns a snapshot of all tasks on the board.
func (tb *TaskBoard) List() []*Task {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	result := make([]*Task, 0, len(tb.tasks))
	for _, t := range tb.tasks {
		snapshot := *t
		result = append(result, &snapshot)
	}
	return result
}
