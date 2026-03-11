package agent

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joakimcarlsson/ai/types"
)

// TaskStatus represents the lifecycle state of a background task.
type TaskStatus string

const (
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

type backgroundTask struct {
	ID        string
	AgentName string
	Status    TaskStatus
	Result    string
	Error     string
	StartedAt time.Time
	EndedAt   time.Time
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
	hooks []Hooks
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
	startedAt := time.Now()

	taskCtx, cancel := context.WithCancel(ctx)
	bt := &backgroundTask{
		ID:        id,
		AgentName: agentName,
		Status:    TaskRunning,
		StartedAt: startedAt,
		done:      make(chan struct{}),
		cancel:    cancel,
	}

	tm.mu.Lock()
	tm.tasks[id] = bt
	tm.mu.Unlock()

	_, _, lineage := taskScopeFromContext(ctx)
	runSubagentStart(ctx, tm.hooks, SubagentEventContext{
		TaskID:    id,
		AgentName: agentName,
		Task:      task,
		Lineage:   lineage,
	})

	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()
		defer close(bt.done)
		defer func() {
			if r := recover(); r != nil {
				endedAt := time.Now()
				panicMsg := fmt.Sprintf("panic: %v", r)
				_ = debug.Stack()

				tm.mu.Lock()
				bt.Status = TaskFailed
				bt.Error = panicMsg
				bt.EndedAt = endedAt
				tm.mu.Unlock()

				runSubagentStop(ctx, tm.hooks, SubagentEventContext{
					TaskID:    id,
					AgentName: agentName,
					Task:      task,
					Lineage:   lineage,
					Error:     fmt.Errorf("%s", panicMsg),
					Duration:  endedAt.Sub(startedAt),
				})
			}
		}()

		scopedCtx := withTaskScope(taskCtx, id, agentName)
		resp, err := runTaskStream(scopedCtx, a, task, opts...)
		endedAt := time.Now()
		duration := endedAt.Sub(startedAt)

		tm.mu.Lock()
		bt.EndedAt = endedAt

		if taskCtx.Err() != nil {
			bt.Status = TaskCancelled
			bt.Error = "task was cancelled"
			tm.mu.Unlock()

			runSubagentStop(ctx, tm.hooks, SubagentEventContext{
				TaskID:    id,
				AgentName: agentName,
				Task:      task,
				Lineage:   lineage,
				Error:     fmt.Errorf("task was cancelled"),
				Duration:  duration,
			})
			return
		}
		if err != nil {
			bt.Status = TaskFailed
			bt.Error = err.Error()
			tm.mu.Unlock()

			runSubagentStop(ctx, tm.hooks, SubagentEventContext{
				TaskID:    id,
				AgentName: agentName,
				Task:      task,
				Lineage:   lineage,
				Error:     err,
				Duration:  duration,
			})
			return
		}
		bt.Status = TaskCompleted
		bt.Result = resp.Content
		tm.mu.Unlock()

		runSubagentStop(ctx, tm.hooks, SubagentEventContext{
			TaskID:    id,
			AgentName: agentName,
			Task:      task,
			Lineage:   lineage,
			Result:    resp.Content,
			Duration:  duration,
		})
	}()

	return id
}

func runTaskStream(
	ctx context.Context,
	a *Agent,
	task string,
	opts ...ChatOption,
) (*ChatResponse, error) {
	var final *ChatResponse
	var content strings.Builder

	for event := range a.ChatStream(ctx, task, opts...) {
		switch event.Type {
		case types.EventContentDelta:
			content.WriteString(event.Content)
		case types.EventComplete:
			if event.Response != nil {
				final = event.Response
			}
		case types.EventError:
			if event.Error != nil {
				return nil, event.Error
			}
			return nil, fmt.Errorf("background task stream failed")
		}
	}

	if final != nil {
		return final, nil
	}

	if content.Len() > 0 {
		return &ChatResponse{Content: content.String()}, nil
	}

	return nil, fmt.Errorf("background task stream ended without completion")
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
