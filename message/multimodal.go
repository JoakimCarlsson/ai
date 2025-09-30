package message

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeImage    ContentType = "image"
	ContentTypeBinary   ContentType = "binary"
	ContentTypeImageURL ContentType = "image_url"
)

type MultiModalContent struct {
	Type     ContentType `json:"type"`
	Text     string      `json:"text,omitempty"`
	ImageURL string      `json:"image_url,omitempty"`
	MIMEType string      `json:"mime_type,omitempty"`
	Data     []byte      `json:"data,omitempty"`
	Detail   string      `json:"detail,omitempty"`
}

// NewTextContent creates a text content part for multimodal messages
func NewTextContent(text string) MultiModalContent {
	return MultiModalContent{
		Type: ContentTypeText,
		Text: text,
	}
}

// NewImageURLContent creates an image URL content part with optional detail level
func NewImageURLContent(url, detail string) MultiModalContent {
	return MultiModalContent{
		Type:     ContentTypeImageURL,
		ImageURL: url,
		Detail:   detail,
	}
}

// NewBinaryContent creates a binary content part with MIME type and data
func NewBinaryContent(mimeType string, data []byte) MultiModalContent {
	return MultiModalContent{
		Type:     ContentTypeBinary,
		MIMEType: mimeType,
		Data:     data,
	}
}

func (mmc MultiModalContent) String() string {
	switch mmc.Type {
	case ContentTypeText:
		return mmc.Text
	case ContentTypeImageURL:
		return mmc.ImageURL
	case ContentTypeBinary:
		if len(mmc.Data) > 0 {
			return base64.StdEncoding.EncodeToString(mmc.Data)
		}
		return ""
	default:
		return ""
	}
}

func (mmc MultiModalContent) GetDataURL(provider model.ModelProvider) string {
	if mmc.Type == ContentTypeBinary && len(mmc.Data) > 0 {
		base64Encoded := base64.StdEncoding.EncodeToString(mmc.Data)
		if provider == model.ProviderOpenAI {
			return "data:" + mmc.MIMEType + ";base64," + base64Encoded
		}
		return base64Encoded
	}
	return ""
}

type MultiModalMessage struct {
	baseMessage
	Contents []MultiModalContent `json:"contents"`
}

// NewMultiModalMessage creates a multimodal message with source, role and contents
func NewMultiModalMessage(
	source MessageSource,
	role MessageRole,
	contents []MultiModalContent,
) *MultiModalMessage {
	return &MultiModalMessage{
		baseMessage: newBaseMessage(source, role),
		Contents:    contents,
	}
}

// NewUserMultiModalMessage creates a user multimodal message with the given contents
func NewUserMultiModalMessage(contents []MultiModalContent) *MultiModalMessage {
	source := NewMessageSource("user", "")
	return NewMultiModalMessage(source, User, contents)
}

// NewUserMultiModalMessageWithText creates a user multimodal message with text content
func NewUserMultiModalMessageWithText(text string) *MultiModalMessage {
	contents := []MultiModalContent{NewTextContent(text)}
	return NewUserMultiModalMessage(contents)
}

// NewUserMultiModalMessageWithAttachments creates a user message with text and file attachments
func NewUserMultiModalMessageWithAttachments(
	text string,
	attachments []Attachment,
) *MultiModalMessage {
	contents := []MultiModalContent{NewTextContent(text)}
	for _, attachment := range attachments {
		contents = append(
			contents,
			NewBinaryContent(attachment.MIMEType, attachment.Data),
		)
	}
	return NewUserMultiModalMessage(contents)
}

// GetContent returns the contents as an interface for BaseMessage compatibility
func (mmm *MultiModalMessage) GetContent() interface{} {
	return mmm.Contents
}

// GetContents returns all multimodal contents in the message
func (mmm *MultiModalMessage) GetContents() []MultiModalContent {
	return mmm.Contents
}

// GetTextContent returns the first text content or empty string if none found
func (mmm *MultiModalMessage) GetTextContent() string {
	for _, content := range mmm.Contents {
		if content.Type == ContentTypeText {
			return content.Text
		}
	}
	return ""
}

// GetImageURLContents returns all image URL contents from the message
func (mmm *MultiModalMessage) GetImageURLContents() []MultiModalContent {
	var images []MultiModalContent
	for _, content := range mmm.Contents {
		if content.Type == ContentTypeImageURL {
			images = append(images, content)
		}
	}
	return images
}

// GetBinaryContents returns all binary contents from the message
func (mmm *MultiModalMessage) GetBinaryContents() []MultiModalContent {
	var binaries []MultiModalContent
	for _, content := range mmm.Contents {
		if content.Type == ContentTypeBinary {
			binaries = append(binaries, content)
		}
	}
	return binaries
}

// AddContent appends a new content part to the message
func (mmm *MultiModalMessage) AddContent(content MultiModalContent) {
	mmm.Contents = append(mmm.Contents, content)
}

// AddTextContent adds a text content part to the message
func (mmm *MultiModalMessage) AddTextContent(text string) {
	mmm.AddContent(NewTextContent(text))
}

// AddImageURL adds an image URL content part to the message
func (mmm *MultiModalMessage) AddImageURL(url, detail string) {
	mmm.AddContent(NewImageURLContent(url, detail))
}

// AddBinary adds binary content to the message with the specified MIME type
func (mmm *MultiModalMessage) AddBinary(mimeType string, data []byte) {
	mmm.AddContent(NewBinaryContent(mimeType, data))
}

// AppendTextContent adds text to existing text content or creates new text content
func (mmm *MultiModalMessage) AppendTextContent(delta string) {
	for i, content := range mmm.Contents {
		if content.Type == ContentTypeText {
			mmm.Contents[i].Text += delta
			return
		}
	}
	mmm.AddTextContent(delta)
}

type multiModalMessageJSON struct {
	Source    MessageSource          `json:"source"`
	CreatedAt int64                  `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
	Role      MessageRole            `json:"role"`
	Model     model.ModelID          `json:"model"`
	Contents  []MultiModalContent    `json:"contents"`
	Type      string                 `json:"type"`
}

func (mmm *MultiModalMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(multiModalMessageJSON{
		Source:    mmm.Source,
		CreatedAt: mmm.CreatedAt.Unix(),
		Metadata:  mmm.Metadata,
		Role:      mmm.Role,
		Model:     mmm.Model,
		Contents:  mmm.Contents,
		Type:      "multimodal",
	})
}

func (mmm *MultiModalMessage) UnmarshalJSON(data []byte) error {
	var mmmJSON multiModalMessageJSON
	if err := json.Unmarshal(data, &mmmJSON); err != nil {
		return err
	}

	mmm.Source = mmmJSON.Source
	mmm.CreatedAt = time.Unix(mmmJSON.CreatedAt, 0)
	mmm.Metadata = mmmJSON.Metadata
	mmm.Role = mmmJSON.Role
	mmm.Model = mmmJSON.Model
	mmm.Contents = mmmJSON.Contents

	return nil
}
