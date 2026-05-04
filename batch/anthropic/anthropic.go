// Package anthropic provides an Anthropic native batch API implementation of [batch.Processor].
//
// Anthropic's Message Batches API submits a list of message requests, polls
// until the batch is complete, then streams results.
package anthropic

import (
	"context"
	"fmt"
	"time"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joakimcarlsson/ai/batch"
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tool"
)

// Options configures the Anthropic batch processor.
type Options struct {
	apiKey           string
	model            model.Model
	maxTokens        int64
	progressCallback batch.ProgressCallback
	pollInterval     time.Duration
	timeout          *time.Duration
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel sets the LLM model.
func WithModel(m model.Model) Option { return func(o *Options) { o.model = m } }

// WithMaxTokens sets the maximum number of tokens to generate per request.
func WithMaxTokens(
	maxTokens int64,
) Option {
	return func(o *Options) { o.maxTokens = maxTokens }
}

// WithProgressCallback sets a callback invoked with progress updates.
func WithProgressCallback(fn batch.ProgressCallback) Option {
	return func(o *Options) { o.progressCallback = fn }
}

// WithPollInterval sets the polling interval for the native batch API.
func WithPollInterval(
	d time.Duration,
) Option {
	return func(o *Options) { o.pollInterval = d }
}

// WithTimeout sets the maximum duration for batch requests.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// Processor implements [batch.Processor] against the Anthropic Message Batches API.
type Processor struct {
	options Options
	client  anthropicsdk.Client
}

// NewProcessor constructs an Anthropic batch processor.
func NewProcessor(opts ...Option) batch.Processor {
	options := Options{
		pollInterval: 30 * time.Second,
		maxTokens:    4096,
	}
	for _, o := range opts {
		o(&options)
	}

	clientOpts := []option.RequestOption{}
	if options.apiKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(options.apiKey))
	}

	return &Processor{
		options: options,
		client:  anthropicsdk.NewClient(clientOpts...),
	}
}

// Process submits all chat requests via Anthropic's Batches API.
// Embedding requests are not supported by Anthropic's batch API.
func (p *Processor) Process(
	ctx context.Context,
	requests []batch.Request,
) (*batch.Response, error) {
	if len(requests) == 0 {
		return &batch.Response{Results: []batch.Result{}, Total: 0}, nil
	}
	batch.AssignIDs(requests)

	for _, r := range requests {
		if r.Type != batch.RequestTypeChat {
			return nil, fmt.Errorf(
				"batch: anthropic native batch only supports chat requests",
			)
		}
	}

	results := make([]batch.Result, len(requests))
	idxMap := make(map[string]int, len(requests))
	for i, r := range requests {
		idxMap[r.ID] = i
		results[i] = batch.Result{ID: r.ID, Index: i}
	}

	batchRequests := make(
		[]anthropicsdk.MessageBatchNewParamsRequest,
		len(requests),
	)
	for i, req := range requests {
		msgs, system := convertMessagesToAnthropic(req.Messages)
		tools := convertToolsToAnthropic(req.Tools)

		params := anthropicsdk.MessageBatchNewParamsRequestParams{
			MaxTokens: p.options.maxTokens,
			Messages:  msgs,
			Model:     anthropicsdk.Model(p.options.model.APIModel),
			Tools:     tools,
		}

		if len(system) > 0 {
			systemBlocks := make([]anthropicsdk.TextBlockParam, len(system))
			for j, s := range system {
				systemBlocks[j] = anthropicsdk.TextBlockParam{Text: s}
			}
			params.System = systemBlocks
		}

		batchRequests[i] = anthropicsdk.MessageBatchNewParamsRequest{
			CustomID: req.ID,
			Params:   params,
		}
	}

	if p.options.progressCallback != nil {
		p.options.progressCallback(batch.Progress{
			Total:  len(requests),
			Status: "submitting",
		})
	}

	job, err := p.client.Messages.Batches.New(
		ctx,
		anthropicsdk.MessageBatchNewParams{
			Requests: batchRequests,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"batch: failed to create anthropic batch: %w",
			err,
		)
	}

	job, err = p.pollUntilDone(ctx, job.ID, len(requests))
	if err != nil {
		return nil, fmt.Errorf("batch: anthropic batch polling failed: %w", err)
	}

	if err := p.retrieveResults(ctx, job.ID, results, idxMap); err != nil {
		return nil, fmt.Errorf(
			"batch: failed to retrieve anthropic results: %w",
			err,
		)
	}

	completed, failed := 0, 0
	for _, r := range results {
		if r.Err != nil {
			failed++
		} else if r.ChatResponse != nil {
			completed++
		}
	}

	return &batch.Response{
		Results:   results,
		Completed: completed,
		Failed:    failed,
		Total:     len(requests),
	}, nil
}

func (p *Processor) pollUntilDone(
	ctx context.Context,
	batchID string,
	total int,
) (*anthropicsdk.MessageBatch, error) {
	ticker := time.NewTicker(p.options.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			job, err := p.client.Messages.Batches.Get(ctx, batchID)
			if err != nil {
				return nil, err
			}

			if p.options.progressCallback != nil {
				p.options.progressCallback(batch.Progress{
					Total:     total,
					Completed: int(job.RequestCounts.Succeeded),
					Failed: int(
						job.RequestCounts.Errored +
							job.RequestCounts.Canceled +
							job.RequestCounts.Expired,
					),
					Status: "polling",
				})
			}

			switch job.ProcessingStatus {
			case anthropicsdk.MessageBatchProcessingStatusEnded:
				return job, nil
			case anthropicsdk.MessageBatchProcessingStatusCanceling:
				continue
			}
		}
	}
}

func (p *Processor) retrieveResults(
	ctx context.Context,
	batchID string,
	results []batch.Result,
	idxMap map[string]int,
) error {
	stream := p.client.Messages.Batches.ResultsStreaming(ctx, batchID)
	defer stream.Close()

	for stream.Next() {
		entry := stream.Current()

		idx, ok := idxMap[entry.CustomID]
		if !ok {
			continue
		}

		switch entry.Result.Type {
		case "succeeded":
			succeeded := entry.Result.AsSucceeded()
			results[idx].ChatResponse = convertAnthropicMessage(
				succeeded.Message,
			)
		case "errored":
			errored := entry.Result.AsErrored()
			results[idx].Err = fmt.Errorf("%s", errored.Error.Error.Message)
		case "canceled":
			results[idx].Err = fmt.Errorf("request was canceled")
		case "expired":
			results[idx].Err = fmt.Errorf("request expired")
		}
	}

	return stream.Err()
}

// ProcessAsync wraps Process with an event channel.
func (p *Processor) ProcessAsync(
	ctx context.Context,
	requests []batch.Request,
) (<-chan batch.Event, error) {
	ch := make(chan batch.Event, 16)

	go func() {
		defer close(ch)

		origCallback := p.options.progressCallback
		p.options.progressCallback = func(prog batch.Progress) {
			ch <- batch.Event{Type: batch.EventProgress, Progress: &prog}
			if origCallback != nil {
				origCallback(prog)
			}
		}
		defer func() { p.options.progressCallback = origCallback }()

		resp, err := p.Process(ctx, requests)
		if err != nil {
			ch <- batch.Event{Type: batch.EventError, Err: err}
			return
		}

		for i := range resp.Results {
			ch <- batch.Event{Type: batch.EventItem, Result: &resp.Results[i]}
		}

		ch <- batch.Event{
			Type: batch.EventComplete,
			Progress: &batch.Progress{
				Total:     resp.Total,
				Completed: resp.Completed,
				Failed:    resp.Failed,
				Status:    "complete",
			},
		}
	}()

	return ch, nil
}

func convertMessagesToAnthropic(
	msgs []message.Message,
) ([]anthropicsdk.MessageParam, []string) {
	var anthropicMsgs []anthropicsdk.MessageParam
	var systemMsgs []string

	for _, msg := range msgs {
		switch msg.Role {
		case message.System:
			systemMsgs = append(systemMsgs, msg.Content().String())
		case message.User:
			content := anthropicsdk.NewTextBlock(msg.Content().String())
			anthropicMsgs = append(
				anthropicMsgs,
				anthropicsdk.NewUserMessage(content),
			)
		case message.Assistant:
			if msg.Content().String() != "" {
				content := anthropicsdk.NewTextBlock(msg.Content().String())
				anthropicMsgs = append(
					anthropicMsgs,
					anthropicsdk.NewAssistantMessage(content),
				)
			}
		case message.Tool:
			var results []anthropicsdk.ContentBlockParamUnion
			for _, tr := range msg.ToolResults() {
				results = append(results, anthropicsdk.NewToolResultBlock(
					tr.ToolCallID, tr.Content, tr.IsError,
				))
			}
			anthropicMsgs = append(
				anthropicMsgs,
				anthropicsdk.NewUserMessage(results...),
			)
		}
	}

	return anthropicMsgs, systemMsgs
}

func convertToolsToAnthropic(
	tools []tool.BaseTool,
) []anthropicsdk.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]anthropicsdk.ToolUnionParam, len(tools))
	for i, t := range tools {
		info := t.Info()
		out[i] = anthropicsdk.ToolUnionParam{
			OfTool: &anthropicsdk.ToolParam{
				Name:        info.Name,
				Description: anthropicsdk.String(info.Description),
				InputSchema: anthropicsdk.ToolInputSchemaParam{
					Properties: info.Parameters,
				},
			},
		}
	}
	return out
}

func convertAnthropicMessage(msg anthropicsdk.Message) *llm.Response {
	content := ""
	for _, block := range msg.Content {
		if text, ok := block.AsAny().(anthropicsdk.TextBlock); ok {
			content += text.Text
		}
	}

	var toolCalls []message.ToolCall
	for _, block := range msg.Content {
		if variant, ok := block.AsAny().(anthropicsdk.ToolUseBlock); ok {
			toolCalls = append(toolCalls, message.ToolCall{
				ID:       variant.ID,
				Name:     variant.Name,
				Input:    string(variant.Input),
				Type:     string(variant.Type),
				Finished: true,
			})
		}
	}

	finishReason := message.FinishReasonUnknown
	switch string(msg.StopReason) {
	case "end_turn":
		finishReason = message.FinishReasonEndTurn
	case "max_tokens":
		finishReason = message.FinishReasonMaxTokens
	case "tool_use":
		finishReason = message.FinishReasonToolUse
	}
	if len(toolCalls) > 0 {
		finishReason = message.FinishReasonToolUse
	}

	return &llm.Response{
		Content:   content,
		ToolCalls: toolCalls,
		Usage: llm.TokenUsage{
			InputTokens:         msg.Usage.InputTokens,
			OutputTokens:        msg.Usage.OutputTokens,
			CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
			CacheReadTokens:     msg.Usage.CacheReadInputTokens,
		},
		FinishReason: finishReason,
	}
}
