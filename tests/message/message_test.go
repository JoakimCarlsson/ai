package message

import (
	"encoding/json"
	"testing"

	"github.com/joakimcarlsson/ai/message"
)

func TestNewUserMessage(t *testing.T) {
	m := message.NewUserMessage("hello")
	if m.Role != message.User {
		t.Errorf("expected role User, got %s", m.Role)
	}
	if m.Content().Text != "hello" {
		t.Errorf("expected 'hello', got %q", m.Content().Text)
	}
	if m.CreatedAt == 0 {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestNewSystemMessage(t *testing.T) {
	m := message.NewSystemMessage("be helpful")
	if m.Role != message.System {
		t.Errorf("expected role System, got %s", m.Role)
	}
	if m.Content().Text != "be helpful" {
		t.Errorf(
			"expected 'be helpful', got %q",
			m.Content().Text,
		)
	}
}

func TestNewAssistantMessage(t *testing.T) {
	m := message.NewAssistantMessage()
	if m.Role != message.Assistant {
		t.Errorf("expected role Assistant, got %s", m.Role)
	}
	if len(m.Parts) != 0 {
		t.Errorf("expected 0 parts, got %d", len(m.Parts))
	}
}

func TestNewSummaryMessage(t *testing.T) {
	m := message.NewSummaryMessage("summary text")
	if m.Role != message.Summary {
		t.Errorf("expected role Summary, got %s", m.Role)
	}
	if m.Content().Text != "summary text" {
		t.Errorf(
			"expected 'summary text', got %q",
			m.Content().Text,
		)
	}
}

func TestContent_ReturnsFirstText(t *testing.T) {
	m := message.NewMessage(message.User, []message.ContentPart{
		message.ImageURLContent{URL: "http://img.png"},
		message.TextContent{Text: "first"},
		message.TextContent{Text: "second"},
	})
	if m.Content().Text != "first" {
		t.Errorf(
			"expected 'first', got %q",
			m.Content().Text,
		)
	}
}

func TestContent_NoText(t *testing.T) {
	m := message.NewMessage(message.User, []message.ContentPart{
		message.ImageURLContent{URL: "http://img.png"},
	})
	if m.Content().Text != "" {
		t.Errorf("expected empty text, got %q", m.Content().Text)
	}
}

func TestToolCalls(t *testing.T) {
	m := message.NewMessage(
		message.Assistant,
		[]message.ContentPart{
			message.TextContent{Text: "thinking"},
			message.ToolCall{ID: "1", Name: "a"},
			message.ToolCall{ID: "2", Name: "b"},
		},
	)

	calls := m.ToolCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(calls))
	}
	if calls[0].Name != "a" || calls[1].Name != "b" {
		t.Error("wrong tool call names")
	}
}

func TestToolResults(t *testing.T) {
	m := message.NewMessage(
		message.Tool,
		[]message.ContentPart{
			message.ToolResult{
				ToolCallID: "1",
				Name:       "a",
				Content:    "result",
			},
		},
	)

	results := m.ToolResults()
	if len(results) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(results))
	}
	if results[0].Content != "result" {
		t.Errorf("expected 'result', got %q", results[0].Content)
	}
}

func TestBinaryContentAccessor(t *testing.T) {
	m := message.NewMessage(message.User, []message.ContentPart{
		message.TextContent{Text: "look"},
		message.BinaryContent{
			MIMEType: "image/png",
			Data:     []byte("pixels"),
		},
	})

	bins := m.BinaryContent()
	if len(bins) != 1 {
		t.Fatalf("expected 1 binary, got %d", len(bins))
	}
	if bins[0].MIMEType != "image/png" {
		t.Errorf(
			"expected image/png, got %q",
			bins[0].MIMEType,
		)
	}
}

func TestImageURLContentAccessor(t *testing.T) {
	m := message.NewMessage(message.User, []message.ContentPart{
		message.ImageURLContent{URL: "http://a.png", Detail: "high"},
		message.ImageURLContent{URL: "http://b.png"},
	})

	images := m.ImageURLContent()
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].Detail != "high" {
		t.Errorf(
			"expected detail 'high', got %q",
			images[0].Detail,
		)
	}
}

func TestAppendContent_ExistingText(t *testing.T) {
	m := message.NewUserMessage("hello")
	m.AppendContent(" world")
	if m.Content().Text != "hello world" {
		t.Errorf(
			"expected 'hello world', got %q",
			m.Content().Text,
		)
	}
}

func TestAppendContent_NoExistingText(t *testing.T) {
	m := message.NewAssistantMessage()
	m.AppendContent("new text")
	if m.Content().Text != "new text" {
		t.Errorf(
			"expected 'new text', got %q",
			m.Content().Text,
		)
	}
}

func TestSetToolCalls(t *testing.T) {
	m := message.NewUserMessage("hello")
	m.SetToolCalls([]message.ToolCall{
		{ID: "1", Name: "a"},
		{ID: "2", Name: "b"},
	})

	if len(m.Parts) != 2 {
		t.Fatalf(
			"expected 2 parts after SetToolCalls, got %d",
			len(m.Parts),
		)
	}
	if m.Content().Text != "" {
		t.Error("SetToolCalls should replace all parts")
	}
	if len(m.ToolCalls()) != 2 {
		t.Error("expected 2 tool calls")
	}
}

func TestAppendToolCalls(t *testing.T) {
	m := message.NewUserMessage("hello")
	m.AppendToolCalls([]message.ToolCall{{ID: "1", Name: "a"}})
	if len(m.Parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(m.Parts))
	}
}

func TestSetToolResults(t *testing.T) {
	m := message.NewUserMessage("hello")
	m.SetToolResults([]message.ToolResult{
		{ToolCallID: "1", Content: "ok"},
	})

	if len(m.Parts) != 1 {
		t.Fatalf(
			"expected 1 part after SetToolResults, got %d",
			len(m.Parts),
		)
	}
	results := m.ToolResults()
	if len(results) != 1 || results[0].Content != "ok" {
		t.Error("expected tool result with content 'ok'")
	}
}

func TestAddToolResult(t *testing.T) {
	m := message.NewUserMessage("hi")
	m.AddToolResult(
		message.ToolResult{ToolCallID: "1", Content: "ok"},
	)
	if len(m.Parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(m.Parts))
	}
}

func TestAddImageURL(t *testing.T) {
	m := message.NewAssistantMessage()
	m.AddImageURL("http://img.png", "low")
	images := m.ImageURLContent()
	if len(images) != 1 {
		t.Fatal("expected 1 image")
	}
	if images[0].Detail != "low" {
		t.Errorf("expected detail 'low', got %q", images[0].Detail)
	}
}

func TestAddBinary(t *testing.T) {
	m := message.NewAssistantMessage()
	m.AddBinary("image/jpeg", []byte("data"))
	bins := m.BinaryContent()
	if len(bins) != 1 {
		t.Fatal("expected 1 binary")
	}
	if bins[0].MIMEType != "image/jpeg" {
		t.Errorf(
			"expected image/jpeg, got %q",
			bins[0].MIMEType,
		)
	}
}

func TestJSON_RoundTrip_TextContent(t *testing.T) {
	orig := message.NewUserMessage("hello world")

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded message.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Role != message.User {
		t.Errorf("expected role User, got %s", decoded.Role)
	}
	if decoded.Content().Text != "hello world" {
		t.Errorf(
			"expected 'hello world', got %q",
			decoded.Content().Text,
		)
	}
}

func TestJSON_RoundTrip_ToolCall(t *testing.T) {
	orig := message.NewMessage(
		message.Assistant,
		[]message.ContentPart{
			message.TextContent{Text: "calling tool"},
			message.ToolCall{
				ID:       "tc_1",
				Name:     "search",
				Input:    `{"q":"test"}`,
				Type:     "function",
				Finished: true,
			},
		},
	)

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded message.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	calls := decoded.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].ID != "tc_1" {
		t.Errorf("expected ID 'tc_1', got %q", calls[0].ID)
	}
	if !calls[0].Finished {
		t.Error("expected Finished=true")
	}
}

func TestJSON_RoundTrip_ToolResult(t *testing.T) {
	orig := message.NewMessage(
		message.Tool,
		[]message.ContentPart{
			message.ToolResult{
				ToolCallID: "tc_1",
				Name:       "search",
				Content:    "found it",
				IsError:    true,
			},
		},
	)

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded message.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	results := decoded.ToolResults()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "found it" {
		t.Errorf(
			"expected 'found it', got %q",
			results[0].Content,
		)
	}
	if !results[0].IsError {
		t.Error("expected IsError=true to survive round-trip")
	}
}

func TestJSON_RoundTrip_ImageURL(t *testing.T) {
	orig := message.NewMessage(message.User, []message.ContentPart{
		message.ImageURLContent{
			URL:    "http://example.com/img.png",
			Detail: "high",
		},
	})

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded message.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	images := decoded.ImageURLContent()
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].URL != "http://example.com/img.png" {
		t.Errorf("wrong URL: %q", images[0].URL)
	}
	if images[0].Detail != "high" {
		t.Errorf(
			"expected detail 'high', got %q",
			images[0].Detail,
		)
	}
}

func TestJSON_RoundTrip_BinaryContent(t *testing.T) {
	orig := message.NewMessage(message.User, []message.ContentPart{
		message.BinaryContent{
			MIMEType: "image/png",
			Data:     []byte("raw pixels"),
		},
	})

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded message.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	bins := decoded.BinaryContent()
	if len(bins) != 1 {
		t.Fatalf("expected 1 binary, got %d", len(bins))
	}
	if bins[0].MIMEType != "image/png" {
		t.Errorf("wrong mime: %q", bins[0].MIMEType)
	}
	if string(bins[0].Data) != "raw pixels" {
		t.Errorf("wrong data: %q", string(bins[0].Data))
	}
}

func TestJSON_RoundTrip_MixedParts(t *testing.T) {
	orig := message.NewMessage(
		message.Assistant,
		[]message.ContentPart{
			message.TextContent{Text: "here is the result"},
			message.ToolCall{
				ID:    "tc_1",
				Name:  "search",
				Input: `{"q":"ai"}`,
			},
			message.ImageURLContent{URL: "http://img.png"},
		},
	)

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded message.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(decoded.Parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(decoded.Parts))
	}
	if decoded.Content().Text != "here is the result" {
		t.Error("text content mismatch")
	}
	if len(decoded.ToolCalls()) != 1 {
		t.Error("expected 1 tool call")
	}
	if len(decoded.ImageURLContent()) != 1 {
		t.Error("expected 1 image")
	}
}

func TestJSON_UnknownPartTypeSkipped(t *testing.T) {
	raw := `{
		"role": "user",
		"parts": [
			{"type": "text", "data": {"text": "hello"}},
			{"type": "alien_type", "data": {"foo": "bar"}},
			{"type": "text", "data": {"text": "world"}}
		],
		"created_at": 123
	}`

	var m message.Message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(m.Parts) != 2 {
		t.Errorf(
			"expected 2 parts (unknown skipped), got %d",
			len(m.Parts),
		)
	}
}

func TestJSON_PreservesCreatedAt(t *testing.T) {
	orig := message.NewUserMessage("test")

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded message.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.CreatedAt != orig.CreatedAt {
		t.Errorf(
			"CreatedAt mismatch: %d vs %d",
			decoded.CreatedAt,
			orig.CreatedAt,
		)
	}
}

func TestSource_String(t *testing.T) {
	s := message.Source{Type: "api", ID: "123"}
	if s.String() != "api:123" {
		t.Errorf("expected 'api:123', got %q", s.String())
	}

	s2 := message.Source{Type: "api"}
	if s2.String() != "api" {
		t.Errorf("expected 'api', got %q", s2.String())
	}
}

func TestNewSource_GeneratesID(t *testing.T) {
	s := message.NewSource("test", "")
	if s.ID == "" {
		t.Error("expected generated ID")
	}
}

func TestNewSource_UsesProvidedID(t *testing.T) {
	s := message.NewSource("test", "custom")
	if s.ID != "custom" {
		t.Errorf("expected 'custom', got %q", s.ID)
	}
}
