package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/joakimcarlsson/ai/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RetryConfig holds parameters for LLM request retry behavior.
type RetryConfig struct {
	MaxRetries       int
	BaseBackoffMs    int
	JitterPercent    float64
	RetryStatusCodes []int
	CheckRetryAfter  bool
}

// RetryableError marks an error as retryable and exposes the HTTP status code
// plus optional Retry-After header. Vendor packages wrap their SDK errors in a
// type that satisfies this interface; [ShouldRetry] dispatches via [errors.As]
// so the modality core does not depend on any vendor SDK.
type RetryableError interface {
	error
	GetStatusCode() int
	GetRetryAfter() string
}

// GenericRetryableError marks an error retryable with a fixed HTTP status code.
type GenericRetryableError struct {
	Err        error
	StatusCode int
}

// Error implements the error interface.
func (e GenericRetryableError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

// Unwrap exposes the wrapped error for errors.Is / errors.As.
func (e GenericRetryableError) Unwrap() error { return e.Err }

// GetStatusCode returns the status code associated with this retryable error.
func (e GenericRetryableError) GetStatusCode() int { return e.StatusCode }

// GetRetryAfter returns empty; generic errors do not carry Retry-After.
func (e GenericRetryableError) GetRetryAfter() string { return "" }

// DefaultRetryConfig provides standard retry settings for most LLM providers.
// Vendor packages typically derive their own RetryConfig from this and tweak
// the RetryStatusCodes list.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:       MaxRetries,
		BaseBackoffMs:    2000,
		JitterPercent:    0.2,
		RetryStatusCodes: []int{429, 500, 502, 503, 504},
		CheckRetryAfter:  true,
	}
}

// ShouldRetry determines if an operation should be retried based on the error
// and configuration. The error is matched against [RetryableError] via
// [errors.As], so vendor packages wrap their SDK errors in a type that
// satisfies the interface.
func ShouldRetry(attempts int, err error, config RetryConfig) (bool, int64, error) {
	if attempts > config.MaxRetries {
		return false, 0, fmt.Errorf(
			"maximum retry attempts reached: %d retries",
			config.MaxRetries,
		)
	}

	if errors.Is(err, io.EOF) {
		return false, 0, err
	}

	var retryable RetryableError
	if !errors.As(err, &retryable) {
		return false, 0, err
	}

	if !isRetryableStatusCode(retryable.GetStatusCode(), config.RetryStatusCodes) {
		return false, 0, err
	}

	retryMs := calculateBackoff(attempts, config)

	if config.CheckRetryAfter {
		if retryAfter := retryable.GetRetryAfter(); retryAfter != "" {
			if parsedRetryMs, err := parseRetryAfter(retryAfter); err == nil {
				retryMs = parsedRetryMs
			}
		}
	}

	return true, int64(retryMs), nil
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

// ExecuteWithRetry runs an operation with automatic retry logic for transient failures.
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

		shouldRetry, retryAfterMs, retryErr := ShouldRetry(attempts, err, config)
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

		span := trace.SpanFromContext(ctx)
		span.AddEvent("retry", trace.WithAttributes(
			attribute.Int("attempt", attempts),
			attribute.Int64("retry_after_ms", retryAfterMs),
			attribute.String("error", err.Error()),
		))

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(time.Duration(retryAfterMs) * time.Millisecond):
			continue
		}
	}
}

// ExecuteStreamWithRetry runs a streaming operation with automatic retry logic for transient failures.
func ExecuteStreamWithRetry(
	ctx context.Context,
	config RetryConfig,
	operation func() error,
	eventChan chan<- Event,
) {
	attempts := 0

	for {
		attempts++
		err := operation()
		if err == nil {
			return
		}

		shouldRetry, retryAfterMs, retryErr := ShouldRetry(attempts, err, config)
		if retryErr != nil {
			eventChan <- Event{Type: types.EventError, Error: retryErr}
			return
		}

		if !shouldRetry {
			eventChan <- Event{Type: types.EventError, Error: err}
			return
		}

		slog.Warn("Retrying stream operation due to error",
			"attempt", attempts,
			"max_retries", config.MaxRetries,
			"retry_after_ms", retryAfterMs,
			"error", err.Error())

		span := trace.SpanFromContext(ctx)
		span.AddEvent("retry", trace.WithAttributes(
			attribute.Int("attempt", attempts),
			attribute.Int64("retry_after_ms", retryAfterMs),
			attribute.String("error", err.Error()),
		))

		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				eventChan <- Event{Type: types.EventError, Error: ctx.Err()}
			}
			return
		case <-time.After(time.Duration(retryAfterMs) * time.Millisecond):
			continue
		}
	}
}
