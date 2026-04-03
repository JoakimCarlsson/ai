// Package team provides multi-agent team coordination with peer-to-peer messaging.
//
// Teams enable concurrent peer collaboration where a lead agent spawns multiple
// teammate agents that run in parallel goroutines and communicate via async
// message passing. This is distinct from sub-agents (parent-child delegation)
// and handoffs (linear control transfer).
//
// # Core Components
//
//   - [Team]: Manages a roster of concurrently-running peer agents
//   - [Mailbox]: Async message passing between team members (pluggable backend)
//   - [TaskBoard]: Shared task list for decentralized work distribution
//
// # Built-in Implementations
//
//   - [NewChannelMailbox]: In-memory channel-based mailbox (default, zero-config)
//   - [NewTaskBoard]: In-memory thread-safe task board
//
// # Usage with Agent
//
//	lead := agent.New(llmClient,
//	    agent.WithSystemPrompt("You are a team lead..."),
//	    agent.WithTeam(team.Config{
//	        Name:    "research-team",
//	        MaxSize: 5,
//	    }),
//	    agent.WithCoordinatorMode(),
//	)
//
// The lead agent automatically receives team management tools (spawn_teammate,
// send_message, read_messages, list_teammates, stop_teammate) and task board
// tools (create_board_task, claim_board_task, complete_board_task, list_board_tasks).
//
// # Custom Mailbox
//
// Implement the [Mailbox] interface for distributed or persistent message backends.
package team
