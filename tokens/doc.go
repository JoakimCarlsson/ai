// Package tokens provides token counting and context management for AI conversations.
//
// This package implements a BPE (Byte Pair Encoding) tokenizer using the cl100k_base
// vocabulary, which is compatible with GPT-4, Claude, and most modern language models.
// It enables accurate token counting without API calls, allowing for efficient context
// window management.
//
// The package also provides context management strategies that automatically trim
// conversations when they exceed the model's context window. Three strategies are
// available:
//
//   - truncate: Removes oldest messages until the conversation fits
//   - sliding: Keeps only the last N messages
//   - summarize: Uses an LLM to compress older messages into a summary
//
// # Token Counting
//
// The TokenCounter interface provides methods for counting tokens in messages,
// system prompts, and tool definitions:
//
//	counter, err := tokens.NewCounter()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	count, err := counter.CountTokens(ctx, tokens.CountOptions{
//	    Messages:     messages,
//	    SystemPrompt: "You are helpful.",
//	    Tools:        tools,
//	})
//	fmt.Printf("Total tokens: %d\n", count.TotalTokens)
//
// # Context Strategies
//
// Strategies are used with the agent's WithContextStrategy option:
//
//	// Truncate oldest messages
//	agent.WithContextStrategy(truncate.Strategy(), 4096)
//
//	// Keep last 20 messages
//	agent.WithContextStrategy(sliding.Strategy(sliding.KeepLast(20)), 4096)
//
//	// Summarize older messages
//	agent.WithContextStrategy(summarize.Strategy(summaryLLM), 4096)
package tokens
