package batch

import (
	"context"
	"sync"
	"sync/atomic"
)

type concurrentExecutor struct{}

func (e *concurrentExecutor) execute(
	ctx context.Context,
	requests []Request,
	opts processorOptions,
) (*Response, error) {
	results := make([]Result, len(requests))
	var wg sync.WaitGroup
	var sem chan struct{}
	var completed atomic.Int64
	var failed atomic.Int64

	concurrency := opts.maxConcurrency
	if concurrency > 0 {
		sem = make(chan struct{}, concurrency)
	}

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, r Request) {
			defer wg.Done()

			if sem != nil {
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					results[idx] = Result{
						ID:    r.ID,
						Index: idx,
						Err:   ctx.Err(),
					}
					failed.Add(1)
					return
				}
			}

			result := Result{ID: r.ID, Index: idx}

			switch r.Type {
			case RequestTypeChat:
				if opts.llmClient == nil {
					result.Err = ErrNoLLMClient
				} else {
					resp, err := opts.llmClient.SendMessages(ctx, r.Messages, r.Tools)
					if err != nil {
						result.Err = err
					} else {
						result.ChatResponse = resp
					}
				}
			case RequestTypeEmbedding:
				if opts.embeddingClient == nil {
					result.Err = ErrNoEmbeddingClient
				} else {
					resp, err := opts.embeddingClient.GenerateEmbeddings(ctx, r.Texts)
					if err != nil {
						result.Err = err
					} else {
						result.EmbedResponse = resp
					}
				}
			}

			results[idx] = result

			if result.Err != nil {
				failed.Add(1)
			} else {
				completed.Add(1)
			}

			if opts.progressCallback != nil {
				opts.progressCallback(Progress{
					Total:     len(requests),
					Completed: int(completed.Load()),
					Failed:    int(failed.Load()),
					Status:    "processing",
				})
			}
		}(i, req)
	}

	wg.Wait()

	return &Response{
		Results:   results,
		Completed: int(completed.Load()),
		Failed:    int(failed.Load()),
		Total:     len(requests),
	}, nil
}

func (e *concurrentExecutor) executeAsync(
	ctx context.Context,
	requests []Request,
	opts processorOptions,
) (<-chan Event, error) {
	ch := make(chan Event, len(requests)+1)

	go func() {
		defer close(ch)

		results := make([]Result, len(requests))
		var wg sync.WaitGroup
		var sem chan struct{}
		var completed atomic.Int64
		var failed atomic.Int64

		concurrency := opts.maxConcurrency
		if concurrency > 0 {
			sem = make(chan struct{}, concurrency)
		}

		for i, req := range requests {
			wg.Add(1)
			go func(idx int, r Request) {
				defer wg.Done()

				if sem != nil {
					select {
					case sem <- struct{}{}:
						defer func() { <-sem }()
					case <-ctx.Done():
						result := Result{ID: r.ID, Index: idx, Err: ctx.Err()}
						results[idx] = result
						failed.Add(1)
						ch <- Event{Type: EventItem, Result: &result}
						return
					}
				}

				result := Result{ID: r.ID, Index: idx}

				switch r.Type {
				case RequestTypeChat:
					if opts.llmClient == nil {
						result.Err = ErrNoLLMClient
					} else {
						resp, err := opts.llmClient.SendMessages(ctx, r.Messages, r.Tools)
						if err != nil {
							result.Err = err
						} else {
							result.ChatResponse = resp
						}
					}
				case RequestTypeEmbedding:
					if opts.embeddingClient == nil {
						result.Err = ErrNoEmbeddingClient
					} else {
						resp, err := opts.embeddingClient.GenerateEmbeddings(ctx, r.Texts)
						if err != nil {
							result.Err = err
						} else {
							result.EmbedResponse = resp
						}
					}
				}

				results[idx] = result

				if result.Err != nil {
					failed.Add(1)
				} else {
					completed.Add(1)
				}

				ch <- Event{Type: EventItem, Result: &result}
				ch <- Event{
					Type: EventProgress,
					Progress: &Progress{
						Total:     len(requests),
						Completed: int(completed.Load()),
						Failed:    int(failed.Load()),
						Status:    "processing",
					},
				}
			}(i, req)
		}

		wg.Wait()

		ch <- Event{
			Type: EventComplete,
			Progress: &Progress{
				Total:     len(requests),
				Completed: int(completed.Load()),
				Failed:    int(failed.Load()),
				Status:    "complete",
			},
		}
	}()

	return ch, nil
}
