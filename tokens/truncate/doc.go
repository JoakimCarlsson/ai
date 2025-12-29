// Package truncate provides a context management strategy that removes oldest messages.
//
// The truncate strategy removes messages from the beginning of the conversation
// until the total token count fits within the model's context window. This is
// useful when you want to keep the most recent context and don't mind losing
// older messages entirely.
//
// # Usage
//
// Basic usage with defaults:
//
//	agent.WithContextStrategy(truncate.Strategy(), 4096)
//
// With options:
//
//	agent.WithContextStrategy(truncate.Strategy(
//	    truncate.PreservePairs(),
//	    truncate.MinMessages(3),
//	), 4096)
//
// # Options
//
//   - PreservePairs(): When removing a user message, also remove the following
//     assistant response to keep conversations coherent.
//   - MinMessages(n): Never remove messages below this count, even if over the
//     token limit.
package truncate
