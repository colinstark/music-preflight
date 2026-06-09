// Package core is the front-end-agnostic engine for coverfixer. It batch-fixes
// cover art in a music library: resizing embedded artwork inside MP3/M4A files,
// generating and resizing folder cover.jpg files, and optionally transcoding
// audio. The CLI and the (future) Fyne GUI both drive this package via Run.
package core

import "fmt"

// TranscodeMode selects the optional audio conversion applied to every audio file.
type TranscodeMode int

const (
	// TranscodeNone leaves audio untouched (only artwork is processed).
	TranscodeNone TranscodeMode = iota
	// TranscodeMP3_320 converts audio to MP3 CBR 320 kbps.
	TranscodeMP3_320
	// TranscodeAAC_256 converts audio to AAC ~256 kbps in an M4A container.
	TranscodeAAC_256
)

// ParseTranscodeMode maps a CLI string ("none", "mp3-320", "aac-256") to a mode.
func ParseTranscodeMode(s string) (TranscodeMode, error) {
	switch s {
	case "", "none":
		return TranscodeNone, nil
	case "mp3-320":
		return TranscodeMP3_320, nil
	case "aac-256":
		return TranscodeAAC_256, nil
	default:
		return TranscodeNone, fmt.Errorf("invalid transcode mode %q (want none|mp3-320|aac-256)", s)
	}
}

func (m TranscodeMode) String() string {
	switch m {
	case TranscodeMP3_320:
		return "mp3-320"
	case TranscodeAAC_256:
		return "aac-256"
	default:
		return "none"
	}
}

// Options controls a single Run. The zero value is not usable; build one with
// DefaultOptions and adjust, or set fields directly and call applyDefaults via Run.
type Options struct {
	Dir string // root folder to process

	ArtSize     int // max width/height for artwork; images fit within ArtSize×ArtSize
	JPEGQuality int // baseline JPEG quality (1-100)

	Recursive      bool // descend into subfolders
	RenameStrayJPG bool // rename a lone non-cover *.jpg to cover.jpg
	ResizeCoverJPG bool // resize existing cover.jpg to ArtSize baseline JPEG
	ExtractCover   bool // write cover.jpg from embedded art when a folder lacks one
	ResizeEmbedded bool // resize artwork embedded inside audio files, in place

	Transcode TranscodeMode // optional audio conversion

	Backup bool // write a <file>.bak copy before mutating a file
	DryRun bool // report intended actions without changing anything
}

// DefaultOptions returns the recommended defaults: 500×500 baseline JPEG q85,
// recursive, with the script-parity artwork passes enabled.
func DefaultOptions() Options {
	return Options{
		ArtSize:        500,
		JPEGQuality:    85,
		Recursive:      true,
		RenameStrayJPG: true,
		ResizeCoverJPG: true,
		ExtractCover:   true,
		ResizeEmbedded: false,
		Transcode:      TranscodeNone,
	}
}

func (o *Options) applyDefaults() {
	if o.ArtSize <= 0 {
		o.ArtSize = 500
	}
	if o.JPEGQuality <= 0 || o.JPEGQuality > 100 {
		o.JPEGQuality = 85
	}
}
