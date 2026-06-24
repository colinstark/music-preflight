package ui

import (
	"fmt"

	"github.com/colinstark/coverfixer/internal/core"
)

// RunRequest is the wire format the frontend sends to start a run. It uses
// only primitive, JSON-friendly types (strings, ints, bools) so Wails can
// generate TypeScript bindings directly. Options validates and coerces it into
// a core.Options for the engine.
//
// Keeping this conversion server-side (rather than building core.Options in
// JS) preserves the fallback/clamping rules of the original UI verbatim and
// gives them a Go unit-test home.
type RunRequest struct {
	Dir            string         `json:"dir"`
	ArtSize        int            `json:"artSize"`      // EMBEDDED artwork max dimension
	CoverJPGSize   int            `json:"coverJpgSize"` // cover.jpg max dimension (resize/extract)
	JPEGQuality    int            `json:"jpegQuality"`
	Recursive      bool           `json:"recursive"`
	RenameStrayJPG bool           `json:"renameStrayJpg"`
	ResizeCoverJPG bool           `json:"resizeCoverJpg"`
	ExtractCover   bool           `json:"extractCover"`
	ResizeEmbedded bool           `json:"resizeEmbedded"`
	Transcode      string         `json:"transcode"` // "none" | "<mp3|aac>-<320|256|192>"
	SetGenre       bool           `json:"setGenre"`
	Genre          string         `json:"genre"`
	SetAlbumArtist bool           `json:"setAlbumArtist"`
	AlbumArtist    string         `json:"albumArtist"`
	TagEdits       []core.TagEdit `json:"tagEdits"` // staged per-album/per-track edits, applied on Run
	Backup         bool           `json:"backup"`
	DryRun         bool           `json:"dryRun"`
}

// DefaultRequest returns the GUI defaults. Unlike core.DefaultOptions, the GUI
// ships with DryRun ON as a safety measure (mirrors the original Fyne UI, so a
// first-time user cannot accidentally mutate their library on the very first
// Run click).
func DefaultRequest() RunRequest {
	d := core.DefaultOptions()
	return RunRequest{
		ArtSize:        d.ArtSize,
		CoverJPGSize:   d.CoverJPGSize,
		JPEGQuality:    d.JPEGQuality,
		Recursive:      d.Recursive,
		RenameStrayJPG: d.RenameStrayJPG,
		ResizeCoverJPG: d.ResizeCoverJPG,
		ExtractCover:   d.ExtractCover,
		ResizeEmbedded: d.ResizeEmbedded,
		Transcode:      d.Transcode.String(),
		DryRun:         true,
	}
}

// Options converts the request into a validated core.Options. Art size and
// quality fall back to engine defaults when non-positive (so an empty/invalid
// entry on the frontend can be sent as 0); quality above 100 is clamped to 100
// (honor max-quality intent rather than resetting to the default). An unknown
// transcode string is an error.
func (r RunRequest) Options() (core.Options, error) {
	defs := core.DefaultOptions()

	artSize := defs.ArtSize
	if r.ArtSize > 0 {
		artSize = r.ArtSize
	}

	coverSize := defs.CoverJPGSize
	if r.CoverJPGSize > 0 {
		coverSize = r.CoverJPGSize
	}

	quality := defs.JPEGQuality
	if r.JPEGQuality > 0 {
		q := r.JPEGQuality
		if q > 100 {
			q = 100
		}
		quality = q
	}

	mode, err := core.ParseTranscodeMode(r.Transcode)
	if err != nil {
		return core.Options{}, fmt.Errorf("transcode: %w", err)
	}

	return core.Options{
		Dir:            r.Dir,
		ArtSize:        artSize,
		CoverJPGSize:   coverSize,
		JPEGQuality:    quality,
		Recursive:      r.Recursive,
		RenameStrayJPG: r.RenameStrayJPG,
		ResizeCoverJPG: r.ResizeCoverJPG,
		ExtractCover:   r.ExtractCover,
		ResizeEmbedded: r.ResizeEmbedded,
		Transcode:      mode,
		SetGenre:       r.SetGenre,
		Genre:          r.Genre,
		SetAlbumArtist: r.SetAlbumArtist,
		AlbumArtist:    r.AlbumArtist,
		TagEdits:       r.TagEdits,
		Backup:         r.Backup,
		DryRun:         r.DryRun,
	}, nil
}
