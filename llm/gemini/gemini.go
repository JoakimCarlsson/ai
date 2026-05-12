// Package gemini provides a Google Gemini implementation of the [llm.LLM] interface.
//
// The [llm/vertexai] package embeds this one for Gemini-on-Vertex deployments.
package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
	"google.golang.org/genai"
)

// ThinkingLevel controls thinking depth for Gemini models.
type ThinkingLevel string

// ThinkingLevel values.
const (
	ThinkingLevelMinimal ThinkingLevel = "minimal"
	ThinkingLevelLow     ThinkingLevel = "low"
	ThinkingLevelMedium  ThinkingLevel = "medium"
	ThinkingLevelHigh    ThinkingLevel = "high"
)

// Options configures the Gemini LLM client.
type Options struct {
	apiKey           string
	model            model.Model
	maxTokens        int64
	temperature      *float64
	topP             *float64
	topK             *int64
	stopSequences    []string
	timeout          *time.Duration
	disableCache     bool
	frequencyPenalty *float64
	presencePenalty  *float64
	seed             *int64
	thinkingLevel    *ThinkingLevel
	builtinTools     []*genai.Tool
}

// Option configures Options.
type Option func(*Options)

// WithAPIKey sets the API key.
func WithAPIKey(
	apiKey string,
) Option {
	return func(o *Options) { o.apiKey = apiKey }
}

// WithModel selects the LLM model.
func WithModel(m model.Model) Option { return func(o *Options) { o.model = m } }

// WithMaxTokens sets the max generation tokens.
func WithMaxTokens(
	maxTokens int64,
) Option {
	return func(o *Options) { o.maxTokens = maxTokens }
}

// WithTemperature controls randomness.
func WithTemperature(
	t float64,
) Option {
	return func(o *Options) { o.temperature = &t }
}

// WithTopP sets nucleus sampling probability mass.
func WithTopP(p float64) Option { return func(o *Options) { o.topP = &p } }

// WithTopK limits token selection to the top K candidates.
func WithTopK(k int64) Option { return func(o *Options) { o.topK = &k } }

// WithStopSequences sets text sequences that halt generation.
func WithStopSequences(
	seqs ...string,
) Option {
	return func(o *Options) { o.stopSequences = seqs }
}

// WithTimeout sets the maximum duration to wait for API responses.
func WithTimeout(
	timeout time.Duration,
) Option {
	return func(o *Options) { o.timeout = &timeout }
}

// WithDisableCache disables response caching.
func WithDisableCache() Option { return func(o *Options) { o.disableCache = true } }

// WithFrequencyPenalty sets the frequency penalty.
func WithFrequencyPenalty(
	p float64,
) Option {
	return func(o *Options) { o.frequencyPenalty = &p }
}

// WithPresencePenalty sets the presence penalty.
func WithPresencePenalty(
	p float64,
) Option {
	return func(o *Options) { o.presencePenalty = &p }
}

// WithSeed sets a random seed for deterministic generation.
func WithSeed(seed int64) Option { return func(o *Options) { o.seed = &seed } }

// WithThinkingLevel sets the thinking level for Gemini models that support reasoning.
func WithThinkingLevel(level ThinkingLevel) Option {
	return func(o *Options) { o.thinkingLevel = &level }
}

// WithGoogleSearch enables Gemini's built-in Google Search grounding tool.
// Requires Gemini 2.x or newer; Gemini 1.5 used a different tool type that is
// not exposed here.
func WithGoogleSearch() Option {
	return func(o *Options) {
		o.builtinTools = append(o.builtinTools, &genai.Tool{
			GoogleSearch: &genai.GoogleSearch{},
		})
	}
}

// WithURLContext enables Gemini's url_context tool for grounding on URLs in the
// prompt.
func WithURLContext() Option {
	return func(o *Options) {
		o.builtinTools = append(o.builtinTools, &genai.Tool{
			URLContext: &genai.URLContext{},
		})
	}
}

// WithCodeExecution enables Gemini's built-in code_execution tool.
func WithCodeExecution() Option {
	return func(o *Options) {
		o.builtinTools = append(o.builtinTools, &genai.Tool{
			CodeExecution: &genai.ToolCodeExecution{},
		})
	}
}

// RetryConfig provides retry settings tuned for Gemini API behavior.
func RetryConfig() llm.RetryConfig {
	cfg := llm.DefaultRetryConfig()
	cfg.CheckRetryAfter = false
	return cfg
}

// wrapError converts Gemini's string-typed rate-limit errors into a
// [llm.GenericRetryableError] so [llm.ShouldRetry] can dispatch via [errors.As].
// Non-rate-limit errors pass through unchanged.
func wrapError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	keywords := []string{"rate limit", "quota exceeded", "too many requests"}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return llm.GenericRetryableError{Err: err, StatusCode: 429}
		}
	}
	return err
}

// Client implements [llm.LLM] against the Google Gemini API.
type Client struct {
	options Options
	client  *genai.Client
}

// NewLLM constructs a Gemini LLM client.
func NewLLM(opts ...Option) llm.LLM {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	client, _ := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  options.apiKey,
		Backend: genai.BackendGeminiAPI,
	})

	return llm.WithTracing(
		&Client{options: options, client: client},
		llm.TracingAttrs{
			MaxTokens:   options.maxTokens,
			Temperature: options.temperature,
			TopP:        options.topP,
		},
	)
}

// NewWithExistingClient is for embedding by other packages (e.g. llm/vertexai)
// that build the Gemini SDK client themselves and want this package's request logic.
// The returned *Client is the bare implementation, not wrapped in tracing.
func NewWithExistingClient(options Options, client *genai.Client) *Client {
	return &Client{options: options, client: client}
}

// Model returns the configured LLM model.
func (c *Client) Model() model.Model { return c.options.model }

// SupportsStructuredOutput reports whether the model supports structured output.
func (c *Client) SupportsStructuredOutput() bool {
	return c.options.model.SupportsStructuredOut
}

func (c *Client) convertMessages(
	messages []message.Message,
) ([]*genai.Content, []string) {
	var geminiMessages []*genai.Content
	var systemMessages []string

	for _, msg := range messages {
		switch msg.Role {
		case message.System:
			systemMessages = append(systemMessages, msg.Content().String())
		case message.User:
			parts := []*genai.Part{{Text: msg.Content().String()}}
			for _, binaryContent := range msg.BinaryContent() {
				parts = append(parts, &genai.Part{
					InlineData: &genai.Blob{
						MIMEType: binaryContent.MIMEType,
						Data:     binaryContent.Data,
					},
				})
			}
			geminiMessages = append(geminiMessages, &genai.Content{
				Role: "user", Parts: parts,
			})

		case message.Assistant:
			parts := []*genai.Part{}
			if msg.Content().String() != "" {
				parts = append(parts, &genai.Part{Text: msg.Content().String()})
			}
			for _, toolCall := range msg.ToolCalls() {
				var args map[string]interface{}
				_ = json.Unmarshal([]byte(toolCall.Input), &args)
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: toolCall.Name,
						Args: args,
					},
				})
			}
			if len(parts) > 0 {
				geminiMessages = append(geminiMessages, &genai.Content{
					Role: "model", Parts: parts,
				})
			}

		case message.Tool:
			for _, toolResult := range msg.ToolResults() {
				parts := []*genai.Part{{
					FunctionResponse: &genai.FunctionResponse{
						Name: toolResult.Name,
						Response: map[string]interface{}{
							"content": toolResult.Content,
						},
					},
				}}
				geminiMessages = append(geminiMessages, &genai.Content{
					Role: "function", Parts: parts,
				})
			}
		}
	}

	return geminiMessages, systemMessages
}

func (c *Client) convertTools(tools []tool.BaseTool) []*genai.Tool {
	var out []*genai.Tool
	if len(tools) > 0 {
		geminiTool := &genai.Tool{
			FunctionDeclarations: make(
				[]*genai.FunctionDeclaration,
				0,
				len(tools),
			),
		}
		for _, t := range tools {
			info := t.Info()
			geminiTool.FunctionDeclarations = append(
				geminiTool.FunctionDeclarations,
				&genai.FunctionDeclaration{
					Name:        info.Name,
					Description: info.Description,
					Parameters: &genai.Schema{
						Type:       genai.TypeObject,
						Properties: convertSchemaProperties(info.Parameters),
						Required:   info.Required,
					},
				},
			)
		}
		out = append(out, geminiTool)
	}
	return append(out, c.options.builtinTools...)
}

func (c *Client) finishReason(reason genai.FinishReason) message.FinishReason {
	switch reason {
	case genai.FinishReasonStop:
		return message.FinishReasonEndTurn
	case genai.FinishReasonMaxTokens:
		return message.FinishReasonMaxTokens
	default:
		return message.FinishReasonUnknown
	}
}

func (c *Client) applyThinkingConfig(config *genai.GenerateContentConfig) {
	if c.options.thinkingLevel == nil || !c.options.model.CanReason {
		return
	}
	tc := &genai.ThinkingConfig{IncludeThoughts: true}
	switch *c.options.thinkingLevel {
	case ThinkingLevelMinimal:
		tc.ThinkingLevel = genai.ThinkingLevelMinimal
	case ThinkingLevelLow:
		tc.ThinkingLevel = genai.ThinkingLevelLow
	case ThinkingLevelMedium:
		tc.ThinkingLevel = genai.ThinkingLevelMedium
	case ThinkingLevelHigh:
		tc.ThinkingLevel = genai.ThinkingLevelHigh
	}
	config.ThinkingConfig = tc
}

func (c *Client) buildConfig(
	systemMessages []string,
	tools []tool.BaseTool,
) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(c.options.maxTokens),
	}

	pb := llm.NewParameterBuilder(
		c.options.temperature,
		c.options.topP,
		c.options.topK,
	)
	pb.ApplyFloat32Temperature(func(t *float32) { config.Temperature = t })
	pb.ApplyFloat32TopP(func(p *float32) { config.TopP = p })
	pb.ApplyFloat32TopK(func(k *float32) { config.TopK = k })
	pb.ApplyFloat32FrequencyPenalty(c.options.frequencyPenalty,
		func(fp *float32) { config.FrequencyPenalty = fp })
	pb.ApplyFloat32PresencePenalty(c.options.presencePenalty,
		func(pp *float32) { config.PresencePenalty = pp })
	pb.ApplyInt32Seed(c.options.seed, func(s *int32) { config.Seed = s })
	c.applyThinkingConfig(config)

	if len(c.options.stopSequences) > 0 {
		config.StopSequences = c.options.stopSequences
	}

	if len(systemMessages) > 0 {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: strings.Join(systemMessages, "\n\n")}},
		}
	}

	if len(tools) > 0 || len(c.options.builtinTools) > 0 {
		config.Tools = c.convertTools(tools)
	}

	return config
}

// SendMessages sends a conversation and returns the complete response.
func (c *Client) SendMessages(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) (*llm.Response, error) {
	geminiMessages, systemMessages := c.convertMessages(messages)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	if len(geminiMessages) == 0 {
		return nil, errors.New("gemini: no messages to send")
	}
	history := geminiMessages[:len(geminiMessages)-1]
	lastMsg := geminiMessages[len(geminiMessages)-1]
	config := c.buildConfig(systemMessages, tools)

	chat, err := c.client.Chats.Create(
		ctx,
		c.options.model.APIModel,
		config,
		history,
	)
	if err != nil {
		return nil, fmt.Errorf("gemini chat create: %w", err)
	}

	return llm.ExecuteWithRetry(
		ctx,
		RetryConfig(),
		func() (*llm.Response, error) {
			var lastMsgParts []genai.Part
			for _, part := range lastMsg.Parts {
				lastMsgParts = append(lastMsgParts, *part)
			}
			resp, err := chat.SendMessage(ctx, lastMsgParts...)
			if err != nil {
				return nil, wrapError(err)
			}

			var toolCalls []message.ToolCall
			content := ""

			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
					if part.FunctionCall != nil {
						id := "call_" + uuid.New().String()
						args, _ := json.Marshal(part.FunctionCall.Args)
						toolCalls = append(toolCalls, message.ToolCall{
							ID:       id,
							Name:     part.FunctionCall.Name,
							Input:    string(args),
							Type:     "function",
							Finished: true,
						})
						continue
					}
					content += partText(part)
				}
			}
			finishReason := message.FinishReasonEndTurn
			if len(resp.Candidates) > 0 {
				finishReason = c.finishReason(resp.Candidates[0].FinishReason)
			}
			if len(toolCalls) > 0 {
				finishReason = message.FinishReasonToolUse
			}

			return &llm.Response{
				Content:          content,
				ToolCalls:        toolCalls,
				Usage:            c.usage(resp),
				FinishReason:     finishReason,
				ProviderMetadata: groundingMetadata(resp),
			}, nil
		},
	)
}

// StreamResponse sends a conversation and returns a channel of streaming events.
func (c *Client) StreamResponse(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
) <-chan llm.Event {
	return c.streamInternal(ctx, messages, tools, nil)
}

// SendMessagesWithStructuredOutput sends with a JSON schema constraint.
func (c *Client) SendMessagesWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) (*llm.Response, error) {
	geminiMessages, systemMessages := c.convertMessages(messages)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	if len(geminiMessages) == 0 {
		return nil, errors.New("gemini: no messages to send")
	}
	history := geminiMessages[:len(geminiMessages)-1]
	lastMsg := geminiMessages[len(geminiMessages)-1]
	config := c.buildConfig(systemMessages, tools)
	config.ResponseSchema = c.convertSchemaToGenai(
		outputSchema.Parameters,
		outputSchema.Required,
	)

	chat, err := c.client.Chats.Create(
		ctx,
		c.options.model.APIModel,
		config,
		history,
	)
	if err != nil {
		return nil, fmt.Errorf("gemini chat create: %w", err)
	}

	return llm.ExecuteWithRetry(
		ctx,
		RetryConfig(),
		func() (*llm.Response, error) {
			response, err := chat.Send(ctx, lastMsg.Parts[0])
			if err != nil {
				return nil, wrapError(err)
			}

			content := ""
			toolCalls := []message.ToolCall{}
			for _, candidate := range response.Candidates {
				if candidate.Content == nil {
					continue
				}
				for _, part := range candidate.Content.Parts {
					if part.FunctionCall != nil {
						input, _ := json.Marshal(part.FunctionCall.Args)
						toolCalls = append(toolCalls, message.ToolCall{
							ID:    part.FunctionCall.Name,
							Name:  part.FunctionCall.Name,
							Input: string(input),
							Type:  "function",
						})
						continue
					}
					content += partText(part)
				}
			}

			finishReason := message.FinishReasonEndTurn
			if len(response.Candidates) > 0 {
				finishReason = c.finishReason(
					response.Candidates[0].FinishReason,
				)
			}
			if len(toolCalls) > 0 {
				finishReason = message.FinishReasonToolUse
			}

			return &llm.Response{
				Content:                    content,
				ToolCalls:                  toolCalls,
				Usage:                      c.usage(response),
				FinishReason:               finishReason,
				StructuredOutput:           &content,
				UsedNativeStructuredOutput: true,
				ProviderMetadata:           groundingMetadata(response),
			}, nil
		},
	)
}

// StreamResponseWithStructuredOutput streams with a JSON schema constraint.
func (c *Client) StreamResponseWithStructuredOutput(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	return c.streamInternal(ctx, messages, tools, outputSchema)
}

func (c *Client) streamInternal(
	ctx context.Context,
	messages []message.Message,
	tools []tool.BaseTool,
	outputSchema *schema.StructuredOutputInfo,
) <-chan llm.Event {
	geminiMessages, systemMessages := c.convertMessages(messages)

	ctx, cancel := llm.ApplyTimeout(ctx, c.options.timeout)
	defer cancel()

	if len(geminiMessages) == 0 {
		eventChan := make(chan llm.Event, 1)
		eventChan <- llm.Event{Type: types.EventError, Error: errors.New("gemini: no messages to send")}
		close(eventChan)
		return eventChan
	}

	history := geminiMessages[:len(geminiMessages)-1]
	lastMsg := geminiMessages[len(geminiMessages)-1]
	config := c.buildConfig(systemMessages, tools)
	if outputSchema != nil {
		config.ResponseSchema = c.convertSchemaToGenai(
			outputSchema.Parameters,
			outputSchema.Required,
		)
	}

	chat, err := c.client.Chats.Create(
		ctx,
		c.options.model.APIModel,
		config,
		history,
	)
	if err != nil {
		eventChan := make(chan llm.Event, 1)
		eventChan <- llm.Event{Type: types.EventError, Error: fmt.Errorf("gemini chat create: %w", err)}
		close(eventChan)
		return eventChan
	}

	eventChan := make(chan llm.Event)

	go func() {
		defer close(eventChan)

		llm.ExecuteStreamWithRetry(ctx, RetryConfig(), func() error {
			currentContent := ""
			toolCalls := []message.ToolCall{}
			var finalResp *genai.GenerateContentResponse

			eventChan <- llm.Event{Type: types.EventContentStart}

			var lastMsgParts []genai.Part
			for _, part := range lastMsg.Parts {
				lastMsgParts = append(lastMsgParts, *part)
			}

			for resp, err := range chat.SendMessageStream(ctx, lastMsgParts...) {
				if err != nil {
					return wrapError(err)
				}

				finalResp = resp

				if len(resp.Candidates) > 0 &&
					resp.Candidates[0].Content != nil {
					for _, part := range resp.Candidates[0].Content.Parts {
						if part.Thought && part.Text != "" {
							eventChan <- llm.Event{
								Type:     types.EventThinkingDelta,
								Thinking: string(part.Text),
							}
							continue
						}
						if part.FunctionCall != nil {
							id := "call_" + uuid.New().String()
							args, _ := json.Marshal(part.FunctionCall.Args)
							newCall := message.ToolCall{
								ID:       id,
								Name:     part.FunctionCall.Name,
								Input:    string(args),
								Type:     "function",
								Finished: true,
							}

							isNew := true
							for _, existing := range toolCalls {
								if existing.Name == newCall.Name &&
									existing.Input == newCall.Input {
									isNew = false
									break
								}
							}

							if isNew {
								toolCalls = append(toolCalls, newCall)
							}
							continue
						}
						delta := partText(part)
						if delta == "" {
							continue
						}
						currentContent += delta
						eventChan <- llm.Event{
							Type:    types.EventContentDelta,
							Content: delta,
						}
					}
				}
			}

			eventChan <- llm.Event{Type: types.EventContentStop}

			if finalResp != nil {
				finishReason := message.FinishReasonEndTurn
				if len(finalResp.Candidates) > 0 {
					finishReason = c.finishReason(
						finalResp.Candidates[0].FinishReason,
					)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}
				resp := &llm.Response{
					Content:          currentContent,
					ToolCalls:        toolCalls,
					Usage:            c.usage(finalResp),
					FinishReason:     finishReason,
					ProviderMetadata: groundingMetadata(finalResp),
				}
				if outputSchema != nil {
					resp.StructuredOutput = &currentContent
					resp.UsedNativeStructuredOutput = true
				}
				eventChan <- llm.Event{Type: types.EventComplete, Response: resp}
			}
			return nil
		}, eventChan)
	}()

	return eventChan
}

// partText extracts the textual representation of a non-function-call Part,
// formatting executable_code / code_execution_result blocks as fenced code so
// they survive concatenation into Response.Content.
func partText(part *genai.Part) string {
	switch {
	case part.Text != "" && !part.Thought:
		return string(part.Text)
	case part.ExecutableCode != nil:
		lang := strings.ToLower(string(part.ExecutableCode.Language))
		if lang == "" || lang == "language_unspecified" {
			lang = ""
		}
		return fmt.Sprintf("\n```%s\n%s\n```\n", lang, part.ExecutableCode.Code)
	case part.CodeExecutionResult != nil:
		return fmt.Sprintf("\n```\n%s\n```\n", part.CodeExecutionResult.Output)
	}
	return ""
}

// groundingMetadata extracts Gemini grounding / URL-context metadata from a
// completed response into a provider-namespaced map suitable for
// Response.ProviderMetadata. Returns nil when no grounding ran.
func groundingMetadata(resp *genai.GenerateContentResponse) map[string]any {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}
	cand := resp.Candidates[0]
	out := map[string]any{}
	if cand.GroundingMetadata != nil {
		gm := cand.GroundingMetadata
		var chunks []map[string]any
		for _, ch := range gm.GroundingChunks {
			if ch == nil || ch.Web == nil {
				continue
			}
			chunks = append(chunks, map[string]any{
				"uri":    ch.Web.URI,
				"title":  ch.Web.Title,
				"domain": ch.Web.Domain,
			})
		}
		grounding := map[string]any{}
		if len(gm.WebSearchQueries) > 0 {
			grounding["web_search_queries"] = gm.WebSearchQueries
		}
		if len(chunks) > 0 {
			grounding["chunks"] = chunks
		}
		if len(grounding) > 0 {
			out["gemini.grounding"] = grounding
		}
	}
	if cand.URLContextMetadata != nil {
		var urls []map[string]any
		for _, u := range cand.URLContextMetadata.URLMetadata {
			if u == nil {
				continue
			}
			urls = append(urls, map[string]any{
				"retrieved_url":        u.RetrievedURL,
				"url_retrieval_status": string(u.URLRetrievalStatus),
			})
		}
		if len(urls) > 0 {
			out["gemini.url_context"] = map[string]any{"urls": urls}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c *Client) usage(resp *genai.GenerateContentResponse) llm.TokenUsage {
	if resp == nil || resp.UsageMetadata == nil {
		return llm.TokenUsage{}
	}
	return llm.TokenUsage{
		InputTokens:         int64(resp.UsageMetadata.PromptTokenCount),
		OutputTokens:        int64(resp.UsageMetadata.CandidatesTokenCount),
		CacheCreationTokens: 0,
		CacheReadTokens:     int64(resp.UsageMetadata.CachedContentTokenCount),
	}
}

func convertSchemaProperties(
	parameters map[string]interface{},
) map[string]*genai.Schema {
	properties := make(map[string]*genai.Schema)
	for name, param := range parameters {
		properties[name] = convertToSchema(param)
	}
	return properties
}

func convertToSchema(param interface{}) *genai.Schema {
	s := &genai.Schema{Type: genai.TypeString}

	paramMap, ok := param.(map[string]interface{})
	if !ok {
		return s
	}

	if desc, ok := paramMap["description"].(string); ok {
		s.Description = desc
	}

	typeVal, hasType := paramMap["type"]
	if !hasType {
		return s
	}

	typeStr, ok := typeVal.(string)
	if !ok {
		return s
	}

	s.Type = mapJSONTypeToGenAI(typeStr)

	switch typeStr {
	case "array":
		s.Items = processArrayItems(paramMap)
	case "object":
		if props, ok := paramMap["properties"].(map[string]interface{}); ok {
			s.Properties = convertSchemaProperties(props)
		}
	}

	return s
}

func processArrayItems(paramMap map[string]interface{}) *genai.Schema {
	items, ok := paramMap["items"].(map[string]interface{})
	if !ok {
		return nil
	}
	return convertToSchema(items)
}

func mapJSONTypeToGenAI(jsonType string) genai.Type {
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

func (c *Client) convertSchemaToGenai(
	parameters map[string]any,
	required []string,
) *genai.Schema {
	s := &genai.Schema{
		Type:       genai.TypeObject,
		Properties: make(map[string]*genai.Schema),
		Required:   required,
	}

	for name, prop := range parameters {
		if propMap, ok := prop.(map[string]any); ok {
			propSchema := &genai.Schema{}

			if typeVal, ok := propMap["type"].(string); ok {
				propSchema.Type = mapJSONTypeToGenAI(typeVal)
			}
			if desc, ok := propMap["description"].(string); ok {
				propSchema.Description = desc
			}
			if items, ok := propMap["items"].(map[string]any); ok {
				propSchema.Items = c.convertPropertyToGenai(items)
			}
			if enum, ok := propMap["enum"].([]any); ok {
				enumStrings := make([]string, len(enum))
				for i, v := range enum {
					if str, ok := v.(string); ok {
						enumStrings[i] = str
					}
				}
				propSchema.Enum = enumStrings
			}

			s.Properties[name] = propSchema
		}
	}

	return s
}

func (c *Client) convertPropertyToGenai(propMap map[string]any) *genai.Schema {
	s := &genai.Schema{}
	if typeVal, ok := propMap["type"].(string); ok {
		s.Type = mapJSONTypeToGenAI(typeVal)
	}
	if desc, ok := propMap["description"].(string); ok {
		s.Description = desc
	}
	return s
}
