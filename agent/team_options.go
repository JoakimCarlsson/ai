package agent

import "github.com/joakimcarlsson/ai/agent/team"

// WithTeam configures the agent as a team lead with the given team configuration.
// The team is created with a default in-memory channel mailbox and task board.
func WithTeam(config team.Config) Option {
	return func(a *Agent) {
		a.team = team.New(config)
	}
}

// WithCoordinatorMode restricts the lead agent to only team management and
// communication tools, preventing direct tool execution.
func WithCoordinatorMode() Option {
	return func(a *Agent) {
		a.coordinatorMode = true
	}
}

// WithMailbox overrides the team's default mailbox implementation.
// Must be called after WithTeam.
func WithMailbox(mb team.Mailbox) Option {
	return func(a *Agent) {
		if a.team != nil {
			a.team.Mailbox = mb
		}
	}
}

// WithTeammateTemplates registers pre-configured agent templates by name.
// When spawn_teammate is called with a matching name, the template is used
// instead of dynamically creating an agent.
func WithTeammateTemplates(templates map[string]*Agent) Option {
	return func(a *Agent) {
		a.teammateTemplates = templates
	}
}
