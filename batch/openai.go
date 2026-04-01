package batch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/openai/openai-go"
)

type openaiNativeExecutor struct {
	client *openai.Client
}

type openaiRequestLine struct {
	CustomID string          `json:"custom_id"`
	Method   string          `json:"method"`
	URL      string          `json:"url"`
	Body     json.RawMessage `json:"body"`
}

type openaiResponseLine struct {
	ID       string `json:"id"`
	CustomID string `json:"custom_id"`
	Response struct {
		StatusCode int             `json:"status_code"`
		Body       json.RawMessage `json:"body"`
	} `json:"response"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *openaiNativeExecutor) execute(
	ctx context.Context,
	requests []Request,
	opts processorOptions,
) (*Response, error) {
	chatRequests, embedRequests := splitByType(requests)

	results := make([]Result, len(requests))
	idxMap := make(map[string]int, len(requests))
	for i, r := range requests {
		idxMap[r.ID] = i
		results[i] = Result{ID: r.ID, Index: i}
	}

	if len(chatRequests) > 0 {
		if err := e.processBatch(
			ctx, chatRequests, openai.BatchNewParamsEndpointV1ChatCompletions,
			results, idxMap, opts,
		); err != nil {
			return nil, fmt.Errorf("batch: openai chat batch failed: %w", err)
		}
	}

	if len(embedRequests) > 0 {
		if err := e.processBatch(
			ctx, embedRequests, openai.BatchNewParamsEndpointV1Embeddings,
			results, idxMap, opts,
		); err != nil {
			return nil, fmt.Errorf(
				"batch: openai embedding batch failed: %w",
				err,
			)
		}
	}

	completed, failed := 0, 0
	for _, r := range results {
		if r.Err != nil {
			failed++
		} else if r.ChatResponse != nil || r.EmbedResponse != nil {
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

func (e *openaiNativeExecutor) processBatch(
	ctx context.Context,
	requests []Request,
	endpoint openai.BatchNewParamsEndpoint,
	results []Result,
	idxMap map[string]int,
	opts processorOptions,
) error {
	jsonlData, err := e.buildJSONL(requests, endpoint)
	if err != nil {
		return fmt.Errorf("failed to build JSONL: %w", err)
	}

	if opts.progressCallback != nil {
		opts.progressCallback(Progress{
			Total:  len(results),
			Status: "uploading",
		})
	}

	file, err := e.client.Files.New(ctx, openai.FileNewParams{
		File:    bytes.NewReader(jsonlData),
		Purpose: openai.FilePurposeBatch,
	})
	if err != nil {
		return fmt.Errorf("failed to upload batch file: %w", err)
	}

	batch, err := e.client.Batches.New(ctx, openai.BatchNewParams{
		InputFileID:      file.ID,
		Endpoint:         endpoint,
		CompletionWindow: openai.BatchNewParamsCompletionWindow24h,
	})
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	pollInterval := opts.pollInterval
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}

	batch, err = e.pollUntilDone(
		ctx,
		batch.ID,
		pollInterval,
		len(results),
		opts,
	)
	if err != nil {
		return fmt.Errorf("batch polling failed: %w", err)
	}

	if batch.OutputFileID != "" {
		if err := e.parseOutputFile(ctx, batch.OutputFileID, endpoint, results, idxMap); err != nil {
			return fmt.Errorf("failed to parse output file: %w", err)
		}
	}

	if batch.ErrorFileID != "" {
		e.parseErrorFile(ctx, batch.ErrorFileID, results, idxMap)
	}

	return nil
}

func (e *openaiNativeExecutor) buildJSONL(
	requests []Request,
	endpoint openai.BatchNewParamsEndpoint,
) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	for _, req := range requests {
		var body json.RawMessage
		var err error

		switch endpoint {
		case openai.BatchNewParamsEndpointV1ChatCompletions:
			body, err = e.buildChatBody(req)
		case openai.BatchNewParamsEndpointV1Embeddings:
			body, err = e.buildEmbeddingBody(req)
		default:
			return nil, fmt.Errorf("unsupported endpoint: %s", endpoint)
		}

		if err != nil {
			return nil, fmt.Errorf(
				"failed to build request body for %s: %w",
				req.ID,
				err,
			)
		}

		line := openaiRequestLine{
			CustomID: req.ID,
			Method:   "POST",
			URL:      string(endpoint),
			Body:     body,
		}
		if err := enc.Encode(line); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (e *openaiNativeExecutor) buildChatBody(
	req Request,
) (json.RawMessage, error) {
	msgs := convertMessagesToOpenAI(req.Messages)
	tools := convertToolsToOpenAI(req.Tools)

	params := map[string]any{
		"messages": msgs,
	}
	if len(tools) > 0 {
		params["tools"] = tools
	}

	return json.Marshal(params)
}

func (e *openaiNativeExecutor) buildEmbeddingBody(
	req Request,
) (json.RawMessage, error) {
	params := map[string]any{
		"input": req.Texts,
	}
	return json.Marshal(params)
}

func (e *openaiNativeExecutor) pollUntilDone(
	ctx context.Context,
	batchID string,
	interval time.Duration,
	total int,
	opts processorOptions,
) (*openai.Batch, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			batch, err := e.client.Batches.Get(ctx, batchID)
			if err != nil {
				return nil, err
			}

			if opts.progressCallback != nil {
				opts.progressCallback(Progress{
					Total:     total,
					Completed: int(batch.RequestCounts.Completed),
					Failed:    int(batch.RequestCounts.Failed),
					Status:    "polling",
				})
			}

			switch batch.Status {
			case openai.BatchStatusCompleted:
				return batch, nil
			case openai.BatchStatusFailed:
				return batch, fmt.Errorf("batch failed: %s", batch.ID)
			case openai.BatchStatusExpired:
				return batch, fmt.Errorf("batch expired: %s", batch.ID)
			case openai.BatchStatusCancelled:
				return batch, fmt.Errorf("batch cancelled: %s", batch.ID)
			}
		}
	}
}

func (e *openaiNativeExecutor) parseOutputFile(
	ctx context.Context,
	fileID string,
	endpoint openai.BatchNewParamsEndpoint,
	results []Result,
	idxMap map[string]int,
) error {
	resp, err := e.client.Files.Content(ctx, fileID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var line openaiResponseLine
		if err := dec.Decode(&line); err != nil {
			continue
		}

		idx, ok := idxMap[line.CustomID]
		if !ok {
			continue
		}

		if line.Error != nil {
			results[idx].Err = fmt.Errorf(
				"%s: %s",
				line.Error.Code,
				line.Error.Message,
			)
			continue
		}

		if line.Response.StatusCode != 200 {
			results[idx].Err = fmt.Errorf(
				"request failed with status %d",
				line.Response.StatusCode,
			)
			continue
		}

		switch endpoint {
		case openai.BatchNewParamsEndpointV1ChatCompletions:
			results[idx].ChatResponse = parseChatCompletion(line.Response.Body)
		case openai.BatchNewParamsEndpointV1Embeddings:
			results[idx].EmbedResponse = parseEmbeddingResponse(
				line.Response.Body,
			)
		}
	}

	return nil
}

func (e *openaiNativeExecutor) parseErrorFile(
	ctx context.Context,
	fileID string,
	results []Result,
	idxMap map[string]int,
) {
	resp, err := e.client.Files.Content(ctx, fileID)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var line openaiResponseLine
		if err := dec.Decode(&line); err != nil {
			continue
		}

		idx, ok := idxMap[line.CustomID]
		if !ok {
			continue
		}

		if results[idx].Err == nil && line.Error != nil {
			results[idx].Err = fmt.Errorf(
				"%s: %s",
				line.Error.Code,
				line.Error.Message,
			)
		}
	}
}

func (e *openaiNativeExecutor) executeAsync(
	ctx context.Context,
	requests []Request,
	opts processorOptions,
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

		resp, err := e.execute(ctx, requests, wrappedOpts)
		if err != nil {
			ch <- Event{Type: EventError, Err: err}
			return
		}

		for i := range resp.Results {
			ch <- Event{Type: EventItem, Result: &resp.Results[i]}
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

func splitByType(requests []Request) (chat, embed []Request) {
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

func convertMessagesToOpenAI(msgs []message.Message) []map[string]any {
	var result []map[string]any
	for _, msg := range msgs {
		switch msg.Role {
		case message.System:
			result = append(result, map[string]any{
				"role":    "system",
				"content": msg.Content().String(),
			})
		case message.User:
			result = append(result, map[string]any{
				"role":    "user",
				"content": msg.Content().String(),
			})
		case message.Assistant:
			m := map[string]any{
				"role":    "assistant",
				"content": msg.Content().String(),
			}
			if len(msg.ToolCalls()) > 0 {
				var calls []map[string]any
				for _, tc := range msg.ToolCalls() {
					calls = append(calls, map[string]any{
						"id":   tc.ID,
						"type": "function",
						"function": map[string]any{
							"name":      tc.Name,
							"arguments": tc.Input,
						},
					})
				}
				m["tool_calls"] = calls
			}
			result = append(result, m)
		case message.Tool:
			for _, tr := range msg.ToolResults() {
				result = append(result, map[string]any{
					"role":         "tool",
					"tool_call_id": tr.ToolCallID,
					"content":      tr.Content,
				})
			}
		}
	}
	return result
}

func convertToolsToOpenAI(tools []tool.BaseTool) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	var result []map[string]any
	for _, t := range tools {
		info := t.Info()
		params := map[string]any{
			"type":       "object",
			"properties": info.Parameters,
		}
		if len(info.Required) > 0 {
			params["required"] = info.Required
		}
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        info.Name,
				"description": info.Description,
				"parameters":  params,
			},
		})
	}
	return result
}

func parseChatCompletion(body json.RawMessage) *llm.Response {
	var completion struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &completion); err != nil ||
		len(completion.Choices) == 0 {
		return nil
	}

	choice := completion.Choices[0]
	var toolCalls []message.ToolCall
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, message.ToolCall{
			ID:       tc.ID,
			Name:     tc.Function.Name,
			Input:    tc.Function.Arguments,
			Type:     "function",
			Finished: true,
		})
	}

	finishReason := message.FinishReasonUnknown
	switch choice.FinishReason {
	case "stop":
		finishReason = message.FinishReasonEndTurn
	case "length":
		finishReason = message.FinishReasonMaxTokens
	case "tool_calls":
		finishReason = message.FinishReasonToolUse
	}

	if len(toolCalls) > 0 {
		finishReason = message.FinishReasonToolUse
	}

	return &llm.Response{
		Content:   choice.Message.Content,
		ToolCalls: toolCalls,
		Usage: llm.TokenUsage{
			InputTokens:  completion.Usage.PromptTokens,
			OutputTokens: completion.Usage.CompletionTokens,
		},
		FinishReason: finishReason,
	}
}

func parseEmbeddingResponse(
	body json.RawMessage,
) *embeddings.EmbeddingResponse {
	var resp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			TotalTokens int64 `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}

	embs := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		embs[i] = d.Embedding
	}

	return &embeddings.EmbeddingResponse{
		Embeddings: embs,
		Usage: embeddings.EmbeddingUsage{
			TotalTokens: resp.Usage.TotalTokens,
		},
		Model: resp.Model,
	}
}
