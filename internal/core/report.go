package core

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
	Op     string // "rename", "resize-cover", "extract", "resize-embedded", "transcode", "backup"
	Path   string
	Detail string
	Err    error
}

// Report tallies the work performed across a Run, mirroring the counters from
// the original rockbox_covers.sh script.
type Report struct {
	Renamed         int
	CoversResized   int
	Extracted       int
	EmbeddedResized int
	Transcoded      int
	Skipped         int
	Failed          int
}

func (r *Report) info(progress func(Event), op, path, detail string) {
	progress(Event{Kind: EventInfo, Op: op, Path: path, Detail: detail})
}

func (r *Report) action(progress func(Event), op, path, detail string) {
	progress(Event{Kind: EventAction, Op: op, Path: path, Detail: detail})
}

func (r *Report) skip(progress func(Event), op, path, detail string) {
	r.Skipped++
	progress(Event{Kind: EventSkip, Op: op, Path: path, Detail: detail})
}

func (r *Report) fail(progress func(Event), op, path string, err error) {
	r.Failed++
	progress(Event{Kind: EventError, Op: op, Path: path, Err: err})
}
