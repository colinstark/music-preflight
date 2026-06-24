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
	// TranscodeMP3_256 converts audio to MP3 CBR 256 kbps.
	TranscodeMP3_256
	// TranscodeMP3_192 converts audio to MP3 CBR 192 kbps.
	TranscodeMP3_192
	// TranscodeAAC_320 converts audio to AAC ~320 kbps in an M4A container.
	TranscodeAAC_320
	// TranscodeAAC_256 converts audio to AAC ~256 kbps in an M4A container.
	TranscodeAAC_256
	// TranscodeAAC_192 converts audio to AAC ~192 kbps in an M4A container.
	TranscodeAAC_192
)

// ParseTranscodeMode maps a "<format>-<bitrate>" string ("none", "mp3-320",
// "aac-256", ...) to a mode. The bitrate is one of 320/256/192.
func ParseTranscodeMode(s string) (TranscodeMode, error) {
	switch s {
	case "", "none":
		return TranscodeNone, nil
	case "mp3-320":
		return TranscodeMP3_320, nil
	case "mp3-256":
		return TranscodeMP3_256, nil
	case "mp3-192":
		return TranscodeMP3_192, nil
	case "aac-320":
		return TranscodeAAC_320, nil
	case "aac-256":
		return TranscodeAAC_256, nil
	case "aac-192":
		return TranscodeAAC_192, nil
	default:
		return TranscodeNone, fmt.Errorf("invalid transcode mode %q (want none|<mp3|aac>-<320|256|192>)", s)
	}
}

func (m TranscodeMode) String() string {
	switch m {
	case TranscodeMP3_320:
		return "mp3-320"
	case TranscodeMP3_256:
		return "mp3-256"
	case TranscodeMP3_192:
		return "mp3-192"
	case TranscodeAAC_320:
		return "aac-320"
	case TranscodeAAC_256:
		return "aac-256"
	case TranscodeAAC_192:
		return "aac-192"
	default:
		return "none"
	}
}

// Options controls a single Run. The zero value is not usable; build one with
// DefaultOptions and adjust, or set fields directly and call applyDefaults via Run.
type Options struct {
	Dir string // root folder to process

	ArtSize      int // max dimension for EMBEDDED artwork (resize-embedded pass)
	CoverJPGSize int // max dimension for cover.jpg operations (resize/extract)
	JPEGQuality  int // baseline JPEG quality (1-100)

	Recursive      bool // descend into subfolders
	RenameStrayJPG bool // rename a lone non-cover *.jpg to cover.jpg
	ResizeCoverJPG bool // resize existing cover.jpg to CoverJPGSize baseline JPEG
	ExtractCover   bool // write cover.jpg from embedded art when a folder lacks one
	ResizeEmbedded bool // resize artwork embedded inside audio files, in place

	Transcode TranscodeMode // optional audio conversion

	SetGenre bool   // set the genre tag on audio files to Genre
	Genre    string // genre string written when SetGenre is true

	// SetAlbumArtist writes AlbumArtist to the ALBUM-ARTIST tag (TPE2 / aART) of
	// every audio file. It deliberately targets only the album-artist frame, so
	// per-track ARTIST tags (TPE1 / ©ART) are left untouched.
	SetAlbumArtist bool
	AlbumArtist    string

	// TagEdits are front-end-staged per-album metadata edits applied during the
	// run, after the genre pass (so a per-album edit overrides the global genre
	// for its files) and before transcode (so tags carry into any output). Empty
	// for normal runs.
	TagEdits []TagEdit

	Backup bool // duplicate the selected folder into a sibling "backup" dir before mutating
	DryRun bool // report intended actions without changing anything
}

// TagEdit is one album's worth of staged metadata edits, supplied by a
// front-end and applied during a Run. The album-level fields (Album,
// AlbumArtist, Genre, Year) are written to every file listed in Tracks; each
// TrackTagEdit supplies the per-file title, artist and track number. Fields are
// written as-is — empty clears the tag — so a front-end should pre-fill its edit
// UI with current values and send back the full intended set.
type TagEdit struct {
	Album       string         `json:"album"`
	AlbumArtist string         `json:"albumArtist"`
	Genre       string         `json:"genre"`
	Year        string         `json:"year"`
	Tracks      []TrackTagEdit `json:"tracks"`
}

// TrackTagEdit is the per-file portion of a TagEdit.
type TrackTagEdit struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	TrackNumber int    `json:"trackNumber"`
}

// DefaultOptions returns the recommended defaults: 500×500 baseline JPEG q85,
// recursive, with the script-parity artwork passes enabled.
func DefaultOptions() Options {
	return Options{
		ArtSize:        500,
		CoverJPGSize:   500,
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
	if o.CoverJPGSize <= 0 {
		o.CoverJPGSize = 500
	}
	if o.JPEGQuality <= 0 || o.JPEGQuality > 100 {
		o.JPEGQuality = 85
	}
}
