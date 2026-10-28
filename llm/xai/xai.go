// Package xai provides LLM clients targeting xAI's Grok API.
//
// [NewLLM] constructs a client targeting xAI's Responses API, with typed
// support for the server-side built-in tools (web_search, x_search,
// code_execution) and structured citation metadata.
package xai

// DefaultBaseURL is the canonical xAI API endpoint.
const DefaultBaseURL = "https://api.x.ai/v1"
