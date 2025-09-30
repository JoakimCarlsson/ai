package message

import (
	"encoding/json"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

// TextMessage represents a message containing only text content with extended metadata support.
// It implements the BaseMessage interface and provides JSON serialization capabilities.
type TextMessage struct {
	baseMessage
	// Content contains the text content of the message.
	Content string `json:"content"`
}

// NewTextMessage creates a text message with source, role and content.
func NewTextMessage(
	source MessageSource,
	role MessageRole,
	content string,
) *TextMessage {
	return &TextMessage{
		baseMessage: newBaseMessage(source, role),
		Content:     content,
	}
}

// NewUserTextMessage creates a user text message with the given content.
func NewUserTextMessage(content string) *TextMessage {
	source := NewMessageSource("user", "")
	return NewTextMessage(source, User, content)
}

// NewSystemTextMessage creates a system text message with the given content.
func NewSystemTextMessage(content string) *TextMessage {
	source := NewMessageSource("system", "")
	return NewTextMessage(source, System, content)
}

// NewAssistantTextMessage creates an assistant text message with the given content.
func NewAssistantTextMessage(content string) *TextMessage {
	source := NewMessageSource("assistant", "")
	return NewTextMessage(source, Assistant, content)
}

// GetContent returns the text content as an interface for BaseMessage compatibility.
func (tm *TextMessage) GetContent() interface{} {
	return tm.Content
}

// GetText returns the text content of the message.
func (tm *TextMessage) GetText() string {
	return tm.Content
}

// AppendText adds the given text to the existing content.
func (tm *TextMessage) AppendText(delta string) {
	tm.Content += delta
}

// textMessageJSON is used for JSON serialization of TextMessage.
type textMessageJSON struct {
	Source    MessageSource          `json:"source"`
	CreatedAt int64                  `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
	Role      MessageRole            `json:"role"`
	Model     model.ModelID          `json:"model"`
	Content   string                 `json:"content"`
	Type      string                 `json:"type"`
}

// MarshalJSON implements the json.Marshaler interface for TextMessage.
func (tm *TextMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(textMessageJSON{
		Source:    tm.Source,
		CreatedAt: tm.CreatedAt.Unix(),
		Metadata:  tm.Metadata,
		Role:      tm.Role,
		Model:     tm.Model,
		Content:   tm.Content,
		Type:      "text",
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface for TextMessage.
func (tm *TextMessage) UnmarshalJSON(data []byte) error {
	var tmJSON textMessageJSON
	if err := json.Unmarshal(data, &tmJSON); err != nil {
		return err
	}

	tm.Source = tmJSON.Source
	tm.CreatedAt = time.Unix(tmJSON.CreatedAt, 0)
	tm.Metadata = tmJSON.Metadata
	tm.Role = tmJSON.Role
	tm.Model = tmJSON.Model
	tm.Content = tmJSON.Content

	return nil
}
