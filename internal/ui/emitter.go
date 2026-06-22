package ui

// Emitter delivers named events with optional data payloads to the frontend.
// The production implementation (in cmd/coverfixer-gui) wraps Wails'
// runtime.EventsEmit; tests inject a capturing fake. The Controller is the
// only emitter of cf:* events, which keeps the streaming path centralized and
// the engine-wiring free of GUI concerns.
type Emitter interface {
	Emit(name string, data ...any)
}

// Event names emitted by the Controller. The frontend subscribes to these.
const (
	// EventProgress carries a single formatted log line (its sole data arg is a
	// string) for each engine Event, in emission order.
	EventProgress = "cf:progress"
	// EventDone carries a formatted multi-line summary (string) and signals the
	// end of a successful (or cancelled) run.
	EventDone = "cf:done"
	// EventError carries an engine-level error message (string) and signals the
	// end of a failed run.
	EventError = "cf:error"
	// EventState carries a bool reporting whether a run is now in flight, so
	// the frontend can toggle control enablement on start and finish.
	EventState = "cf:state"
)
