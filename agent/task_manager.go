package agent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// TaskStatus represents the lifecycle state of a background task.
type TaskStatus string

const (
	// TaskRunning indicates the task is currently executing.
	TaskRunning TaskStatus = "running"
	// TaskCompleted indicates the task finished successfully.
	TaskCompleted TaskStatus = "completed"
	// TaskFailed indicates the task encountered an error.
	TaskFailed TaskStatus = "failed"
	// TaskCancelled indicates the task was explicitly cancelled.
	TaskCancelled TaskStatus = "cancelled"
)

type backgroundTask struct {
	ID        string
	AgentName string
	Status    TaskStatus
	Result    string
	Error     string
	done      chan struct{}
	cancel    context.CancelFunc
}

// TaskManager coordinates background sub-agent tasks. It tracks task lifecycle,
// supports blocking and non-blocking result retrieval with optional timeouts,
// and provides bulk cancellation for cleanup when the parent agent finishes.
type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*backgroundTask
	wg    sync.WaitGroup
	idGen atomic.Int64
}

func newTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*backgroundTask),
	}
}

// Launch starts a background task that runs the given agent with the provided task message.
// It returns a unique task ID that can be used with GetResult, Stop, or ListAll.
func (tm *TaskManager) Launch(
	ctx context.Context,
	agentName string,
	a *Agent,
	task string,
	opts ...ChatOption,
) string {
	id := fmt.Sprintf("task-%d", tm.idGen.Add(1))

	taskCtx, cancel := context.WithCancel(ctx)
	bt := &backgroundTask{
		ID:        id,
		AgentName: agentName,
		Status:    TaskRunning,
		done:      make(chan struct{}),
		cancel:    cancel,
	}

	tm.mu.Lock()
	tm.tasks[id] = bt
	tm.mu.Unlock()

	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()
		defer close(bt.done)

		resp, err := a.Chat(taskCtx, task, opts...)

		tm.mu.Lock()
		defer tm.mu.Unlock()

		if taskCtx.Err() != nil {
			bt.Status = TaskCancelled
			bt.Error = "task was cancelled"
			return
		}
		if err != nil {
			bt.Status = TaskFailed
			bt.Error = err.Error()
			return
		}
		bt.Status = TaskCompleted
		bt.Result = resp.Content
	}()

	return id
}

// GetResult retrieves the current state of a background task. If wait is true, it blocks
// until the task completes or the timeout expires. A zero timeout means wait indefinitely.
func (tm *TaskManager) GetResult(
	ctx context.Context,
	taskID string,
	wait bool,
	timeout time.Duration,
) (*backgroundTask, error) {
	tm.mu.RLock()
	bt, ok := tm.tasks[taskID]
	tm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task %q not found", taskID)
	}

	if wait {
		if timeout > 0 {
			select {
			case <-bt.done:
			case <-time.After(timeout):
				// Timeout expired — return current snapshot (likely still "running")
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		} else {
			select {
			case <-bt.done:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	tm.mu.RLock()
	snapshot := *bt
	tm.mu.RUnlock()

	return &snapshot, nil
}

// Stop cancels a running background task by its ID.
func (tm *TaskManager) Stop(taskID string) error {
	tm.mu.RLock()
	bt, ok := tm.tasks[taskID]
	tm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	bt.cancel()
	return nil
}

// ListAll returns a snapshot of all tracked background tasks regardless of status.
func (tm *TaskManager) ListAll() []*backgroundTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]*backgroundTask, 0, len(tm.tasks))
	for _, bt := range tm.tasks {
		snapshot := *bt
		result = append(result, &snapshot)
	}
	return result
}

// CancelAll cancels every tracked background task.
func (tm *TaskManager) CancelAll() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	for _, bt := range tm.tasks {
		bt.cancel()
	}
}

// WaitAll blocks until every tracked background task has finished.
func (tm *TaskManager) WaitAll() {
	tm.wg.Wait()
}

type taskManagerKey struct{}

func withTaskManager(ctx context.Context, tm *TaskManager) context.Context {
	return context.WithValue(ctx, taskManagerKey{}, tm)
}

func taskManagerFromContext(ctx context.Context) *TaskManager {
	tm, _ := ctx.Value(taskManagerKey{}).(*TaskManager)
	return tm
}
