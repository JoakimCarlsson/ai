package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/schema"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/types"
	"google.golang.org/genai"
)

type geminiOptions struct {
	disableCache     bool
	frequencyPenalty *float64
	presencePenalty  *float64
	seed             *int64
}

type GeminiOption func(*geminiOptions)

type geminiClient struct {
	providerOptions llmClientOptions
	options         geminiOptions
	client          *genai.Client
}

type GeminiClient LLMClient

func newGeminiClient(opts llmClientOptions) GeminiClient {
	geminiOpts := geminiOptions{}
	for _, o := range opts.geminiOptions {
		o(&geminiOpts)
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: opts.apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil
	}

	return &geminiClient{
		providerOptions: opts,
		options:         geminiOpts,
		client:          client,
	}
}

func (g *geminiClient) convertMessages(messages []message.Message) ([]*genai.Content, []string) {
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

			content := &genai.Content{
				Role:  "user",
				Parts: parts,
			}

			geminiMessages = append(geminiMessages, content)

		case message.Assistant:
			parts := []*genai.Part{}
			if msg.Content().String() != "" {
				parts = append(parts, &genai.Part{Text: msg.Content().String()})
			}

			for _, toolCall := range msg.ToolCalls() {
				var args map[string]interface{}
				json.Unmarshal([]byte(toolCall.Input), &args)
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: toolCall.Name,
						Args: args,
					},
				})
			}

			if len(parts) > 0 {
				geminiMessages = append(geminiMessages, &genai.Content{
					Role:  "model",
					Parts: parts,
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
					Role:  "function",
					Parts: parts,
				})
			}
		}
	}

	return geminiMessages, systemMessages
}

func (g *geminiClient) convertTools(tools []tool.BaseTool) []*genai.Tool {
	geminiTool := &genai.Tool{}
	geminiTool.FunctionDeclarations = make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, tool := range tools {
		info := tool.Info()
		declaration := &genai.FunctionDeclaration{
			Name:        info.Name,
			Description: info.Description,
			Parameters: &genai.Schema{
				Type:       genai.TypeObject,
				Properties: convertSchemaProperties(info.Parameters),
				Required:   info.Required,
			},
		}

		geminiTool.FunctionDeclarations = append(geminiTool.FunctionDeclarations, declaration)
	}

	return []*genai.Tool{geminiTool}
}

func (g *geminiClient) finishReason(reason genai.FinishReason) message.FinishReason {
	switch {
	case reason == genai.FinishReasonStop:
		return message.FinishReasonEndTurn
	case reason == genai.FinishReasonMaxTokens:
		return message.FinishReasonMaxTokens
	default:
		return message.FinishReasonUnknown
	}
}

func (g *geminiClient) send(ctx context.Context, messages []message.Message, tools []tool.BaseTool) (*LLMResponse, error) {
	geminiMessages, systemMessages := g.convertMessages(messages)

	if g.providerOptions.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *g.providerOptions.timeout)
		defer cancel()
	}

	history := geminiMessages[:len(geminiMessages)-1]
	lastMsg := geminiMessages[len(geminiMessages)-1]
	config := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(g.providerOptions.maxTokens),
	}

	if g.providerOptions.temperature != nil {
		temp := float32(*g.providerOptions.temperature)
		config.Temperature = &temp
	}

	if g.providerOptions.topP != nil {
		topP := float32(*g.providerOptions.topP)
		config.TopP = &topP
	}

	if g.providerOptions.topK != nil {
		topK := float32(*g.providerOptions.topK)
		config.TopK = &topK
	}

	if g.options.frequencyPenalty != nil {
		fp := float32(*g.options.frequencyPenalty)
		config.FrequencyPenalty = &fp
	}

	if g.options.presencePenalty != nil {
		pp := float32(*g.options.presencePenalty)
		config.PresencePenalty = &pp
	}

	if g.options.seed != nil {
		seed := int32(*g.options.seed)
		config.Seed = &seed
	}

	if len(g.providerOptions.stopSequences) > 0 {
		config.StopSequences = g.providerOptions.stopSequences
	}

	if len(systemMessages) > 0 {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: strings.Join(systemMessages, "\n\n")}},
		}
	}

	if len(tools) > 0 {
		config.Tools = g.convertTools(tools)
	}
	chat, _ := g.client.Chats.Create(ctx, g.providerOptions.model.APIModel, config, history)

	return ExecuteWithRetry(ctx, GeminiRetryConfig(), func() (*LLMResponse, error) {
		var toolCalls []message.ToolCall

		var lastMsgParts []genai.Part
		for _, part := range lastMsg.Parts {
			lastMsgParts = append(lastMsgParts, *part)
		}
		resp, err := chat.SendMessage(ctx, lastMsgParts...)
		if err != nil {
			return nil, err
		}

		content := ""

		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			for _, part := range resp.Candidates[0].Content.Parts {
				switch {
				case part.Text != "":
					content = string(part.Text)
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
			finishReason = g.finishReason(resp.Candidates[0].FinishReason)
		}
		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &LLMResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        g.usage(resp),
			FinishReason: finishReason,
		}, nil
	})
}

func (g *geminiClient) stream(ctx context.Context, messages []message.Message, tools []tool.BaseTool) <-chan LLMEvent {
	geminiMessages, systemMessages := g.convertMessages(messages)

	if g.providerOptions.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *g.providerOptions.timeout)
		defer cancel()
	}

	history := geminiMessages[:len(geminiMessages)-1]
	lastMsg := geminiMessages[len(geminiMessages)-1]
	config := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(g.providerOptions.maxTokens),
	}

	if g.providerOptions.temperature != nil {
		temp := float32(*g.providerOptions.temperature)
		config.Temperature = &temp
	}

	if g.providerOptions.topP != nil {
		topP := float32(*g.providerOptions.topP)
		config.TopP = &topP
	}

	if g.providerOptions.topK != nil {
		topK := float32(*g.providerOptions.topK)
		config.TopK = &topK
	}

	if g.options.frequencyPenalty != nil {
		fp := float32(*g.options.frequencyPenalty)
		config.FrequencyPenalty = &fp
	}

	if g.options.presencePenalty != nil {
		pp := float32(*g.options.presencePenalty)
		config.PresencePenalty = &pp
	}

	if g.options.seed != nil {
		seed := int32(*g.options.seed)
		config.Seed = &seed
	}

	if len(g.providerOptions.stopSequences) > 0 {
		config.StopSequences = g.providerOptions.stopSequences
	}

	if len(systemMessages) > 0 {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: strings.Join(systemMessages, "\n\n")}},
		}
	}

	if len(tools) > 0 {
		config.Tools = g.convertTools(tools)
	}
	chat, _ := g.client.Chats.Create(ctx, g.providerOptions.model.APIModel, config, history)

	eventChan := make(chan LLMEvent)

	go func() {
		defer close(eventChan)

		ExecuteStreamWithRetry(ctx, GeminiRetryConfig(), func() error {
			currentContent := ""
			toolCalls := []message.ToolCall{}
			var finalResp *genai.GenerateContentResponse

			eventChan <- LLMEvent{Type: types.EventContentStart}

			var lastMsgParts []genai.Part

			for _, part := range lastMsg.Parts {
				lastMsgParts = append(lastMsgParts, *part)
			}
			for resp, err := range chat.SendMessageStream(ctx, lastMsgParts...) {
				if err != nil {
					return err
				}

				finalResp = resp

				if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
					for _, part := range resp.Candidates[0].Content.Parts {
						switch {
						case part.Text != "":
							delta := string(part.Text)
							currentContent += delta
							eventChan <- LLMEvent{
								Type:    types.EventContentDelta,
								Content: delta,
							}
						case part.FunctionCall != nil:
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
								if existing.Name == newCall.Name && existing.Input == newCall.Input {
									isNew = false
									break
								}
							}

							if isNew {
								toolCalls = append(toolCalls, newCall)
							}
						}
					}
				}
			}

			eventChan <- LLMEvent{Type: types.EventContentStop}

			if finalResp != nil {

				finishReason := message.FinishReasonEndTurn
				if len(finalResp.Candidates) > 0 {
					finishReason = g.finishReason(finalResp.Candidates[0].FinishReason)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}
				eventChan <- LLMEvent{
					Type: types.EventComplete,
					Response: &LLMResponse{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        g.usage(finalResp),
						FinishReason: finishReason,
					},
				}
				return nil
			}
			return nil
		}, eventChan)
	}()

	return eventChan
}

func (g *geminiClient) usage(resp *genai.GenerateContentResponse) TokenUsage {
	if resp == nil || resp.UsageMetadata == nil {
		return TokenUsage{}
	}

	return TokenUsage{
		InputTokens:         int64(resp.UsageMetadata.PromptTokenCount),
		OutputTokens:        int64(resp.UsageMetadata.CandidatesTokenCount),
		CacheCreationTokens: 0,
		CacheReadTokens:     int64(resp.UsageMetadata.CachedContentTokenCount),
	}
}

// WithGeminiDisableCache disables response caching for Gemini requests
func WithGeminiDisableCache() GeminiOption {
	return func(options *geminiOptions) {
		options.disableCache = true
	}
}

// WithGeminiFrequencyPenalty sets the frequency penalty to reduce repetition in responses
func WithGeminiFrequencyPenalty(frequencyPenalty float64) GeminiOption {
	return func(options *geminiOptions) {
		options.frequencyPenalty = &frequencyPenalty
	}
}

// WithGeminiPresencePenalty sets the presence penalty to encourage topic diversity
func WithGeminiPresencePenalty(presencePenalty float64) GeminiOption {
	return func(options *geminiOptions) {
		options.presencePenalty = &presencePenalty
	}
}

// WithGeminiSeed sets a random seed for deterministic response generation
func WithGeminiSeed(seed int64) GeminiOption {
	return func(options *geminiOptions) {
		options.seed = &seed
	}
}

func convertSchemaProperties(parameters map[string]interface{}) map[string]*genai.Schema {
	properties := make(map[string]*genai.Schema)

	for name, param := range parameters {
		properties[name] = convertToSchema(param)
	}

	return properties
}

func convertToSchema(param interface{}) *genai.Schema {
	schema := &genai.Schema{Type: genai.TypeString}

	paramMap, ok := param.(map[string]interface{})
	if !ok {
		return schema
	}

	if desc, ok := paramMap["description"].(string); ok {
		schema.Description = desc
	}

	typeVal, hasType := paramMap["type"]
	if !hasType {
		return schema
	}

	typeStr, ok := typeVal.(string)
	if !ok {
		return schema
	}

	schema.Type = mapJSONTypeToGenAI(typeStr)

	switch typeStr {
	case "array":
		schema.Items = processArrayItems(paramMap)
	case "object":
		if props, ok := paramMap["properties"].(map[string]interface{}); ok {
			schema.Properties = convertSchemaProperties(props)
		}
	}

	return schema
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

func (g *geminiClient) supportsStructuredOutput() bool {
	return true
}

func (g *geminiClient) sendWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) (*LLMResponse, error) {
	geminiMessages, systemMessages := g.convertMessages(messages)

	if g.providerOptions.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *g.providerOptions.timeout)
		defer cancel()
	}

	history := geminiMessages[:len(geminiMessages)-1]
	lastMsg := geminiMessages[len(geminiMessages)-1]
	config := &genai.GenerateContentConfig{
		MaxOutputTokens: int32(g.providerOptions.maxTokens),
	}
	
	responseSchema := g.convertSchemaToGenai(outputSchema.Parameters, outputSchema.Required)
	config.ResponseSchema = responseSchema

	if g.providerOptions.temperature != nil {
		temp := float32(*g.providerOptions.temperature)
		config.Temperature = &temp
	}

	if g.providerOptions.topP != nil {
		topP := float32(*g.providerOptions.topP)
		config.TopP = &topP
	}

	if g.providerOptions.topK != nil {
		topK := float32(*g.providerOptions.topK)
		config.TopK = &topK
	}

	if g.options.frequencyPenalty != nil {
		penalty := float32(*g.options.frequencyPenalty)
		config.FrequencyPenalty = &penalty
	}

	if g.options.presencePenalty != nil {
		penalty := float32(*g.options.presencePenalty)
		config.PresencePenalty = &penalty
	}

	if g.options.seed != nil {
		seed := int32(*g.options.seed)
		config.Seed = &seed
	}

	geminiTools := g.convertTools(tools)
	if len(geminiTools) > 0 {
		config.Tools = geminiTools
	}

	if len(systemMessages) > 0 {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: strings.Join(systemMessages, "\n\n")}},
		}
	}
	
	chat, _ := g.client.Chats.Create(ctx, g.providerOptions.model.APIModel, config, history)

	return ExecuteWithRetry(ctx, GeminiRetryConfig(), func() (*LLMResponse, error) {
		response, err := chat.Send(ctx, lastMsg.Parts[0])
		if err != nil {
			return nil, err
		}

		content := ""
		for _, candidate := range response.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					content += string(part.Text)
				}
			}
		}

		toolCalls := []message.ToolCall{}
		for _, candidate := range response.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.FunctionCall != nil {
					input, _ := json.Marshal(part.FunctionCall.Args)
					toolCalls = append(toolCalls, message.ToolCall{
						ID:    part.FunctionCall.Name,
						Name:  part.FunctionCall.Name,
						Input: string(input),
						Type:  "function",
					})
				}
			}
		}

		finishReason := message.FinishReasonEndTurn
		if len(response.Candidates) > 0 {
			finishReason = g.finishReason(response.Candidates[0].FinishReason)
		}

		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &LLMResponse{
			Content:                content,
			ToolCalls:              toolCalls,
			Usage:                  g.usage(response),
			FinishReason:           finishReason,
			StructuredOutput:       &content,
			UsedNativeStructuredOutput: true,
		}, nil
	})
}

func (g *geminiClient) streamWithStructuredOutput(ctx context.Context, messages []message.Message, tools []tool.BaseTool, outputSchema *schema.StructuredOutputInfo) <-chan LLMEvent {
	errChan := make(chan LLMEvent, 1)
	errChan <- LLMEvent{
		Type:  types.EventTypeError,
		Error: errors.New("structured output streaming not yet implemented for Gemini - use non-streaming method"),
	}
	close(errChan)
	return errChan
}

func (g *geminiClient) convertSchemaToGenai(parameters map[string]any, required []string) *genai.Schema {
	schema := &genai.Schema{
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
				propSchema.Items = g.convertPropertyToGenai(items)
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
			
			schema.Properties[name] = propSchema
		}
	}
	
	return schema
}

func (g *geminiClient) convertPropertyToGenai(propMap map[string]any) *genai.Schema {
	schema := &genai.Schema{}
	
	if typeVal, ok := propMap["type"].(string); ok {
		schema.Type = mapJSONTypeToGenAI(typeVal)
	}
	
	if desc, ok := propMap["description"].(string); ok {
		schema.Description = desc
	}
	
	return schema
}
