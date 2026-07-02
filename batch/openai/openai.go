// Package openai provides an OpenAI native batch API implementation of [batch.Processor].
//
// OpenAI's Batch API submits a JSONL file of requests, polls for completion,
// then retrieves a JSONL file of responses. This package handles that lifecycle.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/joakimcarlsson/ai/batch"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tool"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Options configures the OpenAI batch processor.
type Options struct {
	apiKey           string
	model            model.Model
	embeddingModel   model.EmbeddingModel
	maxTokens        int64
	progressCallback batch.ProgressCallback
	pollInterval     time.Duration
	timeout          *time.Duration
	baseURL          string
	extraHeaders     map[string]string
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel sets the LLM model for chat completion batch requests.
func WithModel(m model.Model) Option { return func(o *Options) { o.model = m } }

// WithEmbeddingModel sets the embedding model for embedding batch requests.
func WithEmbeddingModel(m model.EmbeddingModel) Option {
	return func(o *Options) { o.embeddingModel = m }
}

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

// WithBaseURL sets a custom API endpoint for OpenAI-compatible services.
func WithBaseURL(
	baseURL string,
) Option {
	return func(o *Options) { o.baseURL = baseURL }
}

// WithExtraHeaders adds custom HTTP headers to batch API requests.
func WithExtraHeaders(headers map[string]string) Option {
	return func(o *Options) { o.extraHeaders = headers }
}

// Processor implements [batch.Processor] against the OpenAI Batch API.
type Processor struct {
	options Options
	client  openaisdk.Client
}

// NewProcessor constructs an OpenAI batch processor.
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
	if options.baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(options.baseURL))
	}
	for key, value := range options.extraHeaders {
		clientOpts = append(clientOpts, option.WithHeader(key, value))
	}

	return &Processor{
		options: options,
		client:  openaisdk.NewClient(clientOpts...),
	}
}

type requestLine struct {
	CustomID string          `json:"custom_id"`
	Method   string          `json:"method"`
	URL      string          `json:"url"`
	Body     json.RawMessage `json:"body"`
}

type responseLine struct {
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

// Process submits all requests via OpenAI's Batch API, polls until completion,
// then retrieves and parses the result file.
func (p *Processor) Process(
	ctx context.Context,
	requests []batch.Request,
) (*batch.Response, error) {
	if len(requests) == 0 {
		return &batch.Response{Results: []batch.Result{}, Total: 0}, nil
	}
	batch.AssignIDs(requests)

	chatRequests, embedRequests := batch.SplitByType(requests)

	results := make([]batch.Result, len(requests))
	idxMap := make(map[string]int, len(requests))
	for i, r := range requests {
		idxMap[r.ID] = i
		results[i] = batch.Result{ID: r.ID, Index: i}
	}

	if len(chatRequests) > 0 {
		if err := p.processBatch(
			ctx, chatRequests,
			openaisdk.BatchNewParamsEndpointV1ChatCompletions,
			results, idxMap,
		); err != nil {
			return nil, fmt.Errorf("batch: openai chat batch failed: %w", err)
		}
	}

	if len(embedRequests) > 0 {
		if err := p.processBatch(
			ctx, embedRequests,
			openaisdk.BatchNewParamsEndpointV1Embeddings,
			results, idxMap,
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

	return &batch.Response{
		Results:   results,
		Completed: completed,
		Failed:    failed,
		Total:     len(requests),
	}, nil
}

func (p *Processor) processBatch(
	ctx context.Context,
	requests []batch.Request,
	endpoint openaisdk.BatchNewParamsEndpoint,
	results []batch.Result,
	idxMap map[string]int,
) error {
	apiModel := p.options.model.APIModel
	if endpoint == openaisdk.BatchNewParamsEndpointV1Embeddings &&
		p.options.embeddingModel.APIModel != "" {
		apiModel = p.options.embeddingModel.APIModel
	}

	jsonlData, err := p.buildJSONL(requests, endpoint, apiModel)
	if err != nil {
		return fmt.Errorf("failed to build JSONL: %w", err)
	}

	if p.options.progressCallback != nil {
		p.options.progressCallback(batch.Progress{
			Total:  len(results),
			Status: "uploading",
		})
	}

	file, err := p.client.Files.New(ctx, openaisdk.FileNewParams{
		File:    bytes.NewReader(jsonlData),
		Purpose: openaisdk.FilePurposeBatch,
	})
	if err != nil {
		return fmt.Errorf("failed to upload batch file: %w", err)
	}

	job, err := p.client.Batches.New(ctx, openaisdk.BatchNewParams{
		InputFileID:      file.ID,
		Endpoint:         endpoint,
		CompletionWindow: openaisdk.BatchNewParamsCompletionWindow24h,
	})
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	job, err = p.pollUntilDone(ctx, job.ID, len(results))
	if err != nil {
		return fmt.Errorf("batch polling failed: %w", err)
	}

	if job.OutputFileID != "" {
		if err := p.parseOutputFile(
			ctx,
			job.OutputFileID,
			endpoint,
			results,
			idxMap,
		); err != nil {
			return fmt.Errorf("failed to parse output file: %w", err)
		}
	}

	if job.ErrorFileID != "" {
		p.parseErrorFile(ctx, job.ErrorFileID, results, idxMap)
	}

	return nil
}

func (p *Processor) buildJSONL(
	requests []batch.Request,
	endpoint openaisdk.BatchNewParamsEndpoint,
	apiModel string,
) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	for _, req := range requests {
		var body json.RawMessage
		var err error

		switch endpoint {
		case openaisdk.BatchNewParamsEndpointV1ChatCompletions:
			body, err = buildChatBody(req, apiModel)
		case openaisdk.BatchNewParamsEndpointV1Embeddings:
			body, err = buildEmbeddingBody(req, apiModel)
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

		line := requestLine{
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

func buildChatBody(
	req batch.Request,
	apiModel string,
) (json.RawMessage, error) {
	msgs := convertMessagesToOpenAI(req.Messages)
	tools := convertToolsToOpenAI(req.Tools)

	params := map[string]any{
		"model":    apiModel,
		"messages": msgs,
	}
	if len(tools) > 0 {
		params["tools"] = tools
	}

	return json.Marshal(params)
}

func buildEmbeddingBody(
	req batch.Request,
	apiModel string,
) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"model": apiModel,
		"input": req.Texts,
	})
}

func (p *Processor) pollUntilDone(
	ctx context.Context,
	batchID string,
	total int,
) (*openaisdk.Batch, error) {
	ticker := time.NewTicker(p.options.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			job, err := p.client.Batches.Get(ctx, batchID)
			if err != nil {
				return nil, err
			}

			if p.options.progressCallback != nil {
				p.options.progressCallback(batch.Progress{
					Total:     total,
					Completed: int(job.RequestCounts.Completed),
					Failed:    int(job.RequestCounts.Failed),
					Status:    "polling",
				})
			}

			switch job.Status {
			case openaisdk.BatchStatusCompleted:
				return job, nil
			case openaisdk.BatchStatusFailed:
				return job, fmt.Errorf("batch failed: %s", job.ID)
			case openaisdk.BatchStatusExpired:
				return job, fmt.Errorf("batch expired: %s", job.ID)
			case openaisdk.BatchStatusCancelled:
				return job, fmt.Errorf("batch cancelled: %s", job.ID)
			}
		}
	}
}

func (p *Processor) parseOutputFile(
	ctx context.Context,
	fileID string,
	endpoint openaisdk.BatchNewParamsEndpoint,
	results []batch.Result,
	idxMap map[string]int,
) error {
	resp, err := p.client.Files.Content(ctx, fileID)
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
		var line responseLine
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
		case openaisdk.BatchNewParamsEndpointV1ChatCompletions:
			results[idx].ChatResponse = parseChatCompletion(line.Response.Body)
		case openaisdk.BatchNewParamsEndpointV1Embeddings:
			results[idx].EmbedResponse = parseEmbeddingResponse(
				line.Response.Body,
			)
		}
	}

	return nil
}

func (p *Processor) parseErrorFile(
	ctx context.Context,
	fileID string,
	results []batch.Result,
	idxMap map[string]int,
) {
	resp, err := p.client.Files.Content(ctx, fileID)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var line responseLine
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
