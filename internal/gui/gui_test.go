package gui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/colinstark/coverfixer/internal/core"
)

// newTestUI creates a UI with a no-op runner for headless testing.
func newTestUI(t *testing.T) *UI {
	t.Helper()
	app := test.NewApp()
	ui := newWithRunner(app, func(_ context.Context, _ core.Options, _ func(core.Event)) (core.Report, error) {
		return core.Report{}, nil
	})
	return ui
}

// --- VAL-FORM-001: All controls present on window build ---

func TestAllControlsPresent(t *testing.T) {
	ui := newTestUI(t)

	// Folder picker button
	if ui.folderBtn == nil {
		t.Error("folder picker button is nil")
	}
	if ui.folderBtn.Text != "Choose Folder" {
		t.Errorf("folder button text = %q, want %q", ui.folderBtn.Text, "Choose Folder")
	}

	// Selected-path label
	if ui.pathLabel == nil {
		t.Error("path label is nil")
	}

	// Six option checks
	checks := map[string]*widget.Check{
		"Dry-run":          ui.dryRunCheck,
		"Recursive":        ui.recursiveCheck,
		"Rename stray jpg": ui.renameStrayCheck,
		"Resize cover.jpg": ui.resizeCoverCheck,
		"Extract cover":    ui.extractCheck,
		"Resize embedded":  ui.resizeEmbeddedCheck,
	}
	for name, chk := range checks {
		if chk == nil {
			t.Errorf("%s check is nil", name)
		} else if chk.Text != name {
			t.Errorf("check text = %q, want %q", chk.Text, name)
		}
	}

	// Numeric entries
	if ui.artSizeEntry == nil {
		t.Error("art-size entry is nil")
	}
	if ui.qualityEntry == nil {
		t.Error("quality entry is nil")
	}

	// Transcode select
	if ui.transcodeSelect == nil {
		t.Error("transcode select is nil")
	}

	// Backup check
	if ui.backupCheck == nil {
		t.Error("backup check is nil")
	}
	if ui.backupCheck.Text != "Backup" {
		t.Errorf("backup check text = %q, want %q", ui.backupCheck.Text, "Backup")
	}

	// Run button
	if ui.runBtn == nil {
		t.Error("run button is nil")
	}
	if ui.runBtn.Text != "Run" {
		t.Errorf("run button text = %q, want %q", ui.runBtn.Text, "Run")
	}

	// Cancel button
	if ui.cancelBtn == nil {
		t.Error("cancel button is nil")
	}
	if ui.cancelBtn.Text != "Cancel" {
		t.Errorf("cancel button text = %q, want %q", ui.cancelBtn.Text, "Cancel")
	}

	// Progress log and summary exist
	if ui.progressLog == nil {
		t.Error("progress log is nil")
	}
	if ui.summaryLabel == nil {
		t.Error("summary label is nil")
	}
}

// --- VAL-FORM-002: Dry-run check defaults to ON ---

func TestDryRunDefaultOn(t *testing.T) {
	ui := newTestUI(t)
	if !ui.dryRunCheck.Checked {
		t.Error("Dry-run check should default to ON (safety)")
	}
}

// --- VAL-FORM-003: Recursive check defaults to ON ---

func TestRecursiveDefaultOn(t *testing.T) {
	ui := newTestUI(t)
	if !ui.recursiveCheck.Checked {
		t.Error("Recursive check should default to ON")
	}
}

// --- VAL-FORM-004: Rename stray jpg check defaults to ON ---

func TestRenameStrayJPGDefaultOn(t *testing.T) {
	ui := newTestUI(t)
	if !ui.renameStrayCheck.Checked {
		t.Error("Rename stray jpg check should default to ON")
	}
}

// --- VAL-FORM-005: Resize cover.jpg check defaults to ON ---

func TestResizeCoverJPGDefaultOn(t *testing.T) {
	ui := newTestUI(t)
	if !ui.resizeCoverCheck.Checked {
		t.Error("Resize cover.jpg check should default to ON")
	}
}

// --- VAL-FORM-006: Extract cover check defaults to ON ---

func TestExtractCoverDefaultOn(t *testing.T) {
	ui := newTestUI(t)
	if !ui.extractCheck.Checked {
		t.Error("Extract cover check should default to ON")
	}
}

// --- VAL-FORM-007: Resize embedded check defaults to OFF ---

func TestResizeEmbeddedDefaultOff(t *testing.T) {
	ui := newTestUI(t)
	if ui.resizeEmbeddedCheck.Checked {
		t.Error("Resize embedded check should default to OFF")
	}
}

// --- VAL-FORM-008: Art-size entry defaults to "500" ---

func TestArtSizeDefault500(t *testing.T) {
	ui := newTestUI(t)
	if ui.artSizeEntry.Text != "500" {
		t.Errorf("Art-size entry text = %q, want %q", ui.artSizeEntry.Text, "500")
	}
}

// --- VAL-FORM-009: JPEG-quality entry defaults to "85" ---

func TestJPEGQualityDefault85(t *testing.T) {
	ui := newTestUI(t)
	if ui.qualityEntry.Text != "85" {
		t.Errorf("JPEG-quality entry text = %q, want %q", ui.qualityEntry.Text, "85")
	}
}

// --- VAL-FORM-010: Transcode select defaults to "none" ---

func TestTranscodeDefaultNone(t *testing.T) {
	ui := newTestUI(t)
	if ui.transcodeSelect.Selected != "none" {
		t.Errorf("Transcode select = %q, want %q", ui.transcodeSelect.Selected, "none")
	}
}

// --- VAL-FORM-011: Transcode select offers exactly [none, mp3-320, aac-256] ---

func TestTranscodeOptions(t *testing.T) {
	ui := newTestUI(t)
	want := []string{"none", "mp3-320", "aac-256"}
	got := ui.transcodeSelect.Options
	if len(got) != len(want) {
		t.Fatalf("Transcode options length = %d, want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("Transcode option[%d] = %q, want %q", i, got[i], v)
		}
	}
}

// --- VAL-FORM-012: Backup check defaults to OFF ---

func TestBackupDefaultOff(t *testing.T) {
	ui := newTestUI(t)
	if ui.backupCheck.Checked {
		t.Error("Backup check should default to OFF")
	}
}

// --- VAL-FORM-013: Selected-path label empty before any folder chosen ---

func TestPathLabelEmptyInitially(t *testing.T) {
	ui := newTestUI(t)
	if ui.pathLabel.Text != "" {
		t.Errorf("Path label text = %q, want empty", ui.pathLabel.Text)
	}
}

// --- VAL-FORM-014: Run button disabled with no folder selected ---

func TestRunButtonDisabledInitially(t *testing.T) {
	ui := newTestUI(t)
	if !ui.runBtn.Disabled() {
		t.Error("Run button should be disabled before a folder is selected")
	}
}

// --- VAL-FORM-015: Cancel button present and distinct from Run ---

func TestCancelButtonDistinctFromRun(t *testing.T) {
	ui := newTestUI(t)
	if ui.cancelBtn == nil {
		t.Fatal("Cancel button is nil")
	}
	if ui.runBtn == nil {
		t.Fatal("Run button is nil")
	}
	if ui.cancelBtn == ui.runBtn {
		t.Error("Cancel and Run buttons should be distinct widgets")
	}
	if ui.cancelBtn.Text != "Cancel" {
		t.Errorf("Cancel button text = %q, want %q", ui.cancelBtn.Text, "Cancel")
	}
}

// --- VAL-FORM-016: Selecting a folder updates the displayed path label ---

func TestFolderSelectionUpdatesLabel(t *testing.T) {
	ui := newTestUI(t)
	dir := "/tmp/test-music"
	ui.setSelectedFolder(dir)
	if ui.pathLabel.Text != dir {
		t.Errorf("path label = %q, want %q", ui.pathLabel.Text, dir)
	}
}

// --- VAL-FORM-017: Selecting a folder enables the Run button ---

func TestFolderSelectionEnablesRun(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test-music")
	if ui.runBtn.Disabled() {
		t.Error("Run button should be enabled after selecting a folder")
	}
}

// --- VAL-FORM-018: Empty folder selection keeps Run disabled ---

func TestEmptyFolderSelectionKeepsRunDisabled(t *testing.T) {
	ui := newTestUI(t)
	// First select a folder to enable Run, then select empty
	ui.setSelectedFolder("/tmp/test")
	if ui.runBtn.Disabled() {
		t.Fatal("Run should be enabled after selecting a folder (prerequisite)")
	}
	// Now simulate user cancelling the dialog (empty selection)
	ui.setSelectedFolder("")
	if !ui.runBtn.Disabled() {
		t.Error("Run button should be disabled after empty folder selection")
	}
	if ui.pathLabel.Text != "" {
		t.Errorf("path label = %q, want empty after empty selection", ui.pathLabel.Text)
	}
}

// --- VAL-FORM-019: options() Dir reflects the selected folder ---

func TestOptionsDirFromSelectedFolder(t *testing.T) {
	ui := newTestUI(t)
	dir := "/tmp/test-music-dir"
	ui.setSelectedFolder(dir)
	opts := ui.options()
	if opts.Dir != dir {
		t.Errorf("options().Dir = %q, want %q", opts.Dir, dir)
	}
}

// --- VAL-FORM-020: options() maps default widget state to default-equivalent Options ---

func TestOptionsDefaultMapping(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	opts := ui.options()

	defs := core.DefaultOptions()
	if opts.DryRun != true {
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
	if opts.Backup != false {
		t.Error("Backup should be false by default")
	}
}

// --- VAL-FORM-021: Toggling Dry-run off maps to DryRun=false ---

func TestToggleDryRunOff(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.dryRunCheck.SetChecked(false)
	opts := ui.options()
	if opts.DryRun {
		t.Error("DryRun should be false after unchecking Dry-run")
	}
}

// --- VAL-FORM-022: Toggling Recursive off maps to Recursive=false ---

func TestToggleRecursiveOff(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.recursiveCheck.SetChecked(false)
	opts := ui.options()
	if opts.Recursive {
		t.Error("Recursive should be false after unchecking Recursive")
	}
}

// --- VAL-FORM-023: Toggling Rename stray jpg off maps to RenameStrayJPG=false ---

func TestToggleRenameStrayJPGOff(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.renameStrayCheck.SetChecked(false)
	opts := ui.options()
	if opts.RenameStrayJPG {
		t.Error("RenameStrayJPG should be false after unchecking")
	}
}

// --- VAL-FORM-024: Toggling Resize cover.jpg off maps to ResizeCoverJPG=false ---

func TestToggleResizeCoverJPGOff(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.resizeCoverCheck.SetChecked(false)
	opts := ui.options()
	if opts.ResizeCoverJPG {
		t.Error("ResizeCoverJPG should be false after unchecking")
	}
}

// --- VAL-FORM-025: Toggling Extract cover off maps to ExtractCover=false ---

func TestToggleExtractCoverOff(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.extractCheck.SetChecked(false)
	opts := ui.options()
	if opts.ExtractCover {
		t.Error("ExtractCover should be false after unchecking")
	}
}

// --- VAL-FORM-026: Toggling Resize embedded on maps to ResizeEmbedded=true ---

func TestToggleResizeEmbeddedOn(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.resizeEmbeddedCheck.SetChecked(true)
	opts := ui.options()
	if !opts.ResizeEmbedded {
		t.Error("ResizeEmbedded should be true after checking")
	}
}

// --- VAL-FORM-027: Toggling Backup on maps to Backup=true ---

func TestToggleBackupOn(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.backupCheck.SetChecked(true)
	opts := ui.options()
	if !opts.Backup {
		t.Error("Backup should be true after checking")
	}
}

// --- VAL-FORM-028: Art-size entry maps a valid integer to ArtSize ---

func TestArtSizeValidInteger(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.artSizeEntry.SetText("300")
	opts := ui.options()
	if opts.ArtSize != 300 {
		t.Errorf("ArtSize = %d, want 300", opts.ArtSize)
	}
}

// --- VAL-FORM-029: Empty Art-size entry falls back to engine default ---

func TestArtSizeEmptyFallback(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.artSizeEntry.SetText("")
	opts := ui.options()
	if opts.ArtSize != 500 {
		t.Errorf("ArtSize = %d, want 500 (engine default)", opts.ArtSize)
	}
}

// --- VAL-FORM-030: Invalid (non-numeric) Art-size entry falls back to engine default ---

func TestArtSizeInvalidFallback(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.artSizeEntry.SetText("abc")
	opts := ui.options()
	if opts.ArtSize != 500 {
		t.Errorf("ArtSize = %d, want 500 (fallback for invalid)", opts.ArtSize)
	}
}

// --- VAL-FORM-031: JPEG-quality entry maps a valid integer to JPEGQuality ---

func TestJPEGQualityValidInteger(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.qualityEntry.SetText("70")
	opts := ui.options()
	if opts.JPEGQuality != 70 {
		t.Errorf("JPEGQuality = %d, want 70", opts.JPEGQuality)
	}
}

// --- VAL-FORM-032: Empty JPEG-quality entry falls back to engine default ---

func TestJPEGQualityEmptyFallback(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.qualityEntry.SetText("")
	opts := ui.options()
	if opts.JPEGQuality != 85 {
		t.Errorf("JPEGQuality = %d, want 85 (engine default)", opts.JPEGQuality)
	}
}

// --- VAL-FORM-033: Invalid (non-numeric) JPEG-quality entry falls back to engine default ---

func TestJPEGQualityInvalidFallback(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.qualityEntry.SetText("xx")
	opts := ui.options()
	if opts.JPEGQuality != 85 {
		t.Errorf("JPEGQuality = %d, want 85 (fallback for invalid)", opts.JPEGQuality)
	}
}

// --- VAL-FORM-034: Transcode "none" maps to TranscodeNone ---

func TestTranscodeNone(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.transcodeSelect.SetSelected("none")
	opts := ui.options()
	if opts.Transcode != core.TranscodeNone {
		t.Errorf("Transcode = %v, want TranscodeNone", opts.Transcode)
	}
}

// --- VAL-FORM-035: Transcode "mp3-320" maps to TranscodeMP3_320 ---

func TestTranscodeMP3320(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.transcodeSelect.SetSelected("mp3-320")
	opts := ui.options()
	if opts.Transcode != core.TranscodeMP3_320 {
		t.Errorf("Transcode = %v, want TranscodeMP3_320", opts.Transcode)
	}
}

// --- VAL-FORM-036: Transcode "aac-256" maps to TranscodeAAC_256 ---

func TestTranscodeAAC256(t *testing.T) {
	ui := newTestUI(t)
	ui.setSelectedFolder("/tmp/test")
	ui.transcodeSelect.SetSelected("aac-256")
	opts := ui.options()
	if opts.Transcode != core.TranscodeAAC_256 {
		t.Errorf("Transcode = %v, want TranscodeAAC_256", opts.Transcode)
	}
}

// --- VAL-FORM-037: Multiple simultaneous changes compose into one Options value ---

func TestOptionsCompositeMapping(t *testing.T) {
	ui := newTestUI(t)
	dir := "/tmp/composite-test"
	ui.setSelectedFolder(dir)

	// Change multiple controls
	ui.dryRunCheck.SetChecked(false)
	ui.resizeEmbeddedCheck.SetChecked(true)
	ui.backupCheck.SetChecked(true)
	ui.artSizeEntry.SetText("256")
	ui.qualityEntry.SetText("90")
	ui.transcodeSelect.SetSelected("aac-256")

	opts := ui.options()

	// Assert all changed fields
	if opts.Dir != dir {
		t.Errorf("Dir = %q, want %q", opts.Dir, dir)
	}
	if opts.DryRun {
		t.Error("DryRun should be false")
	}
	if !opts.ResizeEmbedded {
		t.Error("ResizeEmbedded should be true")
	}
	if !opts.Backup {
		t.Error("Backup should be true")
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

	// Untouched controls should still be at defaults
	if !opts.Recursive {
		t.Error("Recursive should still be true (default)")
	}
	if !opts.RenameStrayJPG {
		t.Error("RenameStrayJPG should still be true (default)")
	}
	if !opts.ResizeCoverJPG {
		t.Error("ResizeCoverJPG should still be true (default)")
	}
	if !opts.ExtractCover {
		t.Error("ExtractCover should still be true (default)")
	}
}
