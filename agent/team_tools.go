package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
)

type spawnTeammateInput struct {
	Name         string `json:"name"          desc:"Unique name for the teammate (used for messaging)"`
	Template     string `json:"template"      desc:"Template name to use for the teammate's configuration. Falls back to name if empty." required:"false"`
	Task         string `json:"task"          desc:"The task to assign to the teammate"`
	SystemPrompt string `json:"system_prompt" desc:"System prompt for the teammate agent"              required:"false"`
	MaxTurns     int    `json:"max_turns"     desc:"Maximum tool-execution turns. 0 uses default."     required:"false"`
	Timeout      int    `json:"timeout"       desc:"Timeout in seconds for the teammate. 0 uses team default." required:"false"`
}

type spawnTeammateTool struct {
	leadAgent *Agent
}

func (t *spawnTeammateTool) Info() tool.Info {
	return tool.NewInfo(
		"spawn_teammate",
		"Spawn a new teammate agent that runs concurrently. The teammate can communicate with other team members via send_message and read_messages.",
		spawnTeammateInput{},
	)
}

func (t *spawnTeammateTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input spawnTeammateInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(
			"invalid parameters: " + err.Error(),
		), nil
	}

	if input.Name == "" {
		return tool.NewTextErrorResponse("name is required"), nil
	}
	if input.Task == "" {
		return tool.NewTextErrorResponse("task is required"), nil
	}

	tm := team.FromContext(ctx)
	if tm == nil {
		return tool.NewTextErrorResponse("no team available"), nil
	}

	var teammate *Agent
	templateKey := input.Template
	if templateKey == "" {
		templateKey = input.Name
	}
	if tmpl, ok := t.leadAgent.teammateTemplates[templateKey]; ok {
		teammate = tmpl
	} else {
		prompt := input.SystemPrompt
		if prompt == "" {
			prompt = fmt.Sprintf("You are a teammate named %q. Complete the assigned task and communicate with your team using send_message and read_messages.", input.Name)
		}
		teammate = New(t.leadAgent.llm,
			WithSystemPrompt(prompt),
		)
	}

	if len(t.leadAgent.hooks) > 0 && len(teammate.hooks) == 0 {
		teammate.hooks = t.leadAgent.hooks
	}

	var opts []ChatOption
	if input.MaxTurns > 0 {
		opts = append(opts, WithMaxTurns(input.MaxTurns))
	}

	var timeout time.Duration
	if input.Timeout > 0 {
		timeout = time.Duration(input.Timeout) * time.Second
	} else {
		timeout = tm.DefaultTimeout()
	}

	eventChan := teamEventChanFromContext(ctx)

	memberID, err := spawnTeammate(
		ctx,
		tm,
		input.Name,
		teammate,
		input.Task,
		timeout,
		t.leadAgent.hooks,
		eventChan,
		opts...)
	if err != nil {
		return tool.NewTextErrorResponse(
			fmt.Sprintf("failed to spawn teammate: %s", err.Error()),
		), nil
	}

	type spawnOutput struct {
		MemberID string `json:"member_id"`
		Name     string `json:"name"`
		Status   string `json:"status"`
	}
	return tool.NewJSONResponse(spawnOutput{
		MemberID: memberID,
		Name:     input.Name,
		Status:   "active",
	}), nil
}

type sendMessageInput struct {
	To      string `json:"to"      desc:"Recipient teammate name, or '*' for broadcast to all teammates"`
	Content string `json:"content" desc:"Message content to send"`
}

type sendMessageTool struct{}

func (t *sendMessageTool) Info() tool.Info {
	return tool.NewInfo(
		"send_message",
		"Send a message to a teammate or broadcast to all teammates. Messages can be read by recipients using read_messages.",
		sendMessageInput{},
	)
}

func (t *sendMessageTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input sendMessageInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(
			"invalid parameters: " + err.Error(),
		), nil
	}

	if input.To == "" {
		return tool.NewTextErrorResponse("to is required"), nil
	}
	if input.Content == "" {
		return tool.NewTextErrorResponse("content is required"), nil
	}

	tm := team.FromContext(ctx)
	if tm == nil {
		return tool.NewTextErrorResponse("no team available"), nil
	}

	_, sender, _ := taskScopeFromContext(ctx)
	if sender == "" {
		if team.IsLead(ctx) {
			sender = "__lead__"
		} else {
			sender = "unknown"
		}
	}

	msg := team.Message{
		From:    sender,
		To:      input.To,
		Content: input.Content,
	}

	if err := tm.Mailbox.Send(ctx, msg); err != nil {
		return tool.NewTextErrorResponse(
			fmt.Sprintf("failed to send message: %s", err.Error()),
		), nil
	}

	if hooks := teamHooksFromContext(ctx); len(hooks) > 0 {
		runOnTeamMessage(ctx, hooks, TeamMessageContext{
			TeamName: tm.Name(),
			Message:  msg,
		})
	}

	if eventChan := teamEventChanFromContext(ctx); eventChan != nil {
		eventChan <- ChatEvent{
			Type:      types.EventTeamMessage,
			AgentName: sender,
		}
	}

	return tool.NewTextResponse(
		fmt.Sprintf("Message sent to %s.", input.To),
	), nil
}

type readMessagesInput struct{}

type readMessagesTool struct{}

func (t *readMessagesTool) Info() tool.Info {
	return tool.NewInfo(
		"read_messages",
		"Read all unread messages from your inbox. Messages are removed from the inbox after reading.",
		readMessagesInput{},
	)
}

func (t *readMessagesTool) Run(
	ctx context.Context,
	_ tool.Call,
) (tool.Response, error) {
	tm := team.FromContext(ctx)
	if tm == nil {
		return tool.NewTextErrorResponse("no team available"), nil
	}

	_, recipient, _ := taskScopeFromContext(ctx)
	if recipient == "" {
		if team.IsLead(ctx) {
			recipient = "__lead__"
		} else {
			recipient = "unknown"
		}
	}

	msgs, err := tm.Mailbox.Read(ctx, recipient)
	if err != nil {
		return tool.NewTextErrorResponse(
			fmt.Sprintf("failed to read messages: %s", err.Error()),
		), nil
	}

	if len(msgs) == 0 {
		return tool.NewTextResponse("No new messages."), nil
	}

	return tool.NewJSONResponse(msgs), nil
}

type listTeammatesInput struct{}

type listTeammatesTool struct{}

func (t *listTeammatesTool) Info() tool.Info {
	return tool.NewInfo(
		"list_teammates",
		"List all teammates in the team and their current status.",
		listTeammatesInput{},
	)
}

func (t *listTeammatesTool) Run(
	ctx context.Context,
	_ tool.Call,
) (tool.Response, error) {
	tm := team.FromContext(ctx)
	if tm == nil {
		return tool.NewTextErrorResponse("no team available"), nil
	}

	members := tm.ListMembers()

	type memberSummary struct {
		Name   string            `json:"name"`
		Status team.MemberStatus `json:"status"`
		Task   string            `json:"task"`
	}

	summaries := make([]memberSummary, len(members))
	for i, m := range members {
		summaries[i] = memberSummary{
			Name:   m.Name,
			Status: m.Status,
			Task:   m.Task,
		}
	}

	return tool.NewJSONResponse(summaries), nil
}

type stopTeammateInput struct {
	Name string `json:"name" desc:"Name of the teammate to stop"`
}

type stopTeammateTool struct{}

func (t *stopTeammateTool) Info() tool.Info {
	return tool.NewInfo(
		"stop_teammate",
		"Stop a running teammate by name. The teammate's context will be cancelled.",
		stopTeammateInput{},
	)
}

func (t *stopTeammateTool) Run(
	ctx context.Context,
	params tool.Call,
) (tool.Response, error) {
	var input stopTeammateInput
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return tool.NewTextErrorResponse(
			"invalid parameters: " + err.Error(),
		), nil
	}

	if input.Name == "" {
		return tool.NewTextErrorResponse("name is required"), nil
	}

	tm := team.FromContext(ctx)
	if tm == nil {
		return tool.NewTextErrorResponse("no team available"), nil
	}

	if err := tm.StopMember(input.Name); err != nil {
		return tool.NewTextErrorResponse(
			fmt.Sprintf("failed to stop teammate: %s", err.Error()),
		), nil
	}

	return tool.NewTextResponse(
		fmt.Sprintf("Teammate %q has been stopped.", input.Name),
	), nil
}

type teamEventChanKey struct{}

func withTeamEventChan(
	ctx context.Context,
	ch chan<- ChatEvent,
) context.Context {
	return context.WithValue(ctx, teamEventChanKey{}, ch)
}

func teamEventChanFromContext(ctx context.Context) chan<- ChatEvent {
	ch, _ := ctx.Value(teamEventChanKey{}).(chan<- ChatEvent)
	return ch
}

type teamHooksKey struct{}

func withTeamHooks(ctx context.Context, hooks []Hooks) context.Context {
	return context.WithValue(ctx, teamHooksKey{}, hooks)
}

func teamHooksFromContext(ctx context.Context) []Hooks {
	h, _ := ctx.Value(teamHooksKey{}).([]Hooks)
	return h
}

func createTeamLeadTools(leadAgent *Agent) []tool.BaseTool {
	return []tool.BaseTool{
		&spawnTeammateTool{leadAgent: leadAgent},
		&stopTeammateTool{},
	}
}

func createTeamCommunicationTools() []tool.BaseTool {
	return []tool.BaseTool{
		&sendMessageTool{},
		&readMessagesTool{},
		&listTeammatesTool{},
	}
}
