// Package summarize provides a context management strategy that summarizes older messages.
//
// The summarize strategy uses an LLM to create a summary of older messages, then
// replaces them with that summary. This preserves the semantic meaning of the
// conversation while reducing token count. It's ideal for long-running assistants
// that need to remember what was discussed.
//
// # How It Works
//
//  1. Check if conversation exceeds token limit
//  2. Split messages: older ones to summarize, recent ones to keep verbatim
//  3. Send older messages to the LLM with a summarization prompt
//  4. Replace older messages with a single "summary" message
//  5. Return: summary message + recent messages
//
// # Usage
//
// Basic usage (keeps last 5 messages verbatim):
//
//	summaryLLM, _ := llm.NewLLM(model.ProviderOpenAI,
//	    llm.WithModel(model.OpenAIModels[model.GPT4oMini]),
//	)
//	agent.WithContextStrategy(summarize.Strategy(summaryLLM), 4096)
//
// Keep last 10 messages verbatim:
//
//	agent.WithContextStrategy(summarize.Strategy(summaryLLM, summarize.KeepRecent(10)), 4096)
//
// # Options
//
//   - KeepRecent(n): Number of recent messages to keep verbatim (not summarized).
//     Default is 5. These messages are preserved exactly as-is, while older
//     messages are compressed into a summary.
package summarize

