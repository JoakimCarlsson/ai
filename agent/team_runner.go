package agent

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/joakimcarlsson/ai/agent/team"
	"github.com/joakimcarlsson/ai/types"
)

func spawnTeammate(
	ctx context.Context,
	t *team.Team,
	name string,
	a *Agent,
	task string,
	hooks []Hooks,
	eventChan chan<- ChatEvent,
	opts ...ChatOption,
) (string, error) {
	taskCtx, cancel := context.WithCancel(ctx)

	member, err := t.AddMember(name, task, cancel)
	if err != nil {
		cancel()
		return "", err
	}

	runOnTeammateJoin(ctx, hooks, TeammateEventContext{
		TeamName:   t.Name(),
		MemberID:   member.ID,
		MemberName: name,
		Task:       task,
	})

	if eventChan != nil {
		eventChan <- ChatEvent{
			Type:      types.EventTeammateSpawned,
			AgentName: name,
		}
	}

	go func() {
		defer t.FinishMember(name)
		defer func() {
			if r := recover(); r != nil {
				_ = debug.Stack()
				panicMsg := fmt.Sprintf("panic: %v", r)
				duration := time.Since(member.JoinedAt)

				t.CompleteMember(name, team.MemberFailed, "", panicMsg)

				runOnTeammateError(ctx, hooks, TeammateEventContext{
					TeamName:   t.Name(),
					MemberID:   member.ID,
					MemberName: name,
					Task:       task,
					Error:      fmt.Errorf("%s", panicMsg),
					Duration:   duration,
				})

				if eventChan != nil {
					eventChan <- ChatEvent{
						Type:      types.EventTeammateError,
						AgentName: name,
						Error:     fmt.Errorf("%s", panicMsg),
					}
				}
			}
		}()

		scopedCtx := withTaskScope(taskCtx, member.ID, name)
		scopedCtx = team.WithContext(scopedCtx, t)

		resp, runErr := runTaskStream(scopedCtx, a, task, opts...)
		duration := time.Since(member.JoinedAt)

		if taskCtx.Err() != nil {
			t.CompleteMember(
				name,
				team.MemberStopped,
				"",
				"teammate was stopped",
			)
			runOnTeammateLeave(ctx, hooks, TeammateEventContext{
				TeamName:   t.Name(),
				MemberID:   member.ID,
				MemberName: name,
				Task:       task,
				Error:      fmt.Errorf("teammate was stopped"),
				Duration:   duration,
			})
			return
		}

		if runErr != nil {
			t.CompleteMember(name, team.MemberFailed, "", runErr.Error())

			runOnTeammateError(ctx, hooks, TeammateEventContext{
				TeamName:   t.Name(),
				MemberID:   member.ID,
				MemberName: name,
				Task:       task,
				Error:      runErr,
				Duration:   duration,
			})

			if eventChan != nil {
				eventChan <- ChatEvent{
					Type:      types.EventTeammateError,
					AgentName: name,
					Error:     runErr,
				}
			}
			return
		}

		result := ""
		if resp != nil {
			result = resp.Content
		}
		t.CompleteMember(name, team.MemberCompleted, result, "")

		_ = t.Mailbox.Send(ctx, team.Message{
			From:    name,
			To:      "__lead__",
			Content: fmt.Sprintf("Teammate %q completed: %s", name, result),
		})

		runOnTeammateComplete(ctx, hooks, TeammateEventContext{
			TeamName:   t.Name(),
			MemberID:   member.ID,
			MemberName: name,
			Task:       task,
			Result:     result,
			Duration:   duration,
		})

		if eventChan != nil {
			eventChan <- ChatEvent{
				Type:      types.EventTeammateComplete,
				AgentName: name,
			}
		}
	}()

	return member.ID, nil
}
