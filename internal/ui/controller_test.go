package ui

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/colinstark/coverfixer/internal/core"
)

// ---------------------------------------------------------------------------
// fakes: capturing emitter + injectable runner
// ---------------------------------------------------------------------------

type emittedEvent struct {
	Name string
	Data []any
}

type fakeEmitter struct {
	mu     sync.Mutex
	events []emittedEvent
}

func (f *fakeEmitter) Emit(name string, data ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, emittedEvent{Name: name, Data: data})
}

func (f *fakeEmitter) snapshot() []emittedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]emittedEvent, len(f.events))
	copy(out, f.events)
	return out
}

// firstDataString returns the first data arg of the last emitted event with the
// given name as a string, or "" if none.
func (f *fakeEmitter) firstDataString(name string) string {
	for _, e := range f.snapshot() {
		if e.Name == name && len(e.Data) > 0 {
			if s, ok := e.Data[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

// dataStrings returns all string data args for events named name, in order.
func (f *fakeEmitter) dataStrings(name string) []string {
	var out []string
	for _, e := range f.snapshot() {
		if e.Name != name {
			continue
		}
		for _, d := range e.Data {
			if s, ok := d.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// fakeRunner simulates core.Run. If hold is non-nil it blocks on it (for
// in-flight tests); otherwise it returns immediately after emitting events.
type fakeRunner struct {
	mu           sync.Mutex
	capturedCtx  context.Context
	capturedOpts core.Options
	callCount    int32

	started chan struct{} // closed once after capturing + emitting events
	hold    chan struct{} // if non-nil, blocks until closed
	events  []core.Event
	report  core.Report
	err     error
}

func newBlockingFakeRunner() *fakeRunner {
	return &fakeRunner{started: make(chan struct{}), hold: make(chan struct{})}
}

func newImmediateFakeRunner(events []core.Event, report core.Report, err error) *fakeRunner {
	return &fakeRunner{started: make(chan struct{}), events: events, report: report, err: err}
}

func (f *fakeRunner) run(ctx context.Context, opts core.Options, progress func(core.Event)) (core.Report, error) {
	f.mu.Lock()
	f.capturedCtx = ctx
	f.capturedOpts = opts
	f.mu.Unlock()
	atomic.AddInt32(&f.callCount, 1)

	for _, e := range f.events {
		progress(e)
	}
	close(f.started)
	if f.hold != nil {
		<-f.hold
	}
	return f.report, f.err
}

func (f *fakeRunner) captured() (context.Context, core.Options) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.capturedCtx, f.capturedOpts
}

func (f *fakeRunner) calls() int32 { return atomic.LoadInt32(&f.callCount) }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func waitForDone(t *testing.T, c *Controller, timeout time.Duration) {
	t.Helper()
	ch := c.Done()
	if ch == nil {
		return
	}
	select {
	case <-ch:
		return
	case <-time.After(timeout):
		t.Fatal("run did not complete within timeout")
	}
}

func waitForStart(t *testing.T, fr *fakeRunner, timeout time.Duration) {
	t.Helper()
	select {
	case <-fr.started:
		return
	case <-time.After(timeout):
		t.Fatal("fake runner was not called within timeout")
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-001: Start invokes the engine exactly once
// ---------------------------------------------------------------------------

func TestStartInvokesRunnerOnce(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	c := newControllerWithRun(&fakeEmitter{}, fr.run)

	if err := c.Start(DefaultRequest()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForDone(t, c, 3*time.Second)

	if fr.calls() != 1 {
		t.Errorf("runner call count = %d, want 1", fr.calls())
	}
}

// ---------------------------------------------------------------------------
// Start forwards the RunRequest as a validated core.Options (covers the
// original VAL-EXEC-002/004/005/006 mapping invariants end-to-end)
// ---------------------------------------------------------------------------

func TestStartForwardsRequestOptions(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	c := newControllerWithRun(&fakeEmitter{}, fr.run)

	req := DefaultRequest()
	req.Dir = "/tmp/specific"
	req.ResizeEmbedded = true
	req.Backup = true
	req.ArtSize = 256
	req.JPEGQuality = 90
	req.Transcode = "aac-256"
	req.DryRun = false

	if err := c.Start(req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForDone(t, c, 3*time.Second)

	_, opts := fr.captured()
	if opts.Dir != "/tmp/specific" {
		t.Errorf("Dir = %q", opts.Dir)
	}
	if !opts.ResizeEmbedded {
		t.Error("ResizeEmbedded should be true")
	}
	if !opts.Backup {
		t.Error("Backup should be true")
	}
	if opts.ArtSize != 256 {
		t.Errorf("ArtSize = %d", opts.ArtSize)
	}
	if opts.JPEGQuality != 90 {
		t.Errorf("JPEGQuality = %d", opts.JPEGQuality)
	}
	if opts.Transcode != core.TranscodeAAC_256 {
		t.Errorf("Transcode = %v", opts.Transcode)
	}
	if opts.DryRun {
		t.Error("DryRun should be false")
	}
}

// ---------------------------------------------------------------------------
// An invalid request returns an error synchronously and never starts
// ---------------------------------------------------------------------------

func TestStartInvalidRequestIsRejected(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	c := newControllerWithRun(&fakeEmitter{}, fr.run)

	err := c.Start(RunRequest{Transcode: "nope"})
	if err == nil {
		t.Fatal("expected error for invalid transcode")
	}
	if fr.calls() != 0 {
		t.Errorf("runner should not be called, got %d", fr.calls())
	}
	if c.IsRunning() {
		t.Error("should not be running after rejected start")
	}
}

// ---------------------------------------------------------------------------
// Double-Start is rejected while a run is in flight
// ---------------------------------------------------------------------------

func TestDoubleStartGuard(t *testing.T) {
	fr := newBlockingFakeRunner()
	c := newControllerWithRun(&fakeEmitter{}, fr.run)

	if err := c.Start(DefaultRequest()); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	waitForStart(t, fr, 3*time.Second)

	if err := c.Start(DefaultRequest()); err == nil {
		t.Error("second Start should error while in flight")
	}
	if fr.calls() != 1 {
		t.Errorf("runner should be called once, got %d", fr.calls())
	}

	close(fr.hold)
	waitForDone(t, c, 3*time.Second)
}

// ---------------------------------------------------------------------------
// Cancel cancels the in-flight context
// ---------------------------------------------------------------------------

func TestCancelCancelsContext(t *testing.T) {
	fr := newBlockingFakeRunner()
	c := newControllerWithRun(&fakeEmitter{}, fr.run)

	if err := c.Start(DefaultRequest()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForStart(t, fr, 3*time.Second)

	c.Cancel()

	ctx, _ := fr.captured()
	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
		}
	default:
		t.Error("context should be cancelled after Cancel()")
	}

	close(fr.hold)
	waitForDone(t, c, 3*time.Second)
}

// ---------------------------------------------------------------------------
// Cancel is safe to call when idle (no-op, no panic)
// ---------------------------------------------------------------------------

func TestCancelWhenIdleIsNoop(t *testing.T) {
	c := newControllerWithRun(&fakeEmitter{}, newImmediateFakeRunner(nil, core.Report{}, nil).run)
	c.Cancel() // must not panic
}

// ---------------------------------------------------------------------------
// cf:state brackets the run (true on start, false on finish) and idle after
// ---------------------------------------------------------------------------

func TestStateEventsBracketRun(t *testing.T) {
	em := &fakeEmitter{}
	fr := newBlockingFakeRunner()
	c := newControllerWithRun(em, fr.run)

	if err := c.Start(DefaultRequest()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForStart(t, fr, 3*time.Second)

	if !c.IsRunning() {
		t.Error("should be running during run")
	}

	close(fr.hold)
	waitForDone(t, c, 3*time.Second)

	if c.IsRunning() {
		t.Error("should be idle after run")
	}

	states := em.dataStrings(EventState)
	// We expect at least one true-ish state then a false; track via raw events.
	evs := em.snapshot()
	var got []bool
	for _, e := range evs {
		if e.Name != EventState {
			continue
		}
		if len(e.Data) == 1 {
			if b, ok := e.Data[0].(bool); ok {
				got = append(got, b)
			}
		}
	}
	_ = states
	if len(got) < 2 || got[0] != true || got[len(got)-1] != false {
		t.Errorf("state events should go true…false, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Progress events stream in order as formatted lines via cf:progress
// (covers original VAL-EXEC-008/009/010/011)
// ---------------------------------------------------------------------------

func TestProgressEventsStreamInOrder(t *testing.T) {
	events := []core.Event{
		{Kind: core.EventAction, Op: "rename", Path: "a.jpg"},
		{Kind: core.EventSkip, Op: "resize-cover", Path: "b.jpg", Detail: "ok"},
		{Kind: core.EventError, Op: "extract", Path: "c.mp3", Err: errors.New("boom")},
	}
	em := &fakeEmitter{}
	fr := newImmediateFakeRunner(events, core.Report{}, nil)
	c := newControllerWithRun(em, fr.run)

	if err := c.Start(DefaultRequest()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForDone(t, c, 3*time.Second)

	lines := em.dataStrings(EventProgress)
	if len(lines) < 3 {
		t.Fatalf("expected >=3 progress lines, got %d: %v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "[ACT]") {
		t.Errorf("first line should be action, got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "[SKIP]") {
		t.Errorf("second line should be skip, got %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "[ERR]") || !strings.Contains(lines[2], "boom") {
		t.Errorf("third line should be error with text, got %q", lines[2])
	}
}

func TestProgressActionFormatting(t *testing.T) {
	events := []core.Event{
		{Kind: core.EventAction, Op: "resize-cover", Path: "/tmp/album/cover.jpg"},
		{Kind: core.EventAction, Op: "rename", Path: "/tmp/album/front.jpg", Detail: "→ cover.jpg"},
	}
	em := &fakeEmitter{}
	fr := newImmediateFakeRunner(events, core.Report{}, nil)
	c := newControllerWithRun(em, fr.run)

	c.Start(DefaultRequest())
	waitForDone(t, c, 3*time.Second)

	lines := em.dataStrings(EventProgress)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "[ACT] resize-cover: /tmp/album/cover.jpg") {
		t.Errorf("missing action line, got: %q", joined)
	}
	if !strings.Contains(joined, "[ACT] rename: /tmp/album/front.jpg (→ cover.jpg)") {
		t.Errorf("missing action line with detail, got: %q", joined)
	}
}

// ---------------------------------------------------------------------------
// Success emits cf:done with all seven counters (VAL-EXEC-012/013)
// ---------------------------------------------------------------------------

func TestDoneSummaryAllCounters(t *testing.T) {
	report := core.Report{
		Renamed: 2, CoversResized: 3, Extracted: 1, EmbeddedResized: 4,
		Transcoded: 5, Skipped: 6, Failed: 1,
	}
	em := &fakeEmitter{}
	fr := newImmediateFakeRunner(nil, report, nil)
	c := newControllerWithRun(em, fr.run)

	c.Start(DefaultRequest())
	waitForDone(t, c, 3*time.Second)

	summary := em.firstDataString(EventDone)
	for _, part := range []string{
		"Renamed: 2", "Covers Resized: 3", "Extracted: 1", "Embedded Resized: 4",
		"Transcoded: 5", "Skipped: 6", "Failed: 1",
	} {
		if !strings.Contains(summary, part) {
			t.Errorf("summary missing %q, got: %q", part, summary)
		}
	}
}

func TestDoneSummaryZero(t *testing.T) {
	em := &fakeEmitter{}
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	c := newControllerWithRun(em, fr.run)

	c.Start(DefaultRequest())
	waitForDone(t, c, 3*time.Second)

	summary := em.firstDataString(EventDone)
	for _, part := range []string{
		"Renamed: 0", "Covers Resized: 0", "Extracted: 0", "Embedded Resized: 0",
		"Transcoded: 0", "Skipped: 0", "Failed: 0",
	} {
		if !strings.Contains(summary, part) {
			t.Errorf("zero summary missing %q, got: %q", part, summary)
		}
	}
}

// Dry-run summary carries the no-files-modified banner.

func TestDoneDryRunBanner(t *testing.T) {
	em := &fakeEmitter{}
	fr := newImmediateFakeRunner(nil, core.Report{CoversResized: 1}, nil)
	c := newControllerWithRun(em, fr.run)

	req := DefaultRequest() // DryRun == true
	c.Start(req)
	waitForDone(t, c, 3*time.Second)

	summary := strings.ToLower(em.firstDataString(EventDone))
	if !strings.Contains(summary, "dry-run") {
		t.Errorf("dry-run done should mention dry-run, got: %q", summary)
	}
	if !strings.Contains(em.firstDataString(EventDone), "Covers Resized: 1") {
		t.Errorf("dry-run done should still show counters, got: %q", em.firstDataString(EventDone))
	}
}

// ---------------------------------------------------------------------------
// Cancelled run surfaces partial results via cf:done, never cf:error
// ---------------------------------------------------------------------------

func TestCancelledSurfacesPartialResults(t *testing.T) {
	em := &fakeEmitter{}
	fr := newImmediateFakeRunner(nil, core.Report{CoversResized: 4, Skipped: 2}, context.Canceled)
	c := newControllerWithRun(em, fr.run)

	c.Start(DefaultRequest())
	waitForDone(t, c, 3*time.Second)

	summary := em.firstDataString(EventDone)
	if !strings.Contains(strings.ToLower(summary), "cancel") {
		t.Errorf("cancelled done should mention cancel, got: %q", summary)
	}
	if !strings.Contains(summary, "Covers Resized: 4") {
		t.Errorf("cancelled done should show partial counters, got: %q", summary)
	}
	if em.firstDataString(EventError) != "" {
		t.Errorf("cancellation should not emit cf:error, got: %q", em.firstDataString(EventError))
	}
}

// ---------------------------------------------------------------------------
// Engine error surfaced via cf:error, and controller returns to idle
// (VAL-EXEC-019/020)
// ---------------------------------------------------------------------------

func TestEngineErrorSurfacedAndIdle(t *testing.T) {
	em := &fakeEmitter{}
	sentinel := "something went badly wrong"
	fr := newImmediateFakeRunner(nil, core.Report{}, errors.New(sentinel))
	c := newControllerWithRun(em, fr.run)

	c.Start(DefaultRequest())
	waitForDone(t, c, 3*time.Second)

	if got := em.firstDataString(EventError); !strings.Contains(got, sentinel) {
		t.Errorf("cf:error = %q, want it to contain %q", got, sentinel)
	}
	if em.firstDataString(EventDone) != "" {
		t.Errorf("failed run should not emit cf:done, got: %q", em.firstDataString(EventDone))
	}
	if c.IsRunning() {
		t.Error("should be idle after error")
	}
}

// ---------------------------------------------------------------------------
// A second run works after the first completes (VAL-EXEC-021 reset semantics)
// ---------------------------------------------------------------------------

func TestSecondRunAfterFirst(t *testing.T) {
	em := &fakeEmitter{}
	r1 := newImmediateFakeRunner(nil, core.Report{CoversResized: 5}, nil)
	c := newControllerWithRun(em, r1.run)

	c.Start(DefaultRequest())
	waitForDone(t, c, 3*time.Second)
	if got := em.firstDataString(EventDone); !strings.Contains(got, "Covers Resized: 5") {
		t.Errorf("first done = %q", got)
	}

	// Swap to a second runner and run again.
	em = &fakeEmitter{}
	c.em = em
	r2 := newImmediateFakeRunner(nil, core.Report{Renamed: 1}, nil)
	c.run = r2.run

	c.Start(DefaultRequest())
	waitForDone(t, c, 3*time.Second)
	if got := em.firstDataString(EventDone); !strings.Contains(got, "Renamed: 1") {
		t.Errorf("second done = %q", got)
	}
	if strings.Contains(em.firstDataString(EventDone), "Covers Resized: 5") {
		t.Errorf("second done should not carry stale first counters: %q", em.firstDataString(EventDone))
	}
}

// ---------------------------------------------------------------------------
// Real end-to-end run against a temp dir (VAL-EXEC-023/024)
// ---------------------------------------------------------------------------

func TestRealEndToEndResize(t *testing.T) {
	dir := t.TempDir()

	// Synthesize an oversized 1000x1000 cover.jpg.
	img := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
	coverPath := filepath.Join(dir, "cover.jpg")
	f, err := os.Create(coverPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 95}); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// Controller wired to the real core.Run.
	em := &fakeEmitter{}
	c := NewController(em)

	req := DefaultRequest()
	req.Dir = dir
	req.DryRun = false // mutate for real

	if err := c.Start(req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForDone(t, c, 10*time.Second)

	// cover.jpg should now decode to <= 500x500.
	data, err := os.ReadFile(coverPath)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode cover.jpg after run: %v", err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() > 500 || bounds.Dy() > 500 {
		t.Errorf("cover.jpg %dx%d, expected <= 500x500", bounds.Dx(), bounds.Dy())
	}

	// cf:done summary should report a non-zero resize.
	summary := em.firstDataString(EventDone)
	if strings.Contains(summary, "Covers Resized: 0") {
		t.Errorf("expected Covers Resized >= 1, got: %q", summary)
	}
}
