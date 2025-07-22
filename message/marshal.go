package message

import (
	"encoding/json"
	"fmt"
)

type MessageEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// MarshalMessage converts a BaseMessage to JSON bytes with type envelope
func MarshalMessage(msg BaseMessage) ([]byte, error) {
	var msgType string
	var data []byte
	var err error

	switch m := msg.(type) {
	case *TextMessage:
		msgType = "text"
		data, err = json.Marshal(m)
	case *MultiModalMessage:
		msgType = "multimodal"
		data, err = json.Marshal(m)
	default:
		return nil, fmt.Errorf("unknown message type: %T", msg)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	envelope := MessageEnvelope{
		Type: msgType,
		Data: data,
	}

	return json.Marshal(envelope)
}

// UnmarshalMessage converts JSON bytes back to a BaseMessage instance
func UnmarshalMessage(data []byte) (BaseMessage, error) {
	var envelope MessageEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message envelope: %w", err)
	}

	switch envelope.Type {
	case "text":
		var msg TextMessage
		if err := json.Unmarshal(envelope.Data, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal text message: %w", err)
		}
		return &msg, nil
	case "multimodal":
		var msg MultiModalMessage
		if err := json.Unmarshal(envelope.Data, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal multimodal message: %w", err)
		}
		return &msg, nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", envelope.Type)
	}
}

// MarshalMessages converts a slice of BaseMessage to JSON bytes
func MarshalMessages(messages []BaseMessage) ([]byte, error) {
	envelopes := make([]MessageEnvelope, len(messages))

	for i, msg := range messages {
		var msgType string
		var data []byte
		var err error

		switch m := msg.(type) {
		case *TextMessage:
			msgType = "text"
			data, err = json.Marshal(m)
		case *MultiModalMessage:
			msgType = "multimodal"
			data, err = json.Marshal(m)
		default:
			return nil, fmt.Errorf("unknown message type at index %d: %T", i, msg)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to marshal message at index %d: %w", i, err)
		}

		envelopes[i] = MessageEnvelope{
			Type: msgType,
			Data: data,
		}
	}

	return json.Marshal(envelopes)
}

// UnmarshalMessages converts JSON bytes back to a slice of BaseMessage
func UnmarshalMessages(data []byte) ([]BaseMessage, error) {
	var envelopes []MessageEnvelope
	if err := json.Unmarshal(data, &envelopes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message envelopes: %w", err)
	}

	messages := make([]BaseMessage, len(envelopes))

	for i, envelope := range envelopes {
		switch envelope.Type {
		case "text":
			var msg TextMessage
			if err := json.Unmarshal(envelope.Data, &msg); err != nil {
				return nil, fmt.Errorf("failed to unmarshal text message at index %d: %w", i, err)
			}
			messages[i] = &msg
		case "multimodal":
			var msg MultiModalMessage
			if err := json.Unmarshal(envelope.Data, &msg); err != nil {
				return nil, fmt.Errorf("failed to unmarshal multimodal message at index %d: %w", i, err)
			}
			messages[i] = &msg
		default:
			return nil, fmt.Errorf("unknown message type at index %d: %s", i, envelope.Type)
		}
	}

	return messages, nil
}
