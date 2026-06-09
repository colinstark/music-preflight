package gui

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

	"fyne.io/fyne/v2/test"
	"github.com/colinstark/coverfixer/internal/core"
)

// ---------------------------------------------------------------------------
// fakeRunner — injectable runner seam for headless tests
// ---------------------------------------------------------------------------

// fakeRunner simulates the core.Run engine for tests. It captures the
// context and options, emits configured events, then either blocks on a
// hold channel (for in-flight testing) or returns immediately.
type fakeRunner struct {
	mu           sync.Mutex
	capturedCtx  context.Context
	capturedOpts core.Options
	callCount    int32 // accessed atomically

	started chan struct{} // closed once after capturing values + emitting events
	hold    chan struct{} // if non-nil, runner blocks here; close to release
	events  []core.Event
	report  core.Report
	err     error
}

func newBlockingFakeRunner() *fakeRunner {
	return &fakeRunner{
		started: make(chan struct{}),
		hold:    make(chan struct{}),
	}
}

func newImmediateFakeRunner(events []core.Event, report core.Report, err error) *fakeRunner {
	return &fakeRunner{
		started: make(chan struct{}),
		events:  events,
		report:  report,
		err:     err,
		// hold is nil → returns immediately
	}
}

func (f *fakeRunner) run(ctx context.Context, opts core.Options, progress func(core.Event)) (core.Report, error) {
	f.mu.Lock()
	f.capturedCtx = ctx
	f.capturedOpts = opts
	f.mu.Unlock()
	atomic.AddInt32(&f.callCount, 1)

	// Emit configured events
	for _, e := range f.events {
		progress(e)
	}

	close(f.started)

	// Block if hold channel exists (for in-flight testing)
	if f.hold != nil {
		<-f.hold
	}

	return f.report, f.err
}

func (f *fakeRunner) getCaptured() (context.Context, core.Options) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.capturedCtx, f.capturedOpts
}

func (f *fakeRunner) getCallCount() int32 {
	return atomic.LoadInt32(&f.callCount)
}

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

func newTestUIWithRunner(run runFunc) *UI {
	app := test.NewApp()
	return newWithRunner(app, run)
}

func waitForRunDone(t *testing.T, ui *UI, timeout time.Duration) {
	t.Helper()
	select {
	case <-ui.runDone:
		return
	case <-time.After(timeout):
		t.Fatal("run did not complete within timeout")
	}
}

func waitForRunnerStart(t *testing.T, fr *fakeRunner, timeout time.Duration) {
	t.Helper()
	select {
	case <-fr.started:
		return
	case <-time.After(timeout):
		t.Fatal("fake runner was not called within timeout")
	}
}

func prepareUIWithFolder(t *testing.T, run runFunc) *UI {
	t.Helper()
	ui := newTestUIWithRunner(run)
	ui.setSelectedFolder("/tmp/test-music")
	return ui
}

// ---------------------------------------------------------------------------
// VAL-EXEC-001: Run invokes the engine runner exactly once
// ---------------------------------------------------------------------------

func TestRunInvokesRunnerOnce(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	if fr.getCallCount() != 1 {
		t.Errorf("runner call count = %d, want 1", fr.getCallCount())
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-002: Run passes the selected Dir to the engine
// ---------------------------------------------------------------------------

func TestRunPassesSelectedDir(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	dir := "/tmp/specific-music-dir"
	ui := newTestUIWithRunner(fr.run)
	ui.setSelectedFolder(dir)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	_, opts := fr.getCaptured()
	if opts.Dir != dir {
		t.Errorf("captured Dir = %q, want %q", opts.Dir, dir)
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-003: Default run maps DefaultOptions parity (dry-run ON)
// VAL-EXEC-004: GUI defaults to dry-run ON (safety)
// ---------------------------------------------------------------------------

func TestDefaultRunOptionsParity(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	_, opts := fr.getCaptured()
	defs := core.DefaultOptions()

	if opts.Dir != "/tmp/test-music" {
		t.Errorf("Dir = %q, want /tmp/test-music", opts.Dir)
	}
	if !opts.DryRun {
		t.Error("DryRun should be true by default (safety)")
	}
	if opts.Recursive != defs.Recursive {
		t.Errorf("Recursive = %v, want %v", opts.Recursive, defs.Recursive)
	}
	if opts.RenameStrayJPG != defs.RenameStrayJPG {
		t.Errorf("RenameStrayJPG = %v, want %v", opts.RenameStrayJPG, defs.RenameStrayJPG)
	}
	if opts.ResizeCoverJPG != defs.ResizeCoverJPG {
		t.Errorf("ResizeCoverJPG = %v, want %v", opts.ResizeCoverJPG, defs.ResizeCoverJPG)
	}
	if opts.ExtractCover != defs.ExtractCover {
		t.Errorf("ExtractCover = %v, want %v", opts.ExtractCover, defs.ExtractCover)
	}
	if opts.ResizeEmbedded != defs.ResizeEmbedded {
		t.Errorf("ResizeEmbedded = %v, want %v", opts.ResizeEmbedded, defs.ResizeEmbedded)
	}
	if opts.ArtSize != defs.ArtSize {
		t.Errorf("ArtSize = %d, want %d", opts.ArtSize, defs.ArtSize)
	}
	if opts.JPEGQuality != defs.JPEGQuality {
		t.Errorf("JPEGQuality = %d, want %d", opts.JPEGQuality, defs.JPEGQuality)
	}
	if opts.Transcode != defs.Transcode {
		t.Errorf("Transcode = %v, want %v", opts.Transcode, defs.Transcode)
	}
	if opts.Backup {
		t.Error("Backup should be false by default")
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-005: Toggling controls remaps options before Run
// ---------------------------------------------------------------------------

func TestTogglingControlsRemapsOptions(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	ui.dryRunCheck.SetChecked(false)
	ui.resizeEmbeddedCheck.SetChecked(true)
	ui.backupCheck.SetChecked(true)
	ui.artSizeEntry.SetText("256")
	ui.qualityEntry.SetText("90")
	ui.transcodeSelect.SetSelected("aac-256")

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	_, opts := fr.getCaptured()

	if opts.DryRun {
		t.Error("DryRun should be false after unchecking")
	}
	if !opts.ResizeEmbedded {
		t.Error("ResizeEmbedded should be true after checking")
	}
	if !opts.Backup {
		t.Error("Backup should be true after checking")
	}
	if opts.ArtSize != 256 {
		t.Errorf("ArtSize = %d, want 256", opts.ArtSize)
	}
	if opts.JPEGQuality != 90 {
		t.Errorf("JPEGQuality = %d, want 90", opts.JPEGQuality)
	}
	if opts.Transcode != core.TranscodeAAC_256 {
		t.Errorf("Transcode = %v, want TranscodeAAC_256", opts.Transcode)
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-006: Invalid Art size / quality fall back to engine defaults
// ---------------------------------------------------------------------------

func TestInvalidNumericFallbacks(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	ui.artSizeEntry.SetText("abc")
	ui.qualityEntry.SetText("xx")

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	_, opts := fr.getCaptured()

	if opts.ArtSize != 500 {
		t.Errorf("ArtSize = %d, want 500 (fallback for invalid)", opts.ArtSize)
	}
	if opts.JPEGQuality != 85 {
		t.Errorf("JPEGQuality = %d, want 85 (fallback for invalid)", opts.JPEGQuality)
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-007: Transcode select string maps to the correct mode
// ---------------------------------------------------------------------------

func TestTranscodeSelectMapping(t *testing.T) {
	tests := []struct {
		selected string
		want     core.TranscodeMode
	}{
		{"none", core.TranscodeNone},
		{"mp3-320", core.TranscodeMP3_320},
		{"aac-256", core.TranscodeAAC_256},
	}
	for _, tc := range tests {
		t.Run(tc.selected, func(t *testing.T) {
			fr := newImmediateFakeRunner(nil, core.Report{}, nil)
			ui := prepareUIWithFolder(t, fr.run)
			ui.transcodeSelect.SetSelected(tc.selected)

			test.Tap(ui.runBtn)
			waitForRunDone(t, ui, 3*time.Second)

			_, opts := fr.getCaptured()
			if opts.Transcode != tc.want {
				t.Errorf("Transcode = %v, want %v", opts.Transcode, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-008: Action events render as readable action log lines
// ---------------------------------------------------------------------------

func TestActionEventFormatting(t *testing.T) {
	events := []core.Event{
		{Kind: core.EventAction, Op: "resize-cover", Path: "/tmp/album/cover.jpg"},
		{Kind: core.EventAction, Op: "rename", Path: "/tmp/album/front.jpg", Detail: "→ cover.jpg"},
	}
	fr := newImmediateFakeRunner(events, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	logText := ui.progressLog.Text
	if !strings.Contains(logText, "[ACT] resize-cover: /tmp/album/cover.jpg") {
		t.Errorf("progress log missing action line for resize-cover, got: %q", logText)
	}
	if !strings.Contains(logText, "[ACT] rename: /tmp/album/front.jpg (→ cover.jpg)") {
		t.Errorf("progress log missing action line with detail, got: %q", logText)
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-009: Skip events render with distinct skip formatting
// ---------------------------------------------------------------------------

func TestSkipEventFormatting(t *testing.T) {
	events := []core.Event{
		{Kind: core.EventSkip, Op: "resize-cover", Path: "/tmp/album/cover.jpg", Detail: "already within size, baseline"},
	}
	fr := newImmediateFakeRunner(events, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	logText := ui.progressLog.Text
	if !strings.Contains(logText, "[SKIP]") {
		t.Errorf("progress log missing [SKIP] prefix, got: %q", logText)
	}
	if !strings.Contains(logText, "already within size, baseline") {
		t.Errorf("progress log missing skip detail, got: %q", logText)
	}
	// Verify skip formatting is distinct from action
	if strings.Contains(logText, "[ACT]") {
		t.Error("skip event should not contain [ACT] prefix")
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-010: Error events render with distinct error formatting and detail
// ---------------------------------------------------------------------------

func TestErrorEventFormatting(t *testing.T) {
	events := []core.Event{
		{Kind: core.EventError, Op: "resize-cover", Path: "/tmp/album/bad.jpg", Err: os.ErrPermission},
	}
	fr := newImmediateFakeRunner(events, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	logText := ui.progressLog.Text
	if !strings.Contains(logText, "[ERR]") {
		t.Errorf("progress log missing [ERR] prefix, got: %q", logText)
	}
	if !strings.Contains(logText, "permission denied") {
		t.Errorf("progress log missing error text, got: %q", logText)
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-011: Multiple progress events stream in emission order
// ---------------------------------------------------------------------------

func TestEventsEmissionOrder(t *testing.T) {
	events := []core.Event{
		{Kind: core.EventAction, Op: "rename", Path: "a.jpg"},
		{Kind: core.EventSkip, Op: "resize-cover", Path: "b.jpg", Detail: "ok"},
		{Kind: core.EventError, Op: "extract", Path: "c.mp3", Err: os.ErrNotExist},
	}
	fr := newImmediateFakeRunner(events, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	logText := ui.progressLog.Text
	lines := strings.Split(logText, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 log lines, got %d: %q", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "[ACT]") {
		t.Errorf("first line should be action, got: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "[SKIP]") {
		t.Errorf("second line should be skip, got: %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "[ERR]") {
		t.Errorf("third line should be error, got: %q", lines[2])
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-012: Summary shows all seven Report counters on completion
// ---------------------------------------------------------------------------

func TestSummaryShowsAllCounters(t *testing.T) {
	report := core.Report{
		Renamed:         2,
		CoversResized:   3,
		Extracted:       1,
		EmbeddedResized: 4,
		Transcoded:      5,
		Skipped:         6,
		Failed:          1,
	}
	fr := newImmediateFakeRunner(nil, report, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	summary := ui.summaryLabel.Text
	expectedParts := []string{
		"Renamed: 2",
		"Covers Resized: 3",
		"Extracted: 1",
		"Embedded Resized: 4",
		"Transcoded: 5",
		"Skipped: 6",
		"Failed: 1",
	}
	for _, part := range expectedParts {
		if !strings.Contains(summary, part) {
			t.Errorf("summary missing %q, got: %q", part, summary)
		}
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-013: Zero report renders a valid all-zero summary
// ---------------------------------------------------------------------------

func TestZeroReportSummary(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	summary := ui.summaryLabel.Text
	expectedParts := []string{
		"Renamed: 0",
		"Covers Resized: 0",
		"Extracted: 0",
		"Embedded Resized: 0",
		"Transcoded: 0",
		"Skipped: 0",
		"Failed: 0",
	}
	for _, part := range expectedParts {
		if !strings.Contains(summary, part) {
			t.Errorf("zero summary missing %q, got: %q", part, summary)
		}
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-014: Controls are disabled while a run is in flight
// ---------------------------------------------------------------------------

func TestControlsDisabledDuringRun(t *testing.T) {
	fr := newBlockingFakeRunner()
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunnerStart(t, fr, 3*time.Second)

	// Controls should be disabled while runner is blocked
	if !ui.runBtn.Disabled() {
		t.Error("Run button should be disabled during run")
	}
	if !ui.folderBtn.Disabled() {
		t.Error("Folder button should be disabled during run")
	}
	if !ui.dryRunCheck.Disabled() {
		t.Error("Dry-run check should be disabled during run")
	}
	if !ui.artSizeEntry.Disabled() {
		t.Error("Art-size entry should be disabled during run")
	}
	if !ui.transcodeSelect.Disabled() {
		t.Error("Transcode select should be disabled during run")
	}

	// Cancel should be enabled during run
	if ui.cancelBtn.Disabled() {
		t.Error("Cancel button should be enabled during run")
	}

	// Release the runner
	close(fr.hold)
	waitForRunDone(t, ui, 3*time.Second)
}

// ---------------------------------------------------------------------------
// VAL-EXEC-015: Controls are re-enabled after the run completes
// ---------------------------------------------------------------------------

func TestControlsReenabledAfterRun(t *testing.T) {
	fr := newImmediateFakeRunner(nil, core.Report{}, nil)
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	if ui.runBtn.Disabled() {
		t.Error("Run button should be re-enabled after run completes")
	}
	if ui.folderBtn.Disabled() {
		t.Error("Folder button should be re-enabled after run completes")
	}
	if ui.dryRunCheck.Disabled() {
		t.Error("Dry-run check should be re-enabled after run completes")
	}
	if !ui.cancelBtn.Disabled() {
		t.Error("Cancel button should be disabled after run completes (nothing to cancel)")
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-016: Run cannot be double-invoked while a run is active
// ---------------------------------------------------------------------------

func TestDoubleInvocationGuard(t *testing.T) {
	fr := newBlockingFakeRunner()
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunnerStart(t, fr, 3*time.Second)

	// Try a second Run tap while the first is in flight
	test.Tap(ui.runBtn)

	if fr.getCallCount() != 1 {
		t.Errorf("runner should be called exactly once, got %d", fr.getCallCount())
	}

	// Clean up
	close(fr.hold)
	waitForRunDone(t, ui, 3*time.Second)
}

// ---------------------------------------------------------------------------
// VAL-EXEC-017: Cancel cancels the in-flight run via context
// ---------------------------------------------------------------------------

func TestCancelCancelsContext(t *testing.T) {
	fr := newBlockingFakeRunner()
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunnerStart(t, fr, 3*time.Second)

	// Tap Cancel
	test.Tap(ui.cancelBtn)

	// The captured context should be cancelled
	ctx, _ := fr.getCaptured()
	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
		}
	default:
		t.Error("context should be cancelled after tapping Cancel")
	}

	// Release the runner so the run can complete
	close(fr.hold)
	waitForRunDone(t, ui, 3*time.Second)
}

// ---------------------------------------------------------------------------
// VAL-EXEC-018: UI returns to idle after a cancelled run
// ---------------------------------------------------------------------------

func TestIdleAfterCancellation(t *testing.T) {
	fr := newBlockingFakeRunner()
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunnerStart(t, fr, 3*time.Second)

	test.Tap(ui.cancelBtn)
	close(fr.hold)
	waitForRunDone(t, ui, 3*time.Second)

	if ui.runBtn.Disabled() {
		t.Error("Run button should be re-enabled after cancellation")
	}
	if ui.running {
		t.Error("UI should not be running after cancellation")
	}
	if !ui.cancelBtn.Disabled() {
		t.Error("Cancel button should be disabled after cancellation (nothing to cancel)")
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-019: Engine-level error is surfaced to the user
// VAL-EXEC-020: A failing run still returns the UI to idle
// ---------------------------------------------------------------------------

func TestEngineErrorSurfacedAndIdle(t *testing.T) {
	sentinelErr := "something went badly wrong"
	fr := newImmediateFakeRunner(nil, core.Report{}, errors.New(sentinelErr))
	ui := prepareUIWithFolder(t, fr.run)

	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	// Error should be surfaced
	errText := ui.errorLabel.Text
	if !strings.Contains(errText, sentinelErr) {
		t.Errorf("error label = %q, want it to contain %q", errText, sentinelErr)
	}

	// UI should be back to idle
	if ui.runBtn.Disabled() {
		t.Error("Run button should be re-enabled after error")
	}
	if ui.running {
		t.Error("UI should not be running after error")
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-021: New run resets the prior summary
// ---------------------------------------------------------------------------

func TestNewRunResetsSummary(t *testing.T) {
	report1 := core.Report{CoversResized: 5, Skipped: 3}
	report2 := core.Report{Renamed: 1, Extracted: 2}

	var callIdx int32
	hold1 := make(chan struct{})
	ui := newTestUIWithRunner(func(ctx context.Context, opts core.Options, progress func(core.Event)) (core.Report, error) {
		idx := atomic.AddInt32(&callIdx, 1)
		if idx == 1 {
			<-hold1 // block first call
			return report1, nil
		}
		return report2, nil
	})
	ui.setSelectedFolder("/tmp/test-music")

	// First run
	test.Tap(ui.runBtn)
	// Wait for the first run to be in flight
	time.Sleep(50 * time.Millisecond) // let the goroutine start
	close(hold1)                      // release the first run
	waitForRunDone(t, ui, 3*time.Second)

	summary1 := ui.summaryLabel.Text
	if !strings.Contains(summary1, "Covers Resized: 5") {
		t.Errorf("after first run, summary should contain report1 data, got: %q", summary1)
	}

	// Second run
	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 3*time.Second)

	summary2 := ui.summaryLabel.Text
	if !strings.Contains(summary2, "Renamed: 1") {
		t.Errorf("after second run, summary should contain report2 data, got: %q", summary2)
	}
	if !strings.Contains(summary2, "Extracted: 2") {
		t.Errorf("after second run, summary should contain report2 Extracted, got: %q", summary2)
	}
	if strings.Contains(summary2, "Covers Resized: 5") {
		t.Errorf("after second run, summary should NOT contain stale report1 data, got: %q", summary2)
	}
}

// ---------------------------------------------------------------------------
// VAL-EXEC-022: Run button disabled until a folder is selected
// (Already covered by TestRunButtonDisabledInitially and
//  TestFolderSelectionEnablesRun in gui_test.go)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// VAL-EXEC-023: Real end-to-end run (dry-run OFF) resizes oversized cover.jpg
// VAL-EXEC-024: Real end-to-end run reports counters from core.Report
// ---------------------------------------------------------------------------

func TestRealEndToEndResize(t *testing.T) {
	// Create a temp directory with a synthetic oversized cover.jpg
	dir := t.TempDir()

	// Create a 1000×1000 JPEG
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

	// Create a UI with the real core.Run engine
	app := test.NewApp()
	ui := newWithRunner(app, core.Run)
	ui.setSelectedFolder(dir)
	ui.dryRunCheck.SetChecked(false) // DryRun OFF for real mutation

	// Run
	test.Tap(ui.runBtn)
	waitForRunDone(t, ui, 10*time.Second)

	// Assert cover.jpg was resized to ≤ 500×500
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
		t.Errorf("cover.jpg dimensions %dx%d, expected ≤ 500×500", bounds.Dx(), bounds.Dy())
	}

	// Assert summary reflects the real report (CoversResized ≥ 1)
	summary := ui.summaryLabel.Text
	if !strings.Contains(summary, "Covers Resized:") {
		t.Error("summary missing 'Covers Resized:' counter")
	}
	// The resized counter should be at least 1
	if strings.Contains(summary, "Covers Resized: 0") {
		t.Errorf("summary shows Covers Resized: 0, expected ≥ 1, got: %q", summary)
	}
}
