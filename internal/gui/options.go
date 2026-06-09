package gui

import (
	"strconv"

	"github.com/colinstark/coverfixer/internal/core"
)

// options reads the current widget state and returns a core.Options.
// Dir is set from the selected folder; transcode is mapped via
// core.ParseTranscodeMode; empty or non-numeric art-size and quality
// entries fall back to the engine defaults (500 and 85 respectively).
func (ui *UI) options() core.Options {
	defs := core.DefaultOptions()

	artSize := defs.ArtSize // 500
	if v, err := strconv.Atoi(ui.artSizeEntry.Text); err == nil && v > 0 {
		artSize = v
	}

	quality := defs.JPEGQuality // 85
	if v, err := strconv.Atoi(ui.qualityEntry.Text); err == nil && v > 0 {
		quality = v
	}

	mode, _ := core.ParseTranscodeMode(ui.transcodeSelect.Selected)

	return core.Options{
		Dir:            ui.selectedDir,
		ArtSize:        artSize,
		JPEGQuality:    quality,
		Recursive:      ui.recursiveCheck.Checked,
		RenameStrayJPG: ui.renameStrayCheck.Checked,
		ResizeCoverJPG: ui.resizeCoverCheck.Checked,
		ExtractCover:   ui.extractCheck.Checked,
		ResizeEmbedded: ui.resizeEmbeddedCheck.Checked,
		Transcode:      mode,
		Backup:         ui.backupCheck.Checked,
		DryRun:         ui.dryRunCheck.Checked,
	}
}
