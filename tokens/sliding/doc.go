// Package sliding provides a context management strategy that keeps the last N messages.
//
// The sliding window strategy keeps only the most recent messages, regardless of
// token count. This provides simple, predictable context management that's ideal
// for chatbots where only recent conversation matters.
//
// # Usage
//
// Basic usage with defaults (keeps last 10 messages):
//
//	agent.WithContextStrategy(sliding.Strategy(), 4096)
//
// Keep last 20 messages:
//
//	agent.WithContextStrategy(sliding.Strategy(sliding.KeepLast(20)), 4096)
//
// # Options
//
//   - KeepLast(n): Number of recent messages to retain. Default is 10.
package sliding
