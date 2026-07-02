// Package batch provides async batch processing for sending bulk LLM, embedding,
// and other API requests efficiently using provider batch APIs where available,
// with a fallback concurrent execution strategy.
//
// This package defines the [Processor] interface and the data types that flow
// through it. Concrete vendor implementations live in subpackages
// (batch/openai, batch/anthropic, batch/gemini for native batch APIs;
// batch/concurrent for the fallback). Each subpackage exports its own
// NewProcessor constructor.
//
// Example usage:
//
//	import (
//		"github.com/joakimcarlsson/ai/batch"
//		batchopenai "github.com/joakimcarlsson/ai/batch/openai"
//	)
//
//	proc := batchopenai.NewProcessor(
//		batchopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
//		batchopenai.WithModel(model.OpenAIModels[model.GPT4o]),
//		batchopenai.WithMaxTokens(4096),
//	)
//
//	resp, err := proc.Process(ctx, requests)
package batch

import (
	"context"
	"errors"
	"strconv"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tool"
)

// ErrNoLLMClient is returned when a chat request is submitted without an LLM client.
var ErrNoLLMClient = errors.New("batch: no LLM client configured")

// ErrNoEmbeddingClient is returned when an embedding request is submitted without an embedding client.
var ErrNoEmbeddingClient = errors.New("batch: no embedding client configured")

// RequestType identifies whether a batch request is a chat completion or embedding.
type RequestType int

// Request types.
const (
	RequestTypeChat RequestType = iota
	RequestTypeEmbedding
)

// Request represents a single item in a batch.
type Request struct {
	ID       string
	Type     RequestType
	Messages []message.Message
	Tools    []tool.BaseTool
	Texts    []string
}

// Result holds the outcome of a single batch request.
type Result struct {
	ID            string
	Index         int
	ChatResponse  *llm.Response
	EmbedResponse *embeddings.EmbeddingResponse
	Err           error
}

// Response contains the aggregated results of a batch operation.
type Response struct {
	Results   []Result
	Completed int
	Failed    int
	Total     int
}

// Processor submits and manages batch requests.
type Processor interface {
	Process(ctx context.Context, requests []Request) (*Response, error)
	ProcessAsync(ctx context.Context, requests []Request) (<-chan Event, error)
}

// EventType identifies the kind of event emitted during batch processing.
type EventType string

// Event types.
const (
	EventProgress EventType = "progress"
	EventItem     EventType = "item"
	EventComplete EventType = "complete"
	EventError    EventType = "error"
)

// Progress tracks the current state of a batch operation.
type Progress struct {
	Total     int
	Completed int
	Failed    int
	Status    string
}

// ProgressCallback is invoked with progress updates during batch processing.
type ProgressCallback func(Progress)

// Event represents a single event emitted during async batch processing.
type Event struct {
	Type     EventType
	Progress *Progress
	Result   *Result
	Err      error
}

// AssignIDs fills in any blank Request IDs with a positional default.
// Vendor implementations call this before submitting.
func AssignIDs(requests []Request) {
	for i := range requests {
		if requests[i].ID == "" {
			requests[i].ID = "req_" + strconv.Itoa(i)
		}
	}
}

// SplitByType separates a slice of Requests into chat and embedding sub-slices.
// Vendor implementations use this when their native batch APIs require
// per-endpoint submission.
func SplitByType(requests []Request) (chat, embed []Request) {
	for _, r := range requests {
		switch r.Type {
		case RequestTypeChat:
			chat = append(chat, r)
		case RequestTypeEmbedding:
			embed = append(embed, r)
		}
	}
	return
}
