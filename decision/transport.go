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

// PostJSON sends reqBody as JSON to url with bearer authentication and the
// given extra headers, and decodes a 200 response into out. A non-200
// response is returned as an [*APIError]. It is the shared HTTP transport for
// vendor Decider implementations, which differ only in their request/response
// shapes.
func PostJSON(
	ctx context.Context,
	httpClient *http.Client,
	url, apiKey string,
	headers map[string]string,
	reqBody, out any,
) error {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal decision request: %w", err)
	}
	return doJSON(
		ctx, httpClient, http.MethodPost, url, apiKey, headers,
		bytes.NewReader(jsonBody), out,
	)
}

// GetJSON sends a GET to url with bearer authentication and decodes a 200
// response into out. A non-200 response is returned as an [*APIError].
func GetJSON(
	ctx context.Context,
	httpClient *http.Client,
	url, apiKey string,
	out any,
) error {
	return doJSON(ctx, httpClient, http.MethodGet, url, apiKey, nil, nil, out)
}

func doJSON(
	ctx context.Context,
	httpClient *http.Client,
	method, url, apiKey string,
	headers map[string]string,
	body io.Reader,
	out any,
) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create decision request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make decision request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read decision response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &APIError{
			StatusCode: resp.StatusCode,
			RetryAfter: resp.Header.Get("Retry-After"),
			Body:       string(respBody),
		}
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("failed to unmarshal decision response: %w", err)
	}
	return nil
}
