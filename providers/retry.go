package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/joakimcarlsson/ai/types"
	"github.com/openai/openai-go"
)

type RetryConfig struct {
	MaxRetries       int
	BaseBackoffMs    int
	JitterPercent    float64
	RetryStatusCodes []int
	CheckRetryAfter  bool
}

type RetryableError interface {
	error
	GetStatusCode() int
	GetRetryAfter() string
}

type OpenAIRetryableError struct {
	err *openai.Error
}

func (e OpenAIRetryableError) Error() string {
	return e.err.Error()
}

func (e OpenAIRetryableError) GetStatusCode() int {
	return e.err.StatusCode
}

func (e OpenAIRetryableError) GetRetryAfter() string {
	if e.err.Response != nil {
		retryAfterValues := e.err.Response.Header.Values("Retry-After")
		if len(retryAfterValues) > 0 {
			return retryAfterValues[0]
		}
	}
	return ""
}

type AnthropicRetryableError struct {
	err *anthropic.Error
}

func (e AnthropicRetryableError) Error() string {
	return e.err.Error()
}

func (e AnthropicRetryableError) GetStatusCode() int {
	return e.err.StatusCode
}

func (e AnthropicRetryableError) GetRetryAfter() string {
	if e.err.Response != nil {
		retryAfterValues := e.err.Response.Header.Values("Retry-After")
		if len(retryAfterValues) > 0 {
			return retryAfterValues[0]
		}
	}
	return ""
}

type GenericRetryableError struct {
	err        error
	statusCode int
}

func (e GenericRetryableError) Error() string {
	return e.err.Error()
}

func (e GenericRetryableError) GetStatusCode() int {
	return e.statusCode
}

func (e GenericRetryableError) GetRetryAfter() string {
	return ""
}

// DefaultRetryConfig provides standard retry settings for most LLM providers
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:       maxRetries,
		BaseBackoffMs:    2000,
		JitterPercent:    0.2,
		RetryStatusCodes: []int{429, 500, 502, 503, 504},
		CheckRetryAfter:  true,
	}
}

// OpenAIRetryConfig provides retry settings optimized for OpenAI API behavior
func OpenAIRetryConfig() RetryConfig {
	config := DefaultRetryConfig()
	config.RetryStatusCodes = []int{429, 500}
	return config
}

// AnthropicRetryConfig provides retry settings optimized for Anthropic API behavior
func AnthropicRetryConfig() RetryConfig {
	config := DefaultRetryConfig()
	config.RetryStatusCodes = []int{429, 529}
	return config
}

// GeminiRetryConfig provides retry settings optimized for Gemini API behavior
func GeminiRetryConfig() RetryConfig {
	config := DefaultRetryConfig()
	config.CheckRetryAfter = false
	return config
}

// ShouldRetry determines if an operation should be retried based on the error and configuration
func ShouldRetry(
	attempts int,
	err error,
	config RetryConfig,
) (bool, int64, error) {
	if attempts > config.MaxRetries {
		return false, 0, fmt.Errorf(
			"maximum retry attempts reached: %d retries",
			config.MaxRetries,
		)
	}

	if errors.Is(err, io.EOF) {
		return false, 0, err
	}

	retryableErr := convertToRetryableError(err)
	if retryableErr == nil {
		return false, 0, err
	}

	if !isRetryableStatusCode(
		retryableErr.GetStatusCode(),
		config.RetryStatusCodes,
	) {
		return false, 0, err
	}

	retryMs := calculateBackoff(attempts, config)

	if config.CheckRetryAfter {
		if retryAfter := retryableErr.GetRetryAfter(); retryAfter != "" {
			if parsedRetryMs, err := parseRetryAfter(retryAfter); err == nil {
				retryMs = parsedRetryMs
			}
		}
	}

	return true, int64(retryMs), nil
}

func convertToRetryableError(err error) RetryableError {
	var openaiErr *openai.Error
	if errors.As(err, &openaiErr) {
		return OpenAIRetryableError{err: openaiErr}
	}

	var anthropicErr *anthropic.Error
	if errors.As(err, &anthropicErr) {
		return AnthropicRetryableError{err: anthropicErr}
	}

	if isGeminiRateLimitError(err) {
		return GenericRetryableError{
			err:        err,
			statusCode: 429,
		}
	}

	return nil
}

func isRetryableStatusCode(statusCode int, retryableCodes []int) bool {
	for _, code := range retryableCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}

func calculateBackoff(attempts int, config RetryConfig) int {
	backoffMs := config.BaseBackoffMs * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * config.JitterPercent)
	return backoffMs + jitterMs
}

func parseRetryAfter(retryAfter string) (int, error) {
	var retryMs int
	if _, err := fmt.Sscanf(retryAfter, "%d", &retryMs); err == nil {
		return retryMs * 1000, nil
	}
	return 0, fmt.Errorf("failed to parse retry-after header: %s", retryAfter)
}

func isGeminiRateLimitError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	rateLimitKeywords := []string{
		"rate limit",
		"quota exceeded",
		"too many requests",
	}

	for _, keyword := range rateLimitKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}

// ExecuteWithRetry runs an operation with automatic retry logic for transient failures
func ExecuteWithRetry[T any](
	ctx context.Context,
	config RetryConfig,
	operation func() (T, error),
) (T, error) {
	var result T
	var err error
	attempts := 0

	for {
		attempts++
		result, err = operation()
		if err == nil {
			return result, nil
		}

		shouldRetry, retryAfterMs, retryErr := ShouldRetry(
			attempts,
			err,
			config,
		)
		if retryErr != nil {
			return result, retryErr
		}

		if !shouldRetry {
			return result, err
		}

		slog.Warn("Retrying operation due to error",
			"attempt", attempts,
			"max_retries", config.MaxRetries,
			"retry_after_ms", retryAfterMs,
			"error", err.Error())

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(time.Duration(retryAfterMs) * time.Millisecond):
			continue
		}
	}
}

// ExecuteStreamWithRetry runs a streaming operation with automatic retry logic for transient failures
func ExecuteStreamWithRetry(
	ctx context.Context,
	config RetryConfig,
	operation func() error,
	eventChan chan<- LLMEvent,
) {
	attempts := 0

	for {
		attempts++
		err := operation()
		if err == nil {
			return
		}

		shouldRetry, retryAfterMs, retryErr := ShouldRetry(
			attempts,
			err,
			config,
		)
		if retryErr != nil {
			eventChan <- LLMEvent{Type: types.EventError, Error: retryErr}
			return
		}

		if !shouldRetry {
			eventChan <- LLMEvent{Type: types.EventError, Error: err}
			return
		}

		slog.Warn("Retrying stream operation due to error",
			"attempt", attempts,
			"max_retries", config.MaxRetries,
			"retry_after_ms", retryAfterMs,
			"error", err.Error())

		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				eventChan <- LLMEvent{Type: types.EventError, Error: ctx.Err()}
			}
			return
		case <-time.After(time.Duration(retryAfterMs) * time.Millisecond):
			continue
		}
	}
}
