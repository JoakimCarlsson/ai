package message

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

type MessageRole string

const (
	Assistant MessageRole = "assistant"
	User      MessageRole = "user"
	System    MessageRole = "system"
	Tool      MessageRole = "tool"
)

type Attachment struct {
	MIMEType string
	Data     []byte
}

type FinishReason string

const (
	FinishReasonEndTurn   FinishReason = "end_turn"
	FinishReasonMaxTokens FinishReason = "max_tokens"
	FinishReasonToolUse   FinishReason = "tool_use"
	FinishReasonCanceled  FinishReason = "canceled"
	FinishReasonError     FinishReason = "error"
	FinishReasonUnknown   FinishReason = "unknown"
)

type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Input    string `json:"input"`
	Type     string `json:"type"`
	Finished bool   `json:"finished"`
}

func (ToolCall) isPart() {}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Metadata   string `json:"metadata"`
	IsError    bool   `json:"is_error"`
}

func (ToolResult) isPart() {}

type ContentPart interface {
	isPart()
}

type TextContent struct {
	Text string `json:"text"`
}

func (tc TextContent) String() string {
	return tc.Text
}

func (TextContent) isPart() {}

type ImageURLContent struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

func (iuc ImageURLContent) String() string {
	return iuc.URL
}

func (ImageURLContent) isPart() {}

type BinaryContent struct {
	Path     string
	MIMEType string
	Data     []byte
}

func (bc BinaryContent) String(provider model.ModelProvider) string {
	base64Encoded := base64.StdEncoding.EncodeToString(bc.Data)
	if provider == model.ProviderOpenAI {
		return "data:" + bc.MIMEType + ";base64," + base64Encoded
	}
	return base64Encoded
}

func (BinaryContent) isPart() {}

type Message struct {
	Role      MessageRole
	Parts     []ContentPart
	Model     model.ModelID
	CreatedAt int64
}

// NewMessage creates a new message with the specified role and content parts
func NewMessage(role MessageRole, parts []ContentPart) Message {
	return Message{
		Role:      role,
		Parts:     parts,
		CreatedAt: time.Now().UnixNano(),
	}
}

// NewUserMessage creates a new user message with the given text content
func NewUserMessage(text string) Message {
	return NewMessage(User, []ContentPart{TextContent{Text: text}})
}

// NewSystemMessage creates a new system message with the given text content
func NewSystemMessage(text string) Message {
	return NewMessage(System, []ContentPart{TextContent{Text: text}})
}

// NewAssistantMessage creates a new empty assistant message
func NewAssistantMessage() Message {
	return NewMessage(Assistant, []ContentPart{})
}

// Content returns the first text content part from the message
func (m *Message) Content() TextContent {
	for _, part := range m.Parts {
		if c, ok := part.(TextContent); ok {
			return c
		}
	}
	return TextContent{}
}

// BinaryContent returns all binary content parts from the message
func (m *Message) BinaryContent() []BinaryContent {
	binaryContents := make([]BinaryContent, 0)
	for _, part := range m.Parts {
		if c, ok := part.(BinaryContent); ok {
			binaryContents = append(binaryContents, c)
		}
	}
	return binaryContents
}

// ToolCalls returns all tool call parts from the message
func (m *Message) ToolCalls() []ToolCall {
	var toolCalls []ToolCall
	for _, part := range m.Parts {
		if tc, ok := part.(ToolCall); ok {
			toolCalls = append(toolCalls, tc)
		}
	}
	return toolCalls
}

// ToolResults returns all tool result parts from the message
func (m *Message) ToolResults() []ToolResult {
	var toolResults []ToolResult
	for _, part := range m.Parts {
		if tr, ok := part.(ToolResult); ok {
			toolResults = append(toolResults, tr)
		}
	}
	return toolResults
}

// AppendContent adds text to the existing text content or creates new text content
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

func (m *Message) AppendReasoningContent(delta string) {
}

// SetToolCalls replaces all message parts with the provided tool calls
func (m *Message) SetToolCalls(tc []ToolCall) {
	m.Parts = []ContentPart{}
	for _, call := range tc {
		m.Parts = append(m.Parts, call)
	}
}

// AppendToolCalls adds tool calls to the message without clearing existing content
func (m *Message) AppendToolCalls(tc []ToolCall) {
	for _, call := range tc {
		m.Parts = append(m.Parts, call)
	}
}

// AddToolResult appends a tool result to the message parts
func (m *Message) AddToolResult(tr ToolResult) {
	m.Parts = append(m.Parts, tr)
}

// SetToolResults replaces all message parts with the provided tool results
func (m *Message) SetToolResults(tr []ToolResult) {
	m.Parts = []ContentPart{}
	for _, result := range tr {
		m.Parts = append(m.Parts, result)
	}
}

// AddFinish adds a finish reason to the message
func (m *Message) AddFinish(reason FinishReason) {
}

// AddImageURL adds an image URL content part to the message
func (m *Message) AddImageURL(url, detail string) {
	m.Parts = append(m.Parts, ImageURLContent{URL: url, Detail: detail})
}

// AddBinary adds binary content to the message with the specified MIME type
func (m *Message) AddBinary(mimeType string, data []byte) {
	m.Parts = append(m.Parts, BinaryContent{MIMEType: mimeType, Data: data})
}

type BaseMessage interface {
	GetSource() string
	GetCreatedAt() time.Time
	GetContent() interface{}
	GetMetadata() map[string]interface{}
	SetMetadata(key string, value interface{})
	GetRole() MessageRole
	GetModel() model.ModelID
	SetModel(modelID model.ModelID)
}

type MessageSource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (ms MessageSource) String() string {
	if ms.ID != "" {
		return ms.Type + ":" + ms.ID
	}
	return ms.Type
}

func generateMessageID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// NewMessageSource creates a message source with type and ID, generating ID if empty
func NewMessageSource(sourceType, id string) MessageSource {
	if id == "" {
		id = generateMessageID()
	}
	return MessageSource{Type: sourceType, ID: id}
}

type baseMessage struct {
	Source    MessageSource          `json:"source"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
	Role      MessageRole            `json:"role"`
	Model     model.ModelID          `json:"model"`
}

func (bm *baseMessage) GetSource() string {
	return bm.Source.String()
}

func (bm *baseMessage) GetCreatedAt() time.Time {
	return bm.CreatedAt
}

func (bm *baseMessage) GetMetadata() map[string]interface{} {
	if bm.Metadata == nil {
		bm.Metadata = make(map[string]interface{})
	}
	return bm.Metadata
}

func (bm *baseMessage) SetMetadata(key string, value interface{}) {
	if bm.Metadata == nil {
		bm.Metadata = make(map[string]interface{})
	}
	bm.Metadata[key] = value
}

func (bm *baseMessage) GetRole() MessageRole {
	return bm.Role
}

func (bm *baseMessage) GetModel() model.ModelID {
	return bm.Model
}

func (bm *baseMessage) SetModel(modelID model.ModelID) {
	bm.Model = modelID
}

func newBaseMessage(source MessageSource, role MessageRole) baseMessage {
	return baseMessage{
		Source:    source,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
		Role:      role,
	}
}
