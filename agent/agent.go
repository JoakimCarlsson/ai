package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/joakimcarlsson/ai/agent/memory"
	"github.com/joakimcarlsson/ai/agent/session"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tool"
)

// Agent is an AI assistant that can chat with users, use tools, and maintain memory.
// Create one using New() with functional options.
type Agent struct {
	llm                 llm.LLM
	memoryLLM           llm.LLM
	tools               []tool.BaseTool
	systemPrompt        string
	maxIterations       int
	autoExecute         bool
	memory              memory.Store
	memoryID            string
	autoExtract         bool
	autoDedup           bool
	session             session.Session
	contextStrategy     tokens.Strategy
	reserveTokens       int64
	maxContextTokens    int64
	parallelTools       bool
	maxParallelTools    int
	state               map[string]any
	instructionProvider func(ctx context.Context, state map[string]any) (string, error)
	handoffs            []HandoffConfig
	taskManager         *TaskManager
	observer            Observer
}

func (a *Agent) getMemoryLLM() llm.LLM {
	if a.memoryLLM != nil {
		return a.memoryLLM
	}
	return a.llm
}

// New creates a new Agent with the given LLM client and options.
// The agent can be configured with tools, memory, session persistence, and more.
//
// Example:
//
//	agent := agent.New(llmClient,
//	    agent.WithSystemPrompt("You are a helpful assistant."),
//	    agent.WithTools(&myTool{}),
//	    agent.WithSession("conv-1", session.FileStore("./sessions")),
//	    agent.WithMemory("user-123", myMemoryStore, memory.AutoExtract()),
//	)
func New(llmClient llm.LLM, opts ...AgentOption) *Agent {
	a := &Agent{
		llm:           llmClient,
		tools:         make([]tool.BaseTool, 0),
		maxIterations: 0,
		autoExecute:   true,
		parallelTools: true,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

func (a *Agent) getTools() []tool.BaseTool {
	allTools := make([]tool.BaseTool, len(a.tools))
	copy(allTools, a.tools)

	if a.memory != nil && !a.autoExtract && a.memoryID != "" {
		memoryTools := createMemoryTools(a.memory, a.memoryID)
		allTools = append(allTools, memoryTools...)
	}

	if a.taskManager != nil {
		allTools = append(allTools, createTaskTools()...)
	}

	return allTools
}

// ParseToolInput parses a JSON tool input string into the specified type.
// This is a helper function for implementing tool.BaseTool.Run().
func ParseToolInput[T any](input string) (T, error) {
	var result T
	err := json.Unmarshal([]byte(input), &result)
	return result, err
}

func (a *Agent) emitEvent(ctx context.Context, evt ObserverEvent) {
	if a.observer == nil {
		return
	}
	evt.Timestamp = time.Now()
	if taskID, agentName := taskScopeFromContext(ctx); taskID != "" {
		evt.TaskID = taskID
		evt.AgentName = agentName
	}
	a.observer.OnEvent(evt)
}
