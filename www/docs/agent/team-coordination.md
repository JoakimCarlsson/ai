# Team Coordination

Team coordination enables peer-to-peer multi-agent collaboration where a lead agent dynamically spawns teammates that communicate via asynchronous messaging and coordinate work through a shared task board.

## Setup

```go
import (
    "github.com/joakimcarlsson/ai/agent"
    "github.com/joakimcarlsson/ai/agent/team"
)

lead := agent.New(llmClient,
    agent.WithSystemPrompt("You are a team lead. Spawn teammates and coordinate their work."),
    agent.WithTeam(team.Config{
        Name:    "research-team",
        MaxSize: 5,
    }),
)

response, _ := lead.Chat(ctx, "Research AI, blockchain, and quantum computing in parallel")
```

The lead agent automatically receives team management tools (`spawn_teammate`, `stop_teammate`) and communication tools (`send_message`, `read_messages`, `list_teammates`), plus task board tools (`create_board_task`, `claim_board_task`, `complete_board_task`, `list_board_tasks`).

Spawned teammates receive communication and task board tools but cannot spawn or stop other teammates.

## How It Works

1. The lead agent decides when to spawn teammates using `spawn_teammate`
2. Each teammate runs concurrently in its own goroutine with an isolated context
3. Teammates communicate with each other and the lead via the mailbox
4. The shared task board provides structured work coordination
5. When a teammate completes, its result is automatically sent to the lead's inbox
6. The lead can stop any teammate at any time via `stop_teammate`

## Configuration Options

| Option | Description |
|--------|-------------|
| `WithTeam(config)` | Configures the agent as a team lead with the given team config |
| `WithCoordinatorMode()` | Restricts the lead to only team tools, preventing direct tool execution |
| `WithMailbox(mb)` | Overrides the default in-memory mailbox (must be called after `WithTeam`) |
| `WithTeammateTemplates(map)` | Registers pre-configured agent templates by name |

### team.Config

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Team name |
| `MaxSize` | `int` | Maximum concurrent active teammates (0 = unlimited) |

## Tools

### Lead-Only Tools

| Tool | Parameters | Description |
|------|-----------|-------------|
| `spawn_teammate` | `name`, `task`, `system_prompt?`, `max_turns?` | Spawn a new concurrent teammate |
| `stop_teammate` | `name` | Cancel a running teammate's context |

### Communication Tools (All Members)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `send_message` | `to`, `content` | Send a message to a teammate or `*` for broadcast |
| `read_messages` | — | Read and consume all unread inbox messages |
| `list_teammates` | — | List all teammates and their current status |

### Task Board Tools (All Members)

| Tool | Parameters | Description |
|------|-----------|-------------|
| `create_board_task` | `title` | Create a new open task on the shared board |
| `claim_board_task` | `task_id` | Claim an open task |
| `complete_board_task` | `task_id`, `result` | Mark a claimed task as completed with a result |
| `list_board_tasks` | — | List all tasks on the board |

## Member Lifecycle

Each teammate moves through these states:

| Status | Description |
|--------|-------------|
| `active` | Running and processing its task |
| `completed` | Finished successfully |
| `failed` | Encountered an error or panic |
| `stopped` | Cancelled by the lead via `stop_teammate` |

## Messaging

The mailbox provides asynchronous, fire-and-forget message passing between team members.

```go
lead := agent.New(llmClient,
    agent.WithSystemPrompt(`You are a team lead.
Spawn a researcher and a writer.
The researcher should send findings to the writer via send_message.
The writer should read_messages to get the research before writing.`),
    agent.WithTeam(team.Config{Name: "content-team", MaxSize: 3}),
)
```

Key behaviors:

- **Point-to-point**: Set `to` to a teammate's name
- **Broadcast**: Set `to` to `*` to reach all teammates except the sender
- **Consumed on read**: Messages are removed from the inbox after `read_messages`
- **Lead inbox**: The lead's recipient name is `__lead__` — teammate completion results are automatically sent here

## Task Board

The task board provides structured coordination where teammates can create, claim, and complete shared tasks.

```go
lead := agent.New(llmClient,
    agent.WithSystemPrompt(`You are a team lead.
Create board tasks for each research topic.
Spawn teammates that claim and complete tasks from the board.`),
    agent.WithTeam(team.Config{Name: "task-team"}),
)
```

Task lifecycle: **Open** → **Claimed** → **Completed**

- Any member can create tasks
- Only open tasks can be claimed
- Only the assignee can complete their claimed task

## Teammate Templates

Pre-configure teammate agents with specific tools and settings:

```go
researcher := agent.New(llmClient,
    agent.WithSystemPrompt("You are a research specialist."),
    agent.WithTools(&webSearchTool{}),
)

writer := agent.New(llmClient,
    agent.WithSystemPrompt("You are a content writer."),
)

lead := agent.New(llmClient,
    agent.WithSystemPrompt("Coordinate research and writing."),
    agent.WithTeam(team.Config{Name: "content-team"}),
    agent.WithTeammateTemplates(map[string]*agent.Agent{
        "researcher": researcher,
        "writer":     writer,
    }),
)
```

When `spawn_teammate` is called with a name matching a template, the pre-configured agent is used instead of dynamically creating one.

## Coordinator Mode

Restrict the lead to only team management and communication tools:

```go
lead := agent.New(llmClient,
    agent.WithSystemPrompt("You only coordinate. Delegate all work to teammates."),
    agent.WithTeam(team.Config{Name: "my-team"}),
    agent.WithCoordinatorMode(),
)
```

In coordinator mode, any non-team tools registered on the lead are filtered out. The lead can only spawn teammates, communicate, and manage the task board.

## Streaming Events

Team coordination emits streaming events during `ChatStream`:

| Event | Description |
|-------|-------------|
| `EventTeammateSpawned` | A new teammate was launched |
| `EventTeamMessage` | A message was sent between members |
| `EventTeammateComplete` | A teammate finished successfully |
| `EventTeammateError` | A teammate encountered an error |

```go
for event := range lead.ChatStream(ctx, "Research these topics") {
    switch event.Type {
    case types.EventTeammateSpawned:
        fmt.Printf("Spawned: %s\n", event.AgentName)
    case types.EventTeammateComplete:
        fmt.Printf("Completed: %s\n", event.AgentName)
    case types.EventTeammateError:
        fmt.Printf("Error in %s: %v\n", event.AgentName, event.Error)
    case types.EventComplete:
        fmt.Println(event.Response.Content)
    }
}
```

## Hooks

Team hooks provide observation-only callbacks for teammate lifecycle and messaging events:

| Hook | Fires | Context |
|------|-------|---------|
| `OnTeammateJoin` | When a teammate is spawned | `TeammateEventContext` |
| `OnTeammateLeave` | When a teammate leaves (stopped) | `TeammateEventContext` |
| `OnTeammateComplete` | When a teammate finishes successfully | `TeammateEventContext` |
| `OnTeammateError` | When a teammate encounters an error | `TeammateEventContext` |
| `OnTeamMessage` | When a message is sent between members | `TeamMessageContext` |

```go
lead := agent.New(llmClient,
    agent.WithTeam(team.Config{Name: "my-team"}),
    agent.WithHooks(agent.Hooks{
        OnTeammateJoin: func(_ context.Context, tc agent.TeammateEventContext) {
            log.Printf("Teammate %s joined team %s", tc.MemberName, tc.TeamName)
        },
        OnTeammateComplete: func(_ context.Context, tc agent.TeammateEventContext) {
            log.Printf("Teammate %s completed in %s: %s", tc.MemberName, tc.Duration, tc.Result)
        },
        OnTeamMessage: func(_ context.Context, mc agent.TeamMessageContext) {
            log.Printf("Message from %s to %s: %s", mc.Message.From, mc.Message.To, mc.Message.Content)
        },
    }),
)
```

Hooks set on the lead automatically propagate to spawned teammates that don't have their own hooks.

### TeammateEventContext

```go
type TeammateEventContext struct {
    TeamName   string
    MemberID   string
    MemberName string
    Task       string
    Result     string
    Error      error
    Duration   time.Duration
}
```

### TeamMessageContext

```go
type TeamMessageContext struct {
    TeamName string
    Message  team.Message
}
```

### team.Message

```go
type Message struct {
    ID        string
    From      string
    To        string
    Content   string
    Timestamp time.Time
}
```

## Comparison with Other Multi-Agent Patterns

| Pattern | Use Case |
|---------|----------|
| [Sub-Agents](sub-agents.md) | Hierarchical delegation — orchestrator calls child agents as tools |
| [Handoffs](handoffs.md) | Sequential transfer — one agent passes control to a peer |
| [Fan-Out](fan-out.md) | Parallel execution — distribute independent tasks to cloned workers |
| **Team Coordination** | Peer-to-peer collaboration — dynamic spawning, messaging, shared task board |
