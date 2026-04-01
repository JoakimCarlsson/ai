package batch

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
)

// AnthropicOption configures Anthropic-specific batch client options.
type AnthropicOption func(*anthropicOptions)

type anthropicOptions struct{}

type anthropicClient struct {
	providerOptions clientOptions
	options         anthropicOptions
	client          anthropic.Client
}

func newAnthropicBatchClient(opts clientOptions) *anthropicClient {
	anthropicOpts := anthropicOptions{}
	for _, o := range opts.anthropicOptions {
		o(&anthropicOpts)
	}

	clientOpts := []option.RequestOption{}
	if opts.apiKey != "" {
		clientOpts = append(clientOpts, option.WithAPIKey(opts.apiKey))
	}

	return &anthropicClient{
		providerOptions: opts,
		options:         anthropicOpts,
		client:          anthropic.NewClient(clientOpts...),
	}
}

func (c *anthropicClient) executeBatch(
	ctx context.Context,
	requests []Request,
	opts clientOptions,
) (*Response, error) {
	for _, r := range requests {
		if r.Type != RequestTypeChat {
			return nil, fmt.Errorf(
				"batch: anthropic native batch only supports chat requests",
			)
		}
	}

	results := make([]Result, len(requests))
	idxMap := make(map[string]int, len(requests))
	for i, r := range requests {
		idxMap[r.ID] = i
		results[i] = Result{ID: r.ID, Index: i}
	}

	batchRequests := make(
		[]anthropic.MessageBatchNewParamsRequest,
		len(requests),
	)
	for i, req := range requests {
		msgs, system := convertMessagesToAnthropic(req.Messages)
		tools := convertToolsToAnthropic(req.Tools)

		params := anthropic.MessageBatchNewParamsRequestParams{
			MaxTokens: opts.maxTokens,
			Messages:  msgs,
			Model:     anthropic.Model(opts.model.APIModel),
			Tools:     tools,
		}

		if len(system) > 0 {
			systemBlocks := make(
				[]anthropic.TextBlockParam,
				len(system),
			)
			for j, s := range system {
				systemBlocks[j] = anthropic.TextBlockParam{Text: s}
			}
			params.System = systemBlocks
		}

		batchRequests[i] = anthropic.MessageBatchNewParamsRequest{
			CustomID: req.ID,
			Params:   params,
		}
	}

	if opts.progressCallback != nil {
		opts.progressCallback(Progress{
			Total:  len(requests),
			Status: "submitting",
		})
	}

	batch, err := c.client.Messages.Batches.New(
		ctx,
		anthropic.MessageBatchNewParams{
			Requests: batchRequests,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"batch: failed to create anthropic batch: %w",
			err,
		)
	}

	pollInterval := opts.pollInterval
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}

	batch, err = c.pollUntilDone(
		ctx,
		batch.ID,
		pollInterval,
		len(requests),
		opts,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"batch: anthropic batch polling failed: %w",
			err,
		)
	}

	if err := c.retrieveResults(
		ctx,
		batch.ID,
		results,
		idxMap,
	); err != nil {
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

	return &Response{
		Results:   results,
		Completed: completed,
		Failed:    failed,
		Total:     len(requests),
	}, nil
}

func (c *anthropicClient) pollUntilDone(
	ctx context.Context,
	batchID string,
	interval time.Duration,
	total int,
	opts clientOptions,
) (*anthropic.MessageBatch, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			batch, err := c.client.Messages.Batches.Get(
				ctx,
				batchID,
			)
			if err != nil {
				return nil, err
			}

			if opts.progressCallback != nil {
				opts.progressCallback(Progress{
					Total:     total,
					Completed: int(batch.RequestCounts.Succeeded),
					Failed: int(
						batch.RequestCounts.Errored +
							batch.RequestCounts.Canceled +
							batch.RequestCounts.Expired,
					),
					Status: "polling",
				})
			}

			switch batch.ProcessingStatus {
			case anthropic.MessageBatchProcessingStatusEnded:
				return batch, nil
			case anthropic.MessageBatchProcessingStatusCanceling:
				continue
			}
		}
	}
}

func (c *anthropicClient) retrieveResults(
	ctx context.Context,
	batchID string,
	results []Result,
	idxMap map[string]int,
) error {
	stream := c.client.Messages.Batches.ResultsStreaming(
		ctx,
		batchID,
	)
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
			results[idx].Err = fmt.Errorf(
				"%s",
				errored.Error.Error.Message,
			)
		case "canceled":
			results[idx].Err = fmt.Errorf("request was canceled")
		case "expired":
			results[idx].Err = fmt.Errorf("request expired")
		}
	}

	if err := stream.Err(); err != nil {
		return err
	}

	return nil
}

func (c *anthropicClient) executeBatchAsync(
	ctx context.Context,
	requests []Request,
	opts clientOptions,
) (<-chan Event, error) {
	ch := make(chan Event, 16)

	go func() {
		defer close(ch)

		wrappedOpts := opts
		origCallback := opts.progressCallback
		wrappedOpts.progressCallback = func(p Progress) {
			ch <- Event{Type: EventProgress, Progress: &p}
			if origCallback != nil {
				origCallback(p)
			}
		}

		resp, err := c.executeBatch(ctx, requests, wrappedOpts)
		if err != nil {
			ch <- Event{Type: EventError, Err: err}
			return
		}

		for i := range resp.Results {
			ch <- Event{
				Type:   EventItem,
				Result: &resp.Results[i],
			}
		}

		ch <- Event{
			Type: EventComplete,
			Progress: &Progress{
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
) ([]anthropic.MessageParam, []string) {
	var anthropicMsgs []anthropic.MessageParam
	var systemMsgs []string

	for _, msg := range msgs {
		switch msg.Role {
		case message.System:
			systemMsgs = append(systemMsgs, msg.Content().String())
		case message.User:
			content := anthropic.NewTextBlock(msg.Content().String())
			anthropicMsgs = append(
				anthropicMsgs,
				anthropic.NewUserMessage(content),
			)
		case message.Assistant:
			if msg.Content().String() != "" {
				content := anthropic.NewTextBlock(
					msg.Content().String(),
				)
				anthropicMsgs = append(
					anthropicMsgs,
					anthropic.NewAssistantMessage(content),
				)
			}
		case message.Tool:
			var results []anthropic.ContentBlockParamUnion
			for _, tr := range msg.ToolResults() {
				results = append(
					results,
					anthropic.NewToolResultBlock(
						tr.ToolCallID,
						tr.Content,
						tr.IsError,
					),
				)
			}
			anthropicMsgs = append(
				anthropicMsgs,
				anthropic.NewUserMessage(results...),
			)
		}
	}

	return anthropicMsgs, systemMsgs
}

func convertToolsToAnthropic(
	tools []tool.BaseTool,
) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	anthropicTools := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		info := t.Info()
		anthropicTools[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        info.Name,
				Description: anthropic.String(info.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: info.Parameters,
				},
			},
		}
	}
	return anthropicTools
}

func convertAnthropicMessage(msg anthropic.Message) *llm.Response {
	content := ""
	for _, block := range msg.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			content += text.Text
		}
	}

	var toolCalls []message.ToolCall
	for _, block := range msg.Content {
		if variant, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
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
