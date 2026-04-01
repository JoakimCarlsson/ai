package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/tool"
	"google.golang.org/genai"
)

// GeminiOption configures Gemini-specific batch client options.
type GeminiOption func(*geminiOptions)

type geminiOptions struct {
	backend genai.Backend
}

// WithGeminiBackend sets the Gemini backend (GeminiAPI or VertexAI).
func WithGeminiBackend(backend genai.Backend) GeminiOption {
	return func(o *geminiOptions) {
		o.backend = backend
	}
}

type geminiBatchClient struct {
	providerOptions clientOptions
	options         geminiOptions
	client          *genai.Client
	model           string
}

func newGeminiBatchClient(opts clientOptions) *geminiBatchClient {
	geminiOpts := geminiOptions{
		backend: genai.BackendGeminiAPI,
	}
	for _, o := range opts.geminiOptions {
		o(&geminiOpts)
	}

	client, _ := genai.NewClient(
		context.Background(),
		&genai.ClientConfig{
			APIKey:  opts.apiKey,
			Backend: geminiOpts.backend,
		},
	)

	apiModel := opts.model.APIModel
	if apiModel == "" && opts.embeddingModel.APIModel != "" {
		apiModel = opts.embeddingModel.APIModel
	}

	return &geminiBatchClient{
		providerOptions: opts,
		options:         geminiOpts,
		client:          client,
		model:           apiModel,
	}
}

func (c *geminiBatchClient) executeBatch(
	ctx context.Context,
	requests []Request,
	opts clientOptions,
) (*Response, error) {
	chatRequests, embedRequests := splitByType(requests)

	results := make([]Result, len(requests))
	for i, r := range requests {
		results[i] = Result{ID: r.ID, Index: i}
	}

	chatIdxMap := make(map[int]int)
	embedIdxMap := make(map[int]int)
	for i, r := range requests {
		switch r.Type {
		case RequestTypeChat:
			chatIdxMap[len(chatIdxMap)] = i
		case RequestTypeEmbedding:
			embedIdxMap[len(embedIdxMap)] = i
		}
	}

	if len(chatRequests) > 0 {
		if err := c.processChatBatch(
			ctx,
			chatRequests,
			results,
			chatIdxMap,
			opts,
		); err != nil {
			return nil, fmt.Errorf(
				"batch: gemini chat batch failed: %w",
				err,
			)
		}
	}

	if len(embedRequests) > 0 {
		if err := c.processEmbeddingBatch(
			ctx,
			embedRequests,
			results,
			embedIdxMap,
			opts,
		); err != nil {
			return nil, fmt.Errorf(
				"batch: gemini embedding batch failed: %w",
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

func (c *geminiBatchClient) processChatBatch(
	ctx context.Context,
	requests []Request,
	results []Result,
	idxMap map[int]int,
	opts clientOptions,
) error {
	inlined := make([]*genai.InlinedRequest, len(requests))
	for i, req := range requests {
		contents, system := convertMessagesToGemini(req.Messages)
		config := &genai.GenerateContentConfig{}
		if len(system) > 0 {
			config.SystemInstruction = &genai.Content{
				Parts: []*genai.Part{
					{Text: strings.Join(system, "\n\n")},
				},
			}
		}
		if len(req.Tools) > 0 {
			config.Tools = convertToolsToGemini(req.Tools)
		}

		inlined[i] = &genai.InlinedRequest{
			Model:    c.model,
			Contents: contents,
			Config:   config,
			Metadata: map[string]string{"custom_id": req.ID},
		}
	}

	if opts.progressCallback != nil {
		opts.progressCallback(
			Progress{Total: len(results), Status: "submitting"},
		)
	}

	job, err := c.client.Batches.Create(
		ctx,
		c.model,
		&genai.BatchJobSource{InlinedRequests: inlined},
		&genai.CreateBatchJobConfig{},
	)
	if err != nil {
		return fmt.Errorf("failed to create batch job: %w", err)
	}

	pollInterval := opts.pollInterval
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}

	job, err = c.pollUntilDone(
		ctx,
		job.Name,
		pollInterval,
		len(results),
		opts,
	)
	if err != nil {
		return err
	}

	if job.Dest != nil && len(job.Dest.InlinedResponses) > 0 {
		for i, resp := range job.Dest.InlinedResponses {
			globalIdx, ok := idxMap[i]
			if !ok {
				continue
			}

			if resp.Error != nil {
				results[globalIdx].Err = fmt.Errorf(
					"%s",
					resp.Error.Message,
				)
				continue
			}

			if resp.Response != nil {
				results[globalIdx].ChatResponse = convertGeminiResponse(
					resp.Response,
				)
			}
		}
	}

	return nil
}

func (c *geminiBatchClient) processEmbeddingBatch(
	ctx context.Context,
	requests []Request,
	results []Result,
	idxMap map[int]int,
	opts clientOptions,
) error {
	var allContents []*genai.Content
	contentToReq := make(map[int]int)
	idx := 0
	for reqI, req := range requests {
		for _, text := range req.Texts {
			allContents = append(allContents, &genai.Content{
				Parts: []*genai.Part{{Text: text}},
			})
			contentToReq[idx] = reqI
			idx++
		}
	}

	if opts.progressCallback != nil {
		opts.progressCallback(
			Progress{Total: len(results), Status: "submitting"},
		)
	}

	embedModel := c.model
	if opts.embeddingModel.APIModel != "" {
		embedModel = opts.embeddingModel.APIModel
	}

	job, err := c.client.Batches.CreateEmbeddings(
		ctx,
		&embedModel,
		&genai.EmbeddingsBatchJobSource{
			InlinedRequests: &genai.EmbedContentBatch{
				Contents: allContents,
			},
		},
		&genai.CreateEmbeddingsBatchJobConfig{},
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create embedding batch job: %w",
			err,
		)
	}

	pollInterval := opts.pollInterval
	if pollInterval == 0 {
		pollInterval = 30 * time.Second
	}

	job, err = c.pollUntilDone(
		ctx,
		job.Name,
		pollInterval,
		len(results),
		opts,
	)
	if err != nil {
		return err
	}

	if job.Dest != nil &&
		len(job.Dest.InlinedEmbedContentResponses) > 0 {
		reqEmbeddings := make(map[int][][]float32)
		reqTokens := make(map[int]int64)

		for i, resp := range job.Dest.InlinedEmbedContentResponses {
			reqIdx := contentToReq[i]

			if resp.Error != nil {
				globalIdx := idxMap[reqIdx]
				results[globalIdx].Err = fmt.Errorf(
					"%s",
					resp.Error.Message,
				)
				continue
			}

			if resp.Response != nil &&
				resp.Response.Embedding != nil {
				reqEmbeddings[reqIdx] = append(
					reqEmbeddings[reqIdx],
					resp.Response.Embedding.Values,
				)
				reqTokens[reqIdx] += resp.Response.TokenCount
			}
		}

		for reqIdx, embs := range reqEmbeddings {
			globalIdx := idxMap[reqIdx]
			if results[globalIdx].Err != nil {
				continue
			}
			results[globalIdx].EmbedResponse = &embeddings.EmbeddingResponse{
				Embeddings: embs,
				Usage: embeddings.EmbeddingUsage{
					TotalTokens: reqTokens[reqIdx],
				},
			}
		}
	}

	return nil
}

func (c *geminiBatchClient) pollUntilDone(
	ctx context.Context,
	jobName string,
	interval time.Duration,
	total int,
	opts clientOptions,
) (*genai.BatchJob, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			job, err := c.client.Batches.Get(ctx, jobName, nil)
			if err != nil {
				return nil, err
			}

			if opts.progressCallback != nil {
				completed := 0
				failed := 0
				if job.CompletionStats != nil {
					completed = int(
						job.CompletionStats.SuccessfulCount,
					)
					failed = int(
						job.CompletionStats.FailedCount,
					)
				}
				opts.progressCallback(Progress{
					Total:     total,
					Completed: completed,
					Failed:    failed,
					Status:    "polling",
				})
			}

			switch job.State {
			case genai.JobStateSucceeded,
				genai.JobStatePartiallySucceeded:
				return job, nil
			case genai.JobStateFailed:
				msg := "batch job failed"
				if job.Error != nil {
					msg = job.Error.Message
				}
				return nil, fmt.Errorf("%s", msg)
			case genai.JobStateCancelled:
				return nil, fmt.Errorf("batch job cancelled")
			case genai.JobStateExpired:
				return nil, fmt.Errorf("batch job expired")
			}
		}
	}
}

func (c *geminiBatchClient) executeBatchAsync(
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

func convertMessagesToGemini(
	msgs []message.Message,
) ([]*genai.Content, []string) {
	var contents []*genai.Content
	var system []string

	for _, msg := range msgs {
		switch msg.Role {
		case message.System:
			system = append(system, msg.Content().String())
		case message.User:
			contents = append(contents, &genai.Content{
				Role: "user",
				Parts: []*genai.Part{
					{Text: msg.Content().String()},
				},
			})
		case message.Assistant:
			parts := []*genai.Part{}
			if msg.Content().String() != "" {
				parts = append(
					parts,
					&genai.Part{Text: msg.Content().String()},
				)
			}
			for _, tc := range msg.ToolCalls() {
				var args map[string]any
				json.Unmarshal([]byte(tc.Input), &args)
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: tc.Name,
						Args: args,
					},
				})
			}
			contents = append(contents, &genai.Content{
				Role:  "model",
				Parts: parts,
			})
		case message.Tool:
			for _, tr := range msg.ToolResults() {
				var respData map[string]any
				if err := json.Unmarshal(
					[]byte(tr.Content),
					&respData,
				); err != nil {
					respData = map[string]any{
						"result": tr.Content,
					}
				}
				contents = append(contents, &genai.Content{
					Role: "user",
					Parts: []*genai.Part{{
						FunctionResponse: &genai.FunctionResponse{
							Name:     tr.ToolCallID,
							Response: respData,
						},
					}},
				})
			}
		}
	}

	return contents, system
}

func convertToolsToGemini(tools []tool.BaseTool) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}

	var declarations []*genai.FunctionDeclaration
	for _, t := range tools {
		info := t.Info()
		declarations = append(
			declarations,
			&genai.FunctionDeclaration{
				Name:        info.Name,
				Description: info.Description,
				Parameters: convertToGenaiSchema(
					info.Parameters,
					info.Required,
				),
			},
		)
	}

	return []*genai.Tool{{FunctionDeclarations: declarations}}
}

func convertToGenaiSchema(
	properties map[string]any,
	required []string,
) *genai.Schema {
	schema := &genai.Schema{
		Type:       genai.TypeObject,
		Properties: make(map[string]*genai.Schema),
		Required:   required,
	}

	for key, val := range properties {
		if propMap, ok := val.(map[string]any); ok {
			propSchema := &genai.Schema{}
			if t, ok := propMap["type"].(string); ok {
				switch t {
				case "string":
					propSchema.Type = genai.TypeString
				case "number":
					propSchema.Type = genai.TypeNumber
				case "integer":
					propSchema.Type = genai.TypeInteger
				case "boolean":
					propSchema.Type = genai.TypeBoolean
				case "array":
					propSchema.Type = genai.TypeArray
				case "object":
					propSchema.Type = genai.TypeObject
				}
			}
			if desc, ok := propMap["description"].(string); ok {
				propSchema.Description = desc
			}
			schema.Properties[key] = propSchema
		}
	}

	return schema
}

func convertGeminiResponse(
	resp *genai.GenerateContentResponse,
) *llm.Response {
	content := ""
	var toolCalls []message.ToolCall

	if len(resp.Candidates) > 0 &&
		resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			switch {
			case part.Text != "":
				content = part.Text
			case part.FunctionCall != nil:
				id := "call_" + uuid.New().String()
				args, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, message.ToolCall{
					ID:       id,
					Name:     part.FunctionCall.Name,
					Input:    string(args),
					Type:     "function",
					Finished: true,
				})
			}
		}
	}

	finishReason := message.FinishReasonEndTurn
	if len(resp.Candidates) > 0 {
		switch resp.Candidates[0].FinishReason {
		case genai.FinishReasonStop:
			finishReason = message.FinishReasonEndTurn
		case genai.FinishReasonMaxTokens:
			finishReason = message.FinishReasonMaxTokens
		default:
			finishReason = message.FinishReasonUnknown
		}
	}
	if len(toolCalls) > 0 {
		finishReason = message.FinishReasonToolUse
	}

	usage := llm.TokenUsage{}
	if resp.UsageMetadata != nil {
		usage = llm.TokenUsage{
			InputTokens: int64(
				resp.UsageMetadata.PromptTokenCount,
			),
			OutputTokens: int64(
				resp.UsageMetadata.CandidatesTokenCount,
			),
			CacheReadTokens: int64(
				resp.UsageMetadata.CachedContentTokenCount,
			),
		}
	}

	return &llm.Response{
		Content:      content,
		ToolCalls:    toolCalls,
		Usage:        usage,
		FinishReason: finishReason,
	}
}
