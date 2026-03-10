package agent

import (
	"context"
	"time"

	llm "github.com/joakimcarlsson/ai/providers"
)

// ObserverEventType identifies the category of an observer event.
type ObserverEventType string

const (
	EventTaskLaunched  ObserverEventType = "task_launched"
	EventTaskCompleted ObserverEventType = "task_completed"
	EventTaskFailed    ObserverEventType = "task_failed"
	EventTaskCancelled ObserverEventType = "task_cancelled"
	EventTaskPanicked  ObserverEventType = "task_panicked"
	EventTurnStarted   ObserverEventType = "turn_started"
	EventTurnCompleted ObserverEventType = "turn_completed"
	EventTurnErrored   ObserverEventType = "turn_errored"
	EventToolStarted   ObserverEventType = "tool_started"
	EventToolSucceeded ObserverEventType = "tool_succeeded"
	EventToolErrored   ObserverEventType = "tool_errored"
)

// ObserverEvent is a structured lifecycle event emitted during agent execution.
// Fields are populated based on Type — irrelevant fields remain at their zero value.
type ObserverEvent struct {
	Type       ObserverEventType
	Timestamp  time.Time
	AgentName  string
	TaskID     string
	TurnIndex  int
	ToolCallID string
	ToolName   string
	Duration   time.Duration
	Usage      llm.TokenUsage
	ToolCount  int
	Error      string
	PanicStack string
}

// Observer receives runtime telemetry events from the agent framework.
// Implementations must be safe for concurrent calls.
type Observer interface {
	OnEvent(event ObserverEvent)
}

// ObserverFunc adapts a plain function to the Observer interface.
type ObserverFunc func(ObserverEvent)

func (f ObserverFunc) OnEvent(event ObserverEvent) { f(event) }

// MultiObserver fans out events to multiple observers.
type MultiObserver []Observer

func (m MultiObserver) OnEvent(event ObserverEvent) {
	for _, obs := range m {
		obs.OnEvent(event)
	}
}

type taskScopeKey struct{}

type taskScope struct {
	TaskID    string
	AgentName string
}

func withTaskScope(ctx context.Context, taskID, agentName string) context.Context {
	return context.WithValue(ctx, taskScopeKey{}, taskScope{
		TaskID:    taskID,
		AgentName: agentName,
	})
}

func taskScopeFromContext(ctx context.Context) (taskID, agentName string) {
	if s, ok := ctx.Value(taskScopeKey{}).(taskScope); ok {
		return s.TaskID, s.AgentName
	}
	return "", ""
}
