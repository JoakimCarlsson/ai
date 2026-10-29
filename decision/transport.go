package decision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError is returned when a decision API responds with a non-200 status. It
// carries the status code and Retry-After header so retry helpers such as
// llm.ShouldRetry can treat 429 and 529 responses as retryable.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// RetryAfter is the raw Retry-After header, empty if absent.
	RetryAfter string
	// Body is the response body.
	Body string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf(
		"decision API request failed with status %d: %s",
		e.StatusCode,
		e.Body,
	)
}

// GetStatusCode returns the HTTP status code of the failed response.
func (e *APIError) GetStatusCode() int { return e.StatusCode }

// GetRetryAfter returns the Retry-After header of the failed response.
func (e *APIError) GetRetryAfter() string { return e.RetryAfter }

// PostJSON sends reqBody as JSON to url with bearer authentication and decodes
// a 200 response into out. A non-200 response is returned as an [*APIError].
// It is the shared HTTP transport for vendor Decider implementations, which
// differ only in their request/response shapes.
func PostJSON(
	ctx context.Context,
	httpClient *http.Client,
	url, apiKey string,
	reqBody, out any,
) error {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal decision request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("failed to create decision request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make decision request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read decision response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &APIError{
			StatusCode: resp.StatusCode,
			RetryAfter: resp.Header.Get("Retry-After"),
			Body:       string(body),
		}
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to unmarshal decision response: %w", err)
	}
	return nil
}
