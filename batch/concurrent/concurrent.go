// Package concurrent provides a fallback batch.Processor that runs requests
// concurrently against existing [llm.LLM] and/or [embeddings.Embedding] clients.
// Use this when the underlying provider has no native batch API.
package concurrent

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/joakimcarlsson/ai/batch"
	"github.com/joakimcarlsson/ai/embeddings"
	"github.com/joakimcarlsson/ai/llm"
)

// Options configures the concurrent batch processor.
type Options struct {
	llmClient        llm.LLM
	embeddingClient  embeddings.Embedding
	maxConcurrency   int
	progressCallback batch.ProgressCallback
}

// Option configures Options.
type Option func(*Options)

// WithLLM sets an existing LLM client. Required for chat requests.
func WithLLM(
	client llm.LLM,
) Option {
	return func(o *Options) { o.llmClient = client }
}

// WithEmbedding sets an existing embedding client. Required for embedding requests.
func WithEmbedding(client embeddings.Embedding) Option {
	return func(o *Options) { o.embeddingClient = client }
}

// WithMaxConcurrency sets the maximum number of concurrent requests (0 = unbounded).
func WithMaxConcurrency(
	n int,
) Option {
	return func(o *Options) { o.maxConcurrency = n }
}

// WithProgressCallback sets a callback invoked with progress updates.
func WithProgressCallback(fn batch.ProgressCallback) Option {
	return func(o *Options) { o.progressCallback = fn }
}

// Processor is the concurrent fallback implementation of batch.Processor.
type Processor struct {
	options Options
}

// NewProcessor constructs a concurrent batch processor.
func NewProcessor(opts ...Option) batch.Processor {
	options := Options{maxConcurrency: 10}
	for _, o := range opts {
		o(&options)
	}
	return &Processor{options: options}
}

// Process runs all requests concurrently and waits for completion.
func (p *Processor) Process(
	ctx context.Context,
	requests []batch.Request,
) (*batch.Response, error) {
	if len(requests) == 0 {
		return &batch.Response{Results: []batch.Result{}, Total: 0}, nil
	}
	batch.AssignIDs(requests)

	results := make([]batch.Result, len(requests))
	var wg sync.WaitGroup
	var sem chan struct{}
	var completed atomic.Int64
	var failed atomic.Int64

	if p.options.maxConcurrency > 0 {
		sem = make(chan struct{}, p.options.maxConcurrency)
	}

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, r batch.Request) {
			defer wg.Done()

			if sem != nil {
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					results[idx] = batch.Result{
						ID:    r.ID,
						Index: idx,
						Err:   ctx.Err(),
					}
					failed.Add(1)
					return
				}
			}

			result := batch.Result{ID: r.ID, Index: idx}
			p.runRequest(ctx, &result, r)

			results[idx] = result

			if result.Err != nil {
				failed.Add(1)
			} else {
				completed.Add(1)
			}

			if p.options.progressCallback != nil {
				p.options.progressCallback(batch.Progress{
					Total:     len(requests),
					Completed: int(completed.Load()),
					Failed:    int(failed.Load()),
					Status:    "processing",
				})
			}
		}(i, req)
	}

	wg.Wait()

	return &batch.Response{
		Results:   results,
		Completed: int(completed.Load()),
		Failed:    int(failed.Load()),
		Total:     len(requests),
	}, nil
}

// ProcessAsync runs requests concurrently and emits events as they complete.
func (p *Processor) ProcessAsync(
	ctx context.Context,
	requests []batch.Request,
) (<-chan batch.Event, error) {
	ch := make(chan batch.Event, len(requests)+1)

	if len(requests) == 0 {
		ch <- batch.Event{
			Type:     batch.EventComplete,
			Progress: &batch.Progress{Total: 0, Status: "complete"},
		}
		close(ch)
		return ch, nil
	}
	batch.AssignIDs(requests)

	go func() {
		defer close(ch)

		results := make([]batch.Result, len(requests))
		var wg sync.WaitGroup
		var sem chan struct{}
		var completed atomic.Int64
		var failed atomic.Int64

		if p.options.maxConcurrency > 0 {
			sem = make(chan struct{}, p.options.maxConcurrency)
		}

		for i, req := range requests {
			wg.Add(1)
			go func(idx int, r batch.Request) {
				defer wg.Done()

				if sem != nil {
					select {
					case sem <- struct{}{}:
						defer func() { <-sem }()
					case <-ctx.Done():
						result := batch.Result{
							ID:    r.ID,
							Index: idx,
							Err:   ctx.Err(),
						}
						results[idx] = result
						failed.Add(1)
						ch <- batch.Event{Type: batch.EventItem, Result: &result}
						return
					}
				}

				result := batch.Result{ID: r.ID, Index: idx}
				p.runRequest(ctx, &result, r)

				results[idx] = result

				if result.Err != nil {
					failed.Add(1)
				} else {
					completed.Add(1)
				}

				ch <- batch.Event{Type: batch.EventItem, Result: &result}
				ch <- batch.Event{
					Type: batch.EventProgress,
					Progress: &batch.Progress{
						Total:     len(requests),
						Completed: int(completed.Load()),
						Failed:    int(failed.Load()),
						Status:    "processing",
					},
				}
			}(i, req)
		}

		wg.Wait()

		ch <- batch.Event{
			Type: batch.EventComplete,
			Progress: &batch.Progress{
				Total:     len(requests),
				Completed: int(completed.Load()),
				Failed:    int(failed.Load()),
				Status:    "complete",
			},
		}
	}()

	return ch, nil
}

func (p *Processor) runRequest(
	ctx context.Context,
	result *batch.Result,
	r batch.Request,
) {
	switch r.Type {
	case batch.RequestTypeChat:
		if p.options.llmClient == nil {
			result.Err = batch.ErrNoLLMClient
			return
		}
		resp, err := p.options.llmClient.SendMessages(ctx, r.Messages, r.Tools)
		if err != nil {
			result.Err = err
		} else {
			result.ChatResponse = resp
		}
	case batch.RequestTypeEmbedding:
		if p.options.embeddingClient == nil {
			result.Err = batch.ErrNoEmbeddingClient
			return
		}
		resp, err := p.options.embeddingClient.GenerateEmbeddings(ctx, r.Texts)
		if err != nil {
			result.Err = err
		} else {
			result.EmbedResponse = resp
		}
	}
}
