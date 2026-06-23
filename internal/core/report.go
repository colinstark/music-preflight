package core

import "sync"

// EventKind classifies a progress Event for front-end formatting.
type EventKind int

const (
	EventInfo   EventKind = iota // informational (scanning, folder headers)
	EventAction                  // a mutation happened (or would, under DryRun)
	EventSkip                    // nothing to do for this item
	EventError                   // a recoverable per-file failure
)

// Event is emitted through Run's progress callback for each unit of work.
type Event struct {
	Kind   EventKind
	Op     string // "rename", "resize-cover", "extract", "resize-embedded", "transcode", "set-genre", "backup"
	Path   string
	Detail string
	Err    error
}

// Report tallies the work performed across a Run, mirroring the counters from
// the original rockbox_covers.sh script. It is a plain, lock-free snapshot
// returned by Run; the guarded accumulator used while a run is in flight is
// reportAccum, internal to this package.
type Report struct {
	Renamed         int
	CoversResized   int
	Extracted       int
	EmbeddedResized int
	Transcoded      int
	GenresSet       int
	Skipped         int
	Failed          int
}

// reportAccum is the concurrency-safe accumulator used while a Run is in
// flight. It embeds Report so Run can return the frozen counters by value
// (without copying a lock); the mutex guards counter increments from the
// parallel artwork/genre/transcode passes. The progress callback is always
// invoked outside the lock so emitting an event can't deadlock against a worker
// waiting to record a counter.
type reportAccum struct {
	Report
	mu sync.Mutex
}

// inc atomically increments a counter field. Call sites pass &a.<Field>.
func (a *reportAccum) inc(counter *int) {
	a.mu.Lock()
	*counter++
	a.mu.Unlock()
}

func (a *reportAccum) info(progress func(Event), op, path, detail string) {
	progress(Event{Kind: EventInfo, Op: op, Path: path, Detail: detail})
}

func (a *reportAccum) action(progress func(Event), op, path, detail string) {
	progress(Event{Kind: EventAction, Op: op, Path: path, Detail: detail})
}

func (a *reportAccum) skip(progress func(Event), op, path, detail string) {
	a.inc(&a.Skipped)
	progress(Event{Kind: EventSkip, Op: op, Path: path, Detail: detail})
}

func (a *reportAccum) fail(progress func(Event), op, path string, err error) {
	a.inc(&a.Failed)
	progress(Event{Kind: EventError, Op: op, Path: path, Err: err})
}
