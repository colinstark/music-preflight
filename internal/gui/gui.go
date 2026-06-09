// Package gui provides the Fyne desktop front-end for coverfixer.
// It drives the core engine through core.Run and exposes full
// core.Options parity controls in a single window.
package gui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/colinstark/coverfixer/internal/core"
)

// runFunc is the injectable runner seam.
// Production uses core.Run; tests inject a fake.
type runFunc func(context.Context, core.Options, func(core.Event)) (core.Report, error)

// UI represents the coverfixer GUI window and its widget state.
type UI struct {
	app    fyne.App
	window fyne.Window
	run    runFunc

	// Widgets — exposed for headless testing (fyne/v2/test).
	folderBtn           *widget.Button
	pathLabel           *widget.Label
	dryRunCheck         *widget.Check
	recursiveCheck      *widget.Check
	renameStrayCheck    *widget.Check
	resizeCoverCheck    *widget.Check
	extractCheck        *widget.Check
	resizeEmbeddedCheck *widget.Check
	artSizeEntry        *widget.Entry
	qualityEntry        *widget.Entry
	transcodeSelect     *widget.Select
	backupCheck         *widget.Check
	runBtn              *widget.Button
	cancelBtn           *widget.Button
	progressLog         *widget.Entry
	summaryLabel        *widget.Label

	// State
	selectedDir string
	cancelFn    context.CancelFunc
	running     bool
	runDone     chan struct{} // closed when a run completes (for test sync)

	// Progress log batching. The worker goroutine appends formatted lines to
	// progressBuf (O(1) amortized); widget updates are coalesced so we issue
	// at most one pending fyne.Do flush at a time instead of one per event.
	progressMu      sync.Mutex
	progressBuf     strings.Builder
	progressPending bool

	// Error display
	errorLabel *widget.Label
}

// New creates a UI that uses the real core.Run engine.
func New(app fyne.App) *UI {
	return newWithRunner(app, core.Run)
}

// newWithRunner creates a UI with an injectable runner for tests.
func newWithRunner(app fyne.App, run runFunc) *UI {
	ui := &UI{
		app: app,
		run: run,
	}
	ui.buildUI()
	return ui
}

// ShowAndRun shows the window and starts the Fyne event loop.
func (ui *UI) ShowAndRun() {
	ui.window.ShowAndRun()
}

// setSelectedFolder is the test seam for folder selection.
// It simulates the dialog callback without opening a real dialog.
func (ui *UI) setSelectedFolder(dir string) {
	ui.selectedDir = dir
	ui.pathLabel.SetText(dir)
	if dir == "" {
		ui.runBtn.Disable()
	} else {
		ui.runBtn.Enable()
	}
}

// onRun is the Run button callback. It reads the current options, disables
// controls, clears prior output, and invokes the runner in a goroutine with
// a fresh cancelable context. Guard against double-invocation while a run is active.
func (ui *UI) onRun() {
	if ui.running || ui.selectedDir == "" {
		return
	}
	ui.running = true
	ui.runDone = make(chan struct{})

	// Clear prior output
	ui.progressMu.Lock()
	ui.progressBuf.Reset()
	ui.progressPending = false
	ui.progressMu.Unlock()
	ui.progressLog.SetText("")
	ui.summaryLabel.SetText("")
	ui.errorLabel.SetText("")

	// Disable input controls and Run; enable Cancel
	ui.setControlsEnabled(false)
	ui.cancelBtn.Enable()

	// Create cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	ui.cancelFn = cancel

	// Capture options before starting the goroutine
	opts := ui.options()

	go func() {
		report, err := ui.run(ctx, opts, ui.progressCallback)

		fyne.Do(func() {
			defer func() {
				ui.running = false
				ui.cancelFn = nil
				ui.setControlsEnabled(true)
				ui.cancelBtn.Disable()
				close(ui.runDone)
			}()

			// Final flush so the log reflects every event, regardless of how
			// the coalesced progress flushes happened to be scheduled.
			ui.flushProgressLog()

			switch {
			case err == nil:
				ui.summaryLabel.SetText(formatReport(report, opts.DryRun))
			case errors.Is(err, context.Canceled):
				// Cancelled mid-run: the engine still returns the work done so
				// far, so surface it rather than leaving the summary blank.
				ui.summaryLabel.SetText("Run cancelled — partial results:\n\n" + formatReport(report, opts.DryRun))
			default:
				ui.errorLabel.SetText(err.Error())
			}
		})
	}()
}

// onCancel is the Cancel button callback. It cancels the in-flight run's context.
func (ui *UI) onCancel() {
	if ui.cancelFn != nil {
		ui.cancelFn()
		ui.cancelFn = nil
	}
}

// progressCallback is the engine progress callback. It runs on the worker
// goroutine, so it appends to progressBuf and marshals a coalesced flush to
// the UI thread via fyne.Do — at most one flush is in flight at a time, so a
// burst of events costs one widget update rather than one per event.
func (ui *UI) progressCallback(e core.Event) {
	line := formatEvent(e)

	ui.progressMu.Lock()
	if ui.progressBuf.Len() > 0 {
		ui.progressBuf.WriteByte('\n')
	}
	ui.progressBuf.WriteString(line)
	schedule := !ui.progressPending
	ui.progressPending = true
	ui.progressMu.Unlock()

	if schedule {
		fyne.Do(ui.flushProgressLog)
	}
}

// flushProgressLog writes the accumulated buffer into the progress widget.
// It must run on the UI thread (called via fyne.Do or from another UI-thread
// callback). Clearing progressPending under the lock allows the next event to
// schedule a fresh flush.
func (ui *UI) flushProgressLog() {
	ui.progressMu.Lock()
	text := ui.progressBuf.String()
	ui.progressPending = false
	ui.progressMu.Unlock()
	ui.progressLog.SetText(text)
}

// setControlsEnabled enables or disables all input controls and the Run button.
// The Cancel button is managed separately by the run lifecycle.
func (ui *UI) setControlsEnabled(enabled bool) {
	if enabled {
		ui.folderBtn.Enable()
		ui.dryRunCheck.Enable()
		ui.recursiveCheck.Enable()
		ui.renameStrayCheck.Enable()
		ui.resizeCoverCheck.Enable()
		ui.extractCheck.Enable()
		ui.resizeEmbeddedCheck.Enable()
		ui.artSizeEntry.Enable()
		ui.qualityEntry.Enable()
		ui.transcodeSelect.Enable()
		ui.backupCheck.Enable()
		if ui.selectedDir != "" {
			ui.runBtn.Enable()
		}
	} else {
		ui.folderBtn.Disable()
		ui.dryRunCheck.Disable()
		ui.recursiveCheck.Disable()
		ui.renameStrayCheck.Disable()
		ui.resizeCoverCheck.Disable()
		ui.extractCheck.Disable()
		ui.resizeEmbeddedCheck.Disable()
		ui.artSizeEntry.Disable()
		ui.qualityEntry.Disable()
		ui.transcodeSelect.Disable()
		ui.backupCheck.Disable()
		ui.runBtn.Disable()
	}
}

// positiveIntValidator returns a fyne entry validator that accepts an empty
// string (the field falls back to the engine default) or an integer in
// [min, max]. A max of 0 means "no upper bound". The label personalizes the
// error message shown beneath the field.
func positiveIntValidator(label string, min, max int) fyne.StringValidator {
	return func(s string) error {
		if s == "" {
			return nil // empty → engine default, handled in options()
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("%s must be a whole number", label)
		}
		if v < min || (max > 0 && v > max) {
			if max > 0 {
				return fmt.Errorf("%s must be between %d and %d", label, min, max)
			}
			return fmt.Errorf("%s must be at least %d", label, min)
		}
		return nil
	}
}

// buildUI constructs all widgets and the window content with correct defaults.
func (ui *UI) buildUI() {
	ui.window = ui.app.NewWindow("Coverfixer")

	// --- Folder picker ---
	ui.pathLabel = widget.NewLabel("")
	ui.folderBtn = widget.NewButton("Choose Folder", func() {
		dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if dir != nil {
				ui.setSelectedFolder(dir.Path())
			}
		}, ui.window).Show()
	})

	// --- Checks with correct defaults ---
	ui.dryRunCheck = widget.NewCheck("Dry-run", nil)
	ui.dryRunCheck.SetChecked(true)

	ui.recursiveCheck = widget.NewCheck("Recursive", nil)
	ui.recursiveCheck.SetChecked(true)

	ui.renameStrayCheck = widget.NewCheck("Rename stray jpg", nil)
	ui.renameStrayCheck.SetChecked(true)

	ui.resizeCoverCheck = widget.NewCheck("Resize cover.jpg", nil)
	ui.resizeCoverCheck.SetChecked(true)

	ui.extractCheck = widget.NewCheck("Extract cover", nil)
	ui.extractCheck.SetChecked(true)

	ui.resizeEmbeddedCheck = widget.NewCheck("Resize embedded", nil)
	ui.resizeEmbeddedCheck.SetChecked(false)

	// --- Numeric entries (with validators for inline feedback) ---
	ui.artSizeEntry = widget.NewEntry()
	ui.artSizeEntry.Validator = positiveIntValidator("Art size", 1, 0)
	ui.artSizeEntry.SetText("500")

	ui.qualityEntry = widget.NewEntry()
	ui.qualityEntry.Validator = positiveIntValidator("JPEG quality", 1, 100)
	ui.qualityEntry.SetText("85")

	// --- Transcode select ---
	ui.transcodeSelect = widget.NewSelect(
		[]string{"none", "mp3-320", "aac-256"},
		nil,
	)
	ui.transcodeSelect.SetSelected("none")

	// --- Backup check ---
	ui.backupCheck = widget.NewCheck("Backup", nil)
	ui.backupCheck.SetChecked(false)

	// --- Run and Cancel buttons ---
	ui.runBtn = widget.NewButton("Run", ui.onRun)
	ui.runBtn.Disable() // disabled until a folder is selected

	ui.cancelBtn = widget.NewButton("Cancel", ui.onCancel)
	ui.cancelBtn.Disable() // nothing to cancel initially

	// --- Progress log (scrolling, read-only) ---
	ui.progressLog = widget.NewMultiLineEntry()
	ui.progressLog.SetPlaceHolder("Progress will appear here…")
	ui.progressLog.Disable() // read-only; programmatic SetText still works

	// --- Summary label (wraps so long content doesn't overflow) ---
	ui.summaryLabel = widget.NewLabel("")
	ui.summaryLabel.Wrapping = fyne.TextWrapWord

	// --- Error label (wraps so long engine errors stay readable) ---
	ui.errorLabel = widget.NewLabel("")
	ui.errorLabel.Wrapping = fyne.TextWrapWord

	// --- Layout ---
	folderRow := container.NewHBox(ui.folderBtn, ui.pathLabel)

	checkCol1 := container.NewVBox(
		ui.dryRunCheck,
		ui.recursiveCheck,
		ui.renameStrayCheck,
	)
	checkCol2 := container.NewVBox(
		ui.resizeCoverCheck,
		ui.extractCheck,
		ui.resizeEmbeddedCheck,
	)
	checkRow := container.NewHBox(checkCol1, checkCol2)

	artSizeForm := container.NewGridWithColumns(2,
		widget.NewLabel("Art size:"),
		ui.artSizeEntry,
	)
	qualityForm := container.NewGridWithColumns(2,
		widget.NewLabel("JPEG quality:"),
		ui.qualityEntry,
	)
	entryRow := container.NewHBox(artSizeForm, qualityForm)

	transcodeRow := container.NewHBox(widget.NewLabel("Transcode:"), ui.transcodeSelect)

	buttonsRow := container.NewHBox(ui.runBtn, ui.cancelBtn)

	progressBox := container.NewBorder(widget.NewLabel("Progress:"), nil, nil, nil,
		container.NewScroll(ui.progressLog),
	)

	summaryBox := container.NewVBox(
		widget.NewLabel("Summary:"),
		ui.summaryLabel,
		ui.errorLabel,
	)

	content := container.NewVBox(
		folderRow,
		widget.NewSeparator(),
		checkRow,
		entryRow,
		transcodeRow,
		ui.backupCheck,
		widget.NewSeparator(),
		buttonsRow,
		widget.NewSeparator(),
		progressBox,
		summaryBox,
	)

	ui.window.SetContent(content)
	ui.window.Resize(fyne.NewSize(600, 500))
}
