package llm

import (
	"context"
	"errors"
	"testing"
)

type quotaErr struct{ terminal bool }

func (quotaErr) Error() string         { return "insufficient_quota" }
func (quotaErr) GetStatusCode() int    { return 429 }
func (quotaErr) GetRetryAfter() string { return "" }
func (e quotaErr) Terminal() bool      { return e.terminal }

// TestExhaustionKeepsTheCause confirms the exhaustion error still lets
// errors.As find the original typed cause.
func TestExhaustionKeepsTheCause(t *testing.T) {
	cause := quotaErr{}
	cfg := RetryConfig{MaxRetries: 2, RetryStatusCodes: []int{429}}

	_, _, retryErr := ShouldRetry(3, cause, cfg)
	if retryErr == nil {
		t.Fatal("ShouldRetry() past MaxRetries returned no error")
	}
	var got quotaErr
	if !errors.As(retryErr, &got) {
		t.Fatalf("the cause did not survive exhaustion: %v", retryErr)
	}
}

// TestATerminalErrorIsNotRetried confirms a terminal error stops
// ExecuteWithRetry after one call and returns the cause, while an otherwise
// identical non-terminal error is retried to the cap.
func TestATerminalErrorIsNotRetried(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:       8,
		BaseBackoffMs:    1,
		RetryStatusCodes: []int{429},
	}

	calls := 0
	_, err := ExecuteWithRetry(context.Background(), cfg, func() (int, error) {
		calls++
		return 0, quotaErr{terminal: true}
	})
	if calls != 1 {
		t.Errorf(
			"a terminal error ran the operation %d times, want 1; the backoff schedule is being spent on a refusal that cannot change",
			calls,
		)
	}
	var got quotaErr
	if !errors.As(err, &got) {
		t.Errorf("the caller did not get the cause back: %v", err)
	}

	calls = 0
	_, _ = ExecuteWithRetry(context.Background(), cfg, func() (int, error) {
		calls++
		return 0, quotaErr{terminal: false}
	})
	if calls != cfg.MaxRetries+1 {
		t.Errorf(
			"an ordinary retryable 429 ran %d times, want %d; the terminal check is too broad",
			calls,
			cfg.MaxRetries+1,
		)
	}
}
