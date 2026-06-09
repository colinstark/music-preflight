// Package gui provides the Fyne desktop front-end for coverfixer.
// It drives the core engine through core.Run and exposes full
// core.Options parity controls in a single window.
package gui

import (
	"context"

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
	cancelCtx   context.CancelFunc
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

	// --- Numeric entries ---
	ui.artSizeEntry = widget.NewEntry()
	ui.artSizeEntry.SetText("500")

	ui.qualityEntry = widget.NewEntry()
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
	ui.runBtn = widget.NewButton("Run", nil)
	ui.runBtn.Disable() // disabled until a folder is selected

	ui.cancelBtn = widget.NewButton("Cancel", nil)

	// --- Progress log (scrolling, read-only) ---
	ui.progressLog = widget.NewMultiLineEntry()
	ui.progressLog.SetPlaceHolder("Progress will appear here…")
	ui.progressLog.Disable() // read-only; programmatic SetText still works

	// --- Summary label ---
	ui.summaryLabel = widget.NewLabel("")

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
	backupRow := container.NewHBox(ui.backupCheck)

	buttonsRow := container.NewHBox(ui.runBtn, ui.cancelBtn)

	progressBox := container.NewBorder(widget.NewLabel("Progress:"), nil, nil, nil,
		container.NewScroll(ui.progressLog),
	)

	summaryBox := container.NewVBox(
		widget.NewLabel("Summary:"),
		ui.summaryLabel,
	)

	content := container.NewVBox(
		folderRow,
		widget.NewSeparator(),
		checkRow,
		entryRow,
		transcodeRow,
		backupRow,
		widget.NewSeparator(),
		buttonsRow,
		widget.NewSeparator(),
		progressBox,
		summaryBox,
	)

	ui.window.SetContent(content)
	ui.window.Resize(fyne.NewSize(600, 500))
}
