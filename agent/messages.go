package agent

import (
	"context"
	"fmt"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/prompt"
	"github.com/joakimcarlsson/ai/tokens"
)

// BuildContextMessages returns the messages that would be sent to the LLM after applying
// the context strategy. This is useful for debugging and testing context management.
// WARNING: This method modifies the session by adding the user message.
func (a *Agent) BuildContextMessages(
	ctx context.Context,
	userMessage string,
) ([]message.Message, error) {
	return a.buildMessages(ctx, userMessage)
}

// PeekContextMessages returns what messages would be sent to the LLM without modifying state.
func (a *Agent) PeekContextMessages(
	ctx context.Context,
	userMessage string,
) ([]message.Message, error) {
	var messages []message.Message

	systemPrompt, err := a.resolveSystemPrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve system prompt: %w", err)
	}

	if systemPrompt != "" {
		sysMsg := message.NewSystemMessage(systemPrompt)
		sysMsg.Model = a.llm.Model().ID
		messages = append(messages, sysMsg)
	}

	if a.session != nil {
		sessionMessages, err := a.session.GetMessages(ctx, nil)
		if err != nil {
			return nil, err
		}
		messages = append(messages, sessionMessages...)
	}

	userMsg := message.NewUserMessage(userMessage)
	userMsg.Model = a.llm.Model().ID
	messages = append(messages, userMsg)

	if a.contextStrategy != nil {
		counter, err := tokens.NewCounter()
		if err != nil {
			return nil, err
		}

		maxTokens := a.maxContextTokens
		if maxTokens == 0 {
			reserveTokens := a.reserveTokens
			if reserveTokens == 0 {
				reserveTokens = 4096
			}
			maxTokens = a.llm.Model().ContextWindow - reserveTokens
		}

		result, err := a.contextStrategy.Fit(ctx, tokens.StrategyInput{
			Messages:     messages,
			SystemPrompt: systemPrompt,
			Tools:        a.getToolsWithContext(ctx),
			Counter:      counter,
			MaxTokens:    maxTokens,
		})
		if err != nil {
			return nil, err
		}

		messages = result.Messages
	}

	return messages, nil
}

func (a *Agent) resolveSystemPrompt(ctx context.Context) (string, error) {
	if a.instructionProvider != nil {
		return a.instructionProvider(ctx, a.state)
	}

	if a.systemPrompt == "" {
		return "", nil
	}

	return prompt.Process(a.systemPrompt, a.state)
}

func (a *Agent) buildMessages(
	ctx context.Context,
	userMessage string,
) ([]message.Message, error) {
	var messages []message.Message

	systemPrompt, err := a.resolveSystemPrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve system prompt: %w", err)
	}

	if a.memory != nil && a.memoryID != "" {
		memories, err := a.memory.Search(ctx, a.memoryID, userMessage, 5)
		if err == nil && len(memories) > 0 {
			var memoryContext string
			for _, m := range memories {
				memoryContext += "- " + m.Content + "\n"
			}
			systemPrompt = systemPrompt + "\n\nRelevant memories about this user:\n" + memoryContext
		}
	}

	var sessionMessages []message.Message
	if a.session != nil {
		var err error
		sessionMessages, err = a.session.GetMessages(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	if systemPrompt != "" {
		sysMsg := message.NewSystemMessage(systemPrompt)
		sysMsg.Model = a.llm.Model().ID
		messages = append(messages, sysMsg)

		if a.session != nil && len(sessionMessages) == 0 {
			if err := a.session.AddMessages(ctx, []message.Message{sysMsg}); err != nil {
				return nil, err
			}
		}
	}

	messages = append(messages, sessionMessages...)

	userMsg := message.NewUserMessage(userMessage)
	userMsg.Model = a.llm.Model().ID
	messages = append(messages, userMsg)

	if a.session != nil {
		if err := a.session.AddMessages(ctx, []message.Message{userMsg}); err != nil {
			return nil, err
		}
	}

	if a.contextStrategy != nil {
		counter, err := tokens.NewCounter()
		if err != nil {
			return nil, fmt.Errorf("failed to create token counter: %w", err)
		}

		maxTokens := a.maxContextTokens
		if maxTokens == 0 {
			reserveTokens := a.reserveTokens
			if reserveTokens == 0 {
				reserveTokens = 4096
			}
			maxTokens = a.llm.Model().ContextWindow - reserveTokens
		}

		result, err := a.contextStrategy.Fit(ctx, tokens.StrategyInput{
			Messages:     messages,
			SystemPrompt: systemPrompt,
			Tools:        a.getToolsWithContext(ctx),
			Counter:      counter,
			MaxTokens:    maxTokens,
		})
		if err != nil {
			return nil, fmt.Errorf("context strategy failed: %w", err)
		}

		messages = result.Messages

		if result.SessionUpdate != nil && a.session != nil &&
			len(result.SessionUpdate.AddMessages) > 0 {
			if err := a.session.AddMessages(ctx, result.SessionUpdate.AddMessages); err != nil {
				return nil, fmt.Errorf("failed to save session update: %w", err)
			}
		}
	}

	return messages, nil
}

func (a *Agent) buildContinueMessages(
	ctx context.Context,
) ([]message.Message, error) {
	var messages []message.Message

	systemPrompt, err := a.resolveSystemPrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve system prompt: %w", err)
	}

	var sessionMessages []message.Message
	if a.session != nil {
		sessionMessages, err = a.session.GetMessages(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	if systemPrompt != "" {
		sysMsg := message.NewSystemMessage(systemPrompt)
		sysMsg.Model = a.llm.Model().ID
		messages = append(messages, sysMsg)
	}

	messages = append(messages, sessionMessages...)

	if a.contextStrategy != nil {
		counter, err := tokens.NewCounter()
		if err != nil {
			return nil, fmt.Errorf("failed to create token counter: %w", err)
		}

		maxTokens := a.maxContextTokens
		if maxTokens == 0 {
			reserveTokens := a.reserveTokens
			if reserveTokens == 0 {
				reserveTokens = 4096
			}
			maxTokens = a.llm.Model().ContextWindow - reserveTokens
		}

		result, err := a.contextStrategy.Fit(ctx, tokens.StrategyInput{
			Messages:     messages,
			SystemPrompt: systemPrompt,
			Tools:        a.getToolsWithContext(ctx),
			Counter:      counter,
			MaxTokens:    maxTokens,
		})
		if err != nil {
			return nil, fmt.Errorf("context strategy failed: %w", err)
		}

		messages = result.Messages

		if result.SessionUpdate != nil && a.session != nil &&
			len(result.SessionUpdate.AddMessages) > 0 {
			if err := a.session.AddMessages(ctx, result.SessionUpdate.AddMessages); err != nil {
				return nil, fmt.Errorf("failed to save session update: %w", err)
			}
		}
	}

	return messages, nil
}
