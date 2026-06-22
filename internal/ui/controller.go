package ui

import (
	"context"
	"errors"
	"sync"

	"github.com/colinstark/coverfixer/internal/core"
)

// runFunc is the engine entry point, injectable for tests.
type runFunc func(context.Context, core.Options, func(core.Event)) (core.Report, error)

// Controller owns the run lifecycle: it is the single-flight guard against
// overlapping runs, owns the cancelable context, and streams progress plus a
// terminal completion event to the frontend through an Emitter. It has no
// dependency on any GUI or Wails type, so its semantics are unit-testable
// headless with a fake emitter and a fake engine.
type Controller struct {
	em  Emitter
	run runFunc

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	done    chan struct{} // closed when the current run completes; nil when idle
}

// NewController returns a Controller that drives the real core.Run engine.
func NewController(em Emitter) *Controller {
	return &Controller{em: em, run: core.Run}
}

// newControllerWithRun is the test seam for injecting a fake engine.
func newControllerWithRun(em Emitter, run runFunc) *Controller {
	return &Controller{em: em, run: run}
}

// Start validates the request, kicks off a run in a goroutine, and returns
// immediately. It streams cf:progress lines as work happens, then emits a
// terminal cf:done (with a summary) on success or cancellation, or cf:error on
// an engine-level failure. A cf:state(true|false) event brackets the run so
// the frontend can toggle control enablement.
//
// It returns an error synchronously only when a run is already in flight or
// the request is invalid; engine-level errors arrive later via cf:error.
func (c *Controller) Start(req RunRequest) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return errors.New("a run is already in progress")
	}
	opts, err := req.Options()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	c.running = true
	c.done = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.mu.Unlock()

	c.em.Emit(EventState, true)

	go func() {
		defer c.finish()
		report, runErr := c.run(ctx, opts, func(e core.Event) {
			c.em.Emit(EventProgress, formatEvent(e))
		})
		switch {
		case runErr == nil:
			c.em.Emit(EventDone, formatReport(report, opts.DryRun))
		case errors.Is(runErr, context.Canceled):
			// The engine returns the work done so far on cancellation; surface
			// it rather than leaving the summary blank, and never present
			// cancellation itself as an error.
			c.em.Emit(EventDone, "Run cancelled — partial results:\n\n"+formatReport(report, opts.DryRun))
		default:
			c.em.Emit(EventError, runErr.Error())
		}
	}()
	return nil
}

// finish transitions the controller back to idle and emits cf:state(false).
func (c *Controller) finish() {
	c.mu.Lock()
	c.running = false
	c.cancel = nil
	ch := c.done
	c.done = nil
	c.mu.Unlock()

	c.em.Emit(EventState, false)
	if ch != nil {
		close(ch)
	}
}

// Cancel cancels the in-flight run's context, if any. Safe to call when idle.
func (c *Controller) Cancel() {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// IsRunning reports whether a run is currently in flight.
func (c *Controller) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// Done returns a channel that is closed when the current run completes, or nil
// when idle. Used by tests to synchronize; not intended for frontend use.
func (c *Controller) Done() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.done
}
