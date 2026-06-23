package core

import (
	"context"
	"runtime"
	"sync"
)

// forEachParallel runs fn over items with a bounded worker pool sized to
// GOMAXPROCS (clamped to len(items) and at least 1). It is the concurrency
// primitive for the per-file artwork, genre, and transcode passes.
//
// ctx cancellation stops dispatching new tasks; in-flight tasks are allowed to
// finish (fn is expected to be quick, or to observe ctx itself, as transcode
// does via exec.CommandContext). The first non-nil error returned by fn cancels
// the pool and is returned, matching the existing engine-level abort semantics
// of the transcode pass; per-file recoverable failures are recorded by fn on
// the Report rather than returned.
func forEachParallel[T any](ctx context.Context, items []T, fn func(context.Context, T) error) error {
	workers := runtime.GOMAXPROCS(0)
	if workers > len(items) {
		workers = len(items)
	}
	if workers < 1 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var (
		mu       sync.Mutex
		firstErr error
	)
	setErr := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

loop:
	for _, it := range items {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}
		wg.Add(1)
		go func(item T) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(ctx, item); err != nil {
				setErr(err)
				cancel()
			}
		}(it)
	}
	wg.Wait()
	return firstErr
}
