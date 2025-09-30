// Package message provides types and utilities for handling AI model messages and content.
//
// This package defines the core message structures used across all AI providers,
// including support for text, images, tool calls, and multimodal content. It provides
// a unified interface for creating and manipulating messages regardless of the
// underlying AI provider.
//
// Key types include Message for representing conversations, various ContentPart
// implementations for different content types, and utility functions for message
// creation and manipulation.
package message

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

// MessageRole represents the role of a message in a conversation.
type MessageRole string

const (
	// Assistant represents messages from the AI assistant.
	Assistant MessageRole = "assistant"
	// User represents messages from the human user.
	User MessageRole = "user"
	// System represents system-level instructions or context.
	System MessageRole = "system"
	// Tool represents responses from tool executions.
	Tool MessageRole = "tool"
)

// Attachment represents a file attachment with its MIME type and binary data.
type Attachment struct {
	// MIMEType specifies the media type of the attachment (e.g., "image/png", "text/plain").
	MIMEType string
	// Data contains the raw binary data of the attachment.
	Data []byte
}

// FinishReason indicates why a model stopped generating tokens.
type FinishReason string

const (
	// FinishReasonEndTurn indicates the model naturally completed its response.
	FinishReasonEndTurn FinishReason = "end_turn"
	// FinishReasonMaxTokens indicates the response was truncated due to token limits.
	FinishReasonMaxTokens FinishReason = "max_tokens"
	// FinishReasonToolUse indicates the model wants to use a tool.
	FinishReasonToolUse FinishReason = "tool_use"
	// FinishReasonCanceled indicates the request was canceled by the user.
	FinishReasonCanceled FinishReason = "canceled"
	// FinishReasonError indicates an error occurred during generation.
	FinishReasonError FinishReason = "error"
	// FinishReasonUnknown indicates an unknown finish reason.
	FinishReasonUnknown FinishReason = "unknown"
)

// ToolCall represents a request to execute a tool with specific parameters.
type ToolCall struct {
	// ID is a unique identifier for this tool call.
	ID string `json:"id"`
	// Name is the name of the tool to execute.
	Name string `json:"name"`
	// Input contains the JSON-encoded parameters for the tool.
	Input string `json:"input"`
	// Type specifies the type of tool call (usually "function").
	Type string `json:"type"`
	// Finished indicates whether the tool call has completed execution.
	Finished bool `json:"finished"`
}

func (ToolCall) isPart() {}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	// ToolCallID links this result to the corresponding tool call.
	ToolCallID string `json:"tool_call_id"`
	// Name is the name of the tool that was executed.
	Name string `json:"name"`
	// Content contains the result content from the tool execution.
	Content string `json:"content"`
	// Metadata contains additional JSON-encoded metadata about the result.
	Metadata string `json:"metadata"`
	// IsError indicates whether the tool execution resulted in an error.
	IsError bool `json:"is_error"`
}

func (ToolResult) isPart() {}

// ContentPart represents a piece of content within a message.
// It can be text, images, tool calls, or tool results.
type ContentPart interface {
	isPart()
}

// TextContent represents plain text content within a message.
type TextContent struct {
	// Text contains the actual text content.
	Text string `json:"text"`
}

// String returns the text content as a string.
func (tc TextContent) String() string {
	return tc.Text
}

func (TextContent) isPart() {}

// ImageURLContent represents an image referenced by URL within a message.
type ImageURLContent struct {
	// URL is the location of the image resource.
	URL string `json:"url"`
	// Detail specifies the level of detail for image processing (e.g., "low", "high").
	Detail string `json:"detail,omitempty"`
}

// String returns the image URL as a string.
func (iuc ImageURLContent) String() string {
	return iuc.URL
}

func (ImageURLContent) isPart() {}

// BinaryContent represents binary data (like images) embedded directly in a message.
type BinaryContent struct {
	// Path is an optional file path identifier for the binary content.
	Path string
	// MIMEType specifies the media type of the binary data.
	MIMEType string
	// Data contains the raw binary content.
	Data []byte
}

// String returns the binary content as a base64-encoded string,
// formatted according to the specified provider's requirements.
func (bc BinaryContent) String(provider model.ModelProvider) string {
	base64Encoded := base64.StdEncoding.EncodeToString(bc.Data)
	if provider == model.ProviderOpenAI {
		return "data:" + bc.MIMEType + ";base64," + base64Encoded
	}
	return base64Encoded
}

func (BinaryContent) isPart() {}

// Message represents a single message in a conversation with an AI model.
// It can contain multiple content parts including text, images, tool calls, and tool results.
type Message struct {
	// Role indicates who sent the message (user, assistant, system, or tool).
	Role MessageRole
	// Parts contains the various content components of the message.
	Parts []ContentPart
	// Model identifies which AI model this message is associated with.
	Model model.ModelID
	// CreatedAt is a Unix timestamp (nanoseconds) indicating when the message was created.
	CreatedAt int64
}

// NewMessage creates a new message with the specified role and content parts.
func NewMessage(role MessageRole, parts []ContentPart) Message {
	return Message{
		Role:      role,
		Parts:     parts,
		CreatedAt: time.Now().UnixNano(),
	}
}

// NewUserMessage creates a new user message with the given text content.
func NewUserMessage(text string) Message {
	return NewMessage(User, []ContentPart{TextContent{Text: text}})
}

// NewSystemMessage creates a new system message with the given text content.
func NewSystemMessage(text string) Message {
	return NewMessage(System, []ContentPart{TextContent{Text: text}})
}

// NewAssistantMessage creates a new empty assistant message.
func NewAssistantMessage() Message {
	return NewMessage(Assistant, []ContentPart{})
}

// Content returns the first text content part from the message.
func (m *Message) Content() TextContent {
	for _, part := range m.Parts {
		if c, ok := part.(TextContent); ok {
			return c
		}
	}
	return TextContent{}
}

// BinaryContent returns all binary content parts from the message.
func (m *Message) BinaryContent() []BinaryContent {
	binaryContents := make([]BinaryContent, 0)
	for _, part := range m.Parts {
		if c, ok := part.(BinaryContent); ok {
			binaryContents = append(binaryContents, c)
		}
	}
	return binaryContents
}

// ImageURLContent returns all image URL content parts from the message.
func (m *Message) ImageURLContent() []ImageURLContent {
	imageURLContents := make([]ImageURLContent, 0)
	for _, part := range m.Parts {
		if c, ok := part.(ImageURLContent); ok {
			imageURLContents = append(imageURLContents, c)
		}
	}
	return imageURLContents
}

// ToolCalls returns all tool call parts from the message.
func (m *Message) ToolCalls() []ToolCall {
	var toolCalls []ToolCall
	for _, part := range m.Parts {
		if tc, ok := part.(ToolCall); ok {
			toolCalls = append(toolCalls, tc)
		}
	}
	return toolCalls
}

// ToolResults returns all tool result parts from the message.
func (m *Message) ToolResults() []ToolResult {
	var toolResults []ToolResult
	for _, part := range m.Parts {
		if tr, ok := part.(ToolResult); ok {
			toolResults = append(toolResults, tr)
		}
	}
	return toolResults
}

// AppendContent adds text to the existing text content or creates new text content.
func (m *Message) AppendContent(delta string) {
	found := false
	for i, part := range m.Parts {
		if c, ok := part.(TextContent); ok {
			m.Parts[i] = TextContent{Text: c.Text + delta}
			found = true
			break
		}
	}
	if !found {
		m.Parts = append(m.Parts, TextContent{Text: delta})
	}
}

// AppendReasoningContent adds reasoning text content to the message.
// This is currently a placeholder for future reasoning content support.
func (m *Message) AppendReasoningContent(delta string) {
}

// SetToolCalls replaces all message parts with the provided tool calls.
func (m *Message) SetToolCalls(tc []ToolCall) {
	m.Parts = []ContentPart{}
	for _, call := range tc {
		m.Parts = append(m.Parts, call)
	}
}

// AppendToolCalls adds tool calls to the message without clearing existing content.
func (m *Message) AppendToolCalls(tc []ToolCall) {
	for _, call := range tc {
		m.Parts = append(m.Parts, call)
	}
}

// AddToolResult appends a tool result to the message parts.
func (m *Message) AddToolResult(tr ToolResult) {
	m.Parts = append(m.Parts, tr)
}

// SetToolResults replaces all message parts with the provided tool results.
func (m *Message) SetToolResults(tr []ToolResult) {
	m.Parts = []ContentPart{}
	for _, result := range tr {
		m.Parts = append(m.Parts, result)
	}
}

// AddFinish adds a finish reason to the message.
// This is currently a placeholder for future finish reason support.
func (m *Message) AddFinish(reason FinishReason) {
}

// AddImageURL adds an image URL content part to the message.
func (m *Message) AddImageURL(url, detail string) {
	m.Parts = append(m.Parts, ImageURLContent{URL: url, Detail: detail})
}

// AddBinary adds binary content to the message with the specified MIME type.
func (m *Message) AddBinary(mimeType string, data []byte) {
	m.Parts = append(m.Parts, BinaryContent{MIMEType: mimeType, Data: data})
}

// BaseMessage defines the interface for advanced message implementations
// with metadata, source tracking, and extended functionality.
type BaseMessage interface {
	// GetSource returns the source identifier for this message.
	GetSource() string
	// GetCreatedAt returns the creation timestamp of the message.
	GetCreatedAt() time.Time
	// GetContent returns the message content as a generic interface.
	GetContent() interface{}
	// GetMetadata returns the metadata map for this message.
	GetMetadata() map[string]interface{}
	// SetMetadata sets a metadata key-value pair for this message.
	SetMetadata(key string, value interface{})
	// GetRole returns the role of the message sender.
	GetRole() MessageRole
	// GetModel returns the model ID associated with this message.
	GetModel() model.ModelID
	// SetModel sets the model ID for this message.
	SetModel(modelID model.ModelID)
}

// MessageSource identifies the origin of a message with type and ID.
type MessageSource struct {
	// Type indicates the category or provider of the message source.
	Type string `json:"type"`
	// ID is a unique identifier within the source type.
	ID string `json:"id"`
}

// String returns a string representation of the message source.
func (ms MessageSource) String() string {
	if ms.ID != "" {
		return ms.Type + ":" + ms.ID
	}
	return ms.Type
}

// generateMessageID creates a unique message identifier based on the current timestamp.
func generateMessageID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// NewMessageSource creates a message source with type and ID, generating ID if empty.
func NewMessageSource(sourceType, id string) MessageSource {
	if id == "" {
		id = generateMessageID()
	}
	return MessageSource{Type: sourceType, ID: id}
}

// baseMessage provides a concrete implementation of the BaseMessage interface
// with source tracking, metadata, and timestamps.
type baseMessage struct {
	// Source identifies where this message originated.
	Source MessageSource `json:"source"`
	// CreatedAt is the timestamp when this message was created.
	CreatedAt time.Time `json:"created_at"`
	// Metadata contains additional key-value data associated with the message.
	Metadata map[string]interface{} `json:"metadata"`
	// Role indicates the sender's role in the conversation.
	Role MessageRole `json:"role"`
	// Model specifies which AI model is associated with this message.
	Model model.ModelID `json:"model"`
}

// GetSource returns the string representation of the message source.
func (bm *baseMessage) GetSource() string {
	return bm.Source.String()
}

// GetCreatedAt returns the creation timestamp of the message.
func (bm *baseMessage) GetCreatedAt() time.Time {
	return bm.CreatedAt
}

// GetMetadata returns the metadata map, initializing it if necessary.
func (bm *baseMessage) GetMetadata() map[string]interface{} {
	if bm.Metadata == nil {
		bm.Metadata = make(map[string]interface{})
	}
	return bm.Metadata
}

// SetMetadata sets a metadata key-value pair, initializing the map if necessary.
func (bm *baseMessage) SetMetadata(key string, value interface{}) {
	if bm.Metadata == nil {
		bm.Metadata = make(map[string]interface{})
	}
	bm.Metadata[key] = value
}

// GetRole returns the role of the message sender.
func (bm *baseMessage) GetRole() MessageRole {
	return bm.Role
}

// GetModel returns the model ID associated with this message.
func (bm *baseMessage) GetModel() model.ModelID {
	return bm.Model
}

// SetModel sets the model ID for this message.
func (bm *baseMessage) SetModel(modelID model.ModelID) {
	bm.Model = modelID
}

// newBaseMessage creates a new base message with the specified source and role.
func newBaseMessage(source MessageSource, role MessageRole) baseMessage {
	return baseMessage{
		Source:    source,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
		Role:      role,
	}
}
