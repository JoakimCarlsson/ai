// Package gemini provides a Google Gemini native batch API implementation of [batch.Processor].
//
// Gemini's batch API supports both chat (GenerateContent) and embedding
// (EmbedContent) jobs via inlined requests; this package handles the lifecycle.
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joakimcarlsson/ai/batch"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/tool"
	"google.golang.org/genai"
)

// Options configures the Gemini batch processor.
type Options struct {
	apiKey           string
	model            model.Model
	embeddingModel   model.EmbeddingModel
	maxTokens        int64
	progressCallback batch.ProgressCallback
	pollInterval     time.Duration
	timeout          *time.Duration
	backend          genai.Backend
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

// WithEmbeddingModel sets the embedding model.
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

// WithBackend sets the Gemini backend (GeminiAPI or VertexAI).
func WithBackend(
	backend genai.Backend,
) Option {
	return func(o *Options) { o.backend = backend }
}

// Processor implements [batch.Processor] against the Gemini batch API.
type Processor struct {
	options Options
	client  *genai.Client
	model   string
}

// NewProcessor constructs a Gemini batch processor.
func NewProcessor(opts ...Option) batch.Processor {
	options := Options{
		pollInterval: 30 * time.Second,
		maxTokens:    4096,
		backend:      genai.BackendGeminiAPI,
	}
	for _, o := range opts {
		o(&options)
	}

	client, _ := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  options.apiKey,
		Backend: options.backend,
	})

	apiModel := options.model.APIModel
	if apiModel == "" && options.embeddingModel.APIModel != "" {
		apiModel = options.embeddingModel.APIModel
	}

	return &Processor{
		options: options,
		client:  client,
		model:   apiModel,
	}
}

// Process submits all requests via Gemini's batch API.
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
	for i, r := range requests {
		results[i] = batch.Result{ID: r.ID, Index: i}
	}

	chatIdxMap := make(map[int]int)
	embedIdxMap := make(map[int]int)
	for i, r := range requests {
		switch r.Type {
		case batch.RequestTypeChat:
			chatIdxMap[len(chatIdxMap)] = i
		case batch.RequestTypeEmbedding:
			embedIdxMap[len(embedIdxMap)] = i
		}
	}

	if len(chatRequests) > 0 {
		if err := p.processChatBatch(
			ctx,
			chatRequests,
			results,
			chatIdxMap,
		); err != nil {
			return nil, fmt.Errorf("batch: gemini chat batch failed: %w", err)
		}
	}

	if len(embedRequests) > 0 {
		if err := p.processEmbeddingBatch(
			ctx,
			embedRequests,
			results,
			embedIdxMap,
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

	return &batch.Response{
		Results:   results,
		Completed: completed,
		Failed:    failed,
		Total:     len(requests),
	}, nil
}

func (p *Processor) processChatBatch(
	ctx context.Context,
	requests []batch.Request,
	results []batch.Result,
	idxMap map[int]int,
) error {
	inlined := make([]*genai.InlinedRequest, len(requests))
	for i, req := range requests {
		contents, system := convertMessagesToGemini(req.Messages)
		config := &genai.GenerateContentConfig{}
		if len(system) > 0 {
			config.SystemInstruction = &genai.Content{
				Parts: []*genai.Part{{Text: strings.Join(system, "\n\n")}},
			}
		}
		if len(req.Tools) > 0 {
			config.Tools = convertToolsToGemini(req.Tools)
		}

		inlined[i] = &genai.InlinedRequest{
			Model:    p.model,
			Contents: contents,
			Config:   config,
			Metadata: map[string]string{"custom_id": req.ID},
		}
	}

	if p.options.progressCallback != nil {
		p.options.progressCallback(batch.Progress{
			Total:  len(results),
			Status: "submitting",
		})
	}

	job, err := p.client.Batches.Create(
		ctx, p.model,
		&genai.BatchJobSource{InlinedRequests: inlined},
		&genai.CreateBatchJobConfig{},
	)
	if err != nil {
		return fmt.Errorf("failed to create batch job: %w", err)
	}

	job, err = p.pollUntilDone(ctx, job.Name, len(results))
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
				results[globalIdx].Err = fmt.Errorf("%s", resp.Error.Message)
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

func (p *Processor) processEmbeddingBatch(
	ctx context.Context,
	requests []batch.Request,
	results []batch.Result,
	idxMap map[int]int,
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

	if p.options.progressCallback != nil {
		p.options.progressCallback(batch.Progress{
			Total:  len(results),
			Status: "submitting",
		})
	}

	embedModel := p.model
	if p.options.embeddingModel.APIModel != "" {
		embedModel = p.options.embeddingModel.APIModel
	}

	job, err := p.client.Batches.CreateEmbeddings(
		ctx, &embedModel,
		&genai.EmbeddingsBatchJobSource{
			InlinedRequests: &genai.EmbedContentBatch{
				Contents: allContents,
			},
		},
		&genai.CreateEmbeddingsBatchJobConfig{},
	)
	if err != nil {
		return fmt.Errorf("failed to create embedding batch job: %w", err)
	}

	job, err = p.pollUntilDone(ctx, job.Name, len(results))
	if err != nil {
		return err
	}

	if job.Dest != nil && len(job.Dest.InlinedEmbedContentResponses) > 0 {
		reqEmbeddings := make(map[int][][]float32)
		reqTokens := make(map[int]int64)

		for i, resp := range job.Dest.InlinedEmbedContentResponses {
			reqIdx := contentToReq[i]

			if resp.Error != nil {
				globalIdx := idxMap[reqIdx]
				results[globalIdx].Err = fmt.Errorf("%s", resp.Error.Message)
				continue
			}

			if resp.Response != nil && resp.Response.Embedding != nil {
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

func (p *Processor) pollUntilDone(
	ctx context.Context,
	jobName string,
	total int,
) (*genai.BatchJob, error) {
	ticker := time.NewTicker(p.options.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			job, err := p.client.Batches.Get(ctx, jobName, nil)
			if err != nil {
				return nil, err
			}

			if p.options.progressCallback != nil {
				completed := 0
				failed := 0
				if job.CompletionStats != nil {
					completed = int(job.CompletionStats.SuccessfulCount)
					failed = int(job.CompletionStats.FailedCount)
				}
				p.options.progressCallback(batch.Progress{
					Total:     total,
					Completed: completed,
					Failed:    failed,
					Status:    "polling",
				})
			}

			switch job.State {
			case genai.JobStateSucceeded, genai.JobStatePartiallySucceeded:
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
				Role:  "user",
				Parts: []*genai.Part{{Text: msg.Content().String()}},
			})
		case message.Assistant:
			parts := []*genai.Part{}
			if msg.Content().String() != "" {
				parts = append(parts, &genai.Part{Text: msg.Content().String()})
			}
			for _, tc := range msg.ToolCalls() {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Input), &args)
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: tc.Name,
						Args: args,
					},
				})
			}
			contents = append(
				contents,
				&genai.Content{Role: "model", Parts: parts},
			)
		case message.Tool:
			for _, tr := range msg.ToolResults() {
				var respData map[string]any
				if err := json.Unmarshal(
					[]byte(tr.Content),
					&respData,
				); err != nil {
					respData = map[string]any{"result": tr.Content}
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
		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:        info.Name,
			Description: info.Description,
			Parameters:  convertToGenaiSchema(info.Parameters, info.Required),
		})
	}

	return []*genai.Tool{{FunctionDeclarations: declarations}}
}

func convertToGenaiSchema(
	properties map[string]any,
	required []string,
) *genai.Schema {
	s := &genai.Schema{
		Type:       genai.TypeObject,
		Properties: make(map[string]*genai.Schema),
		Required:   required,
	}

	for key, val := range properties {
		if propMap, ok := val.(map[string]any); ok {
			s.Properties[key] = convertPropertyToGenaiSchema(propMap)
		}
	}

	return s
}

// convertPropertyToGenaiSchema converts a single JSON-schema property to a
// genai.Schema, recursing into array items and nested object properties. An
// `array` without Items, or an `object` without nested Properties, is rejected
// by the Gemini API ("response_schema.properties[x].items: missing field"), so
// the conversion must descend rather than emit only the top-level Type.
func convertPropertyToGenaiSchema(propMap map[string]any) *genai.Schema {
	s := &genai.Schema{}

	if desc, ok := propMap["description"].(string); ok {
		s.Description = desc
	}

	typeStr, ok := propMap["type"].(string)
	if !ok {
		s.Type = genai.TypeString
		return s
	}
	s.Type = mapJSONTypeToGenai(typeStr)

	switch typeStr {
	case "array":
		if items, ok := propMap["items"].(map[string]any); ok {
			s.Items = convertPropertyToGenaiSchema(items)
		}
	case "object":
		if props, ok := propMap["properties"].(map[string]any); ok {
			s.Properties = make(map[string]*genai.Schema, len(props))
			for k, v := range props {
				if vm, ok := v.(map[string]any); ok {
					s.Properties[k] = convertPropertyToGenaiSchema(vm)
				}
			}
		}
		if req, ok := propMap["required"].([]any); ok {
			reqStrs := make([]string, 0, len(req))
			for _, r := range req {
				if rs, ok := r.(string); ok {
					reqStrs = append(reqStrs, rs)
				}
			}
			s.Required = reqStrs
		}
	}

	if enum, ok := propMap["enum"].([]any); ok {
		enumStrings := make([]string, 0, len(enum))
		for _, v := range enum {
			if str, ok := v.(string); ok {
				enumStrings = append(enumStrings, str)
			}
		}
		s.Enum = enumStrings
	}

	return s
}

func mapJSONTypeToGenai(jsonType string) genai.Type {
	switch jsonType {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeString
	}
}

func convertGeminiResponse(resp *genai.GenerateContentResponse) *llm.Response {
	content := ""
	var toolCalls []message.ToolCall

	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
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
			InputTokens:     int64(resp.UsageMetadata.PromptTokenCount),
			OutputTokens:    int64(resp.UsageMetadata.CandidatesTokenCount),
			CacheReadTokens: int64(resp.UsageMetadata.CachedContentTokenCount),
		}
	}

	return &llm.Response{
		Content:      content,
		ToolCalls:    toolCalls,
		Usage:        usage,
		FinishReason: finishReason,
	}
}
