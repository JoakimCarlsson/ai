package fim

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Post sends reqBody as JSON to url with bearer authentication, adding the SSE
// Accept header when stream is true. On a non-200 status it reads and closes
// the body and returns an error tagged with provider. On success the caller
// owns resp.Body and must close it. It is the shared HTTP transport for vendor
// FIM implementations, which differ only in their request/response shapes.
func Post(
	ctx context.Context,
	httpClient *http.Client,
	url, apiKey, provider string,
	reqBody any,
	stream bool,
) (*http.Response, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, url, bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf(
			"%s fim api error (status %d): %s",
			provider, resp.StatusCode, string(bodyBytes),
		)
	}
	return resp, nil
}

// StreamChunk is the provider-agnostic content decoded from one SSE data line.
type StreamChunk struct {
	// Delta is the text produced by this chunk, if any.
	Delta string
	// FinishReason, when non-nil, marks why generation stopped.
	FinishReason *FinishReason
	// Usage, when non-nil, carries updated token counts.
	Usage *Usage
}

// StreamSSE reads a Server-Sent Events body, invoking decode for each "data:"
// line and emitting fim.Events on out. It owns content accumulation, [DONE] and
// EOF handling, and error framing; decode returns false to skip a line. FIM
// responses carry a single choice per chunk, so per-chunk deltas are emitted in
// order.
func StreamSSE(
	body io.Reader,
	decode func(data []byte) (StreamChunk, bool),
	out chan<- Event,
) {
	reader := bufio.NewReader(body)
	var content strings.Builder
	var usage Usage
	var finish FinishReason

	complete := func() {
		out <- Event{
			Type: EventComplete,
			Response: &Response{
				Content:      content.String(),
				Usage:        usage,
				FinishReason: finish,
			},
		}
	}

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				complete()
				return
			}
			out <- Event{
				Type:  EventError,
				Error: fmt.Errorf("error reading stream: %w", err),
			}
			return
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			complete()
			return
		}

		chunk, ok := decode(data)
		if !ok {
			continue
		}
		if chunk.Delta != "" {
			content.WriteString(chunk.Delta)
			out <- Event{Type: EventContentDelta, Content: chunk.Delta}
		}
		if chunk.FinishReason != nil {
			finish = *chunk.FinishReason
		}
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}
	}
}
