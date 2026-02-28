package agent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

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
	done      chan struct{}
	cancel    context.CancelFunc
}

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

func (tm *TaskManager) Launch(ctx context.Context, agentName string, a *Agent, task string) string {
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

		resp, err := a.Chat(taskCtx, task)

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

func (tm *TaskManager) GetResult(ctx context.Context, taskID string, wait bool, timeout time.Duration) (*backgroundTask, error) {
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

func (tm *TaskManager) CancelAll() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	for _, bt := range tm.tasks {
		bt.cancel()
	}
}

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
