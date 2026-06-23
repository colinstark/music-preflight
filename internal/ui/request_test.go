package ui

import (
	"testing"

	"github.com/colinstark/coverfixer/internal/core"
)

// --- DefaultRequest: GUI defaults ---

func TestDefaultRequestValues(t *testing.T) {
	d := core.DefaultOptions()
	r := DefaultRequest()

	if r.ArtSize != d.ArtSize {
		t.Errorf("ArtSize = %d, want %d", r.ArtSize, d.ArtSize)
	}
	if r.JPEGQuality != d.JPEGQuality {
		t.Errorf("JPEGQuality = %d, want %d", r.JPEGQuality, d.JPEGQuality)
	}
	if r.Recursive != d.Recursive {
		t.Errorf("Recursive = %v, want %v", r.Recursive, d.Recursive)
	}
	if r.RenameStrayJPG != d.RenameStrayJPG {
		t.Errorf("RenameStrayJPG = %v, want %v", r.RenameStrayJPG, d.RenameStrayJPG)
	}
	if r.ResizeCoverJPG != d.ResizeCoverJPG {
		t.Errorf("ResizeCoverJPG = %v, want %v", r.ResizeCoverJPG, d.ResizeCoverJPG)
	}
	if r.ExtractCover != d.ExtractCover {
		t.Errorf("ExtractCover = %v, want %v", r.ExtractCover, d.ExtractCover)
	}
	if r.ResizeEmbedded != d.ResizeEmbedded {
		t.Errorf("ResizeEmbedded = %v, want %v", r.ResizeEmbedded, d.ResizeEmbedded)
	}
	if r.Transcode != "none" {
		t.Errorf("Transcode = %q, want %q", r.Transcode, "none")
	}
	// Safety: the GUI must ship with dry-run ON.
	if !r.DryRun {
		t.Error("DryRun default must be true (safety)")
	}
}

// --- Options(): Dir is forwarded ---

func TestOptionsDirForwarded(t *testing.T) {
	dir := "/tmp/test-music-dir"
	opts, err := RunRequest{Dir: dir}.Options()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Dir != dir {
		t.Errorf("Dir = %q, want %q", opts.Dir, dir)
	}
}

// --- Options(): numeric fallback + clamp ---

func TestOptionsArtSizeFallbackAndValue(t *testing.T) {
	t.Run("zero_falls_back", func(t *testing.T) {
		opts, _ := RunRequest{}.Options()
		if opts.ArtSize != 500 {
			t.Errorf("ArtSize = %d, want 500 (fallback)", opts.ArtSize)
		}
	})
	t.Run("negative_falls_back", func(t *testing.T) {
		opts, _ := RunRequest{ArtSize: -5}.Options()
		if opts.ArtSize != 500 {
			t.Errorf("ArtSize = %d, want 500 (fallback for negative)", opts.ArtSize)
		}
	})
	t.Run("valid", func(t *testing.T) {
		opts, _ := RunRequest{ArtSize: 300}.Options()
		if opts.ArtSize != 300 {
			t.Errorf("ArtSize = %d, want 300", opts.ArtSize)
		}
	})
	t.Run("large_ok", func(t *testing.T) {
		opts, _ := RunRequest{ArtSize: 100000}.Options()
		if opts.ArtSize != 100000 {
			t.Errorf("ArtSize = %d, want 100000", opts.ArtSize)
		}
	})
}

func TestOptionsQualityFallbackAndClamp(t *testing.T) {
	t.Run("zero_falls_back", func(t *testing.T) {
		opts, _ := RunRequest{}.Options()
		if opts.JPEGQuality != 85 {
			t.Errorf("JPEGQuality = %d, want 85 (fallback)", opts.JPEGQuality)
		}
	})
	t.Run("valid", func(t *testing.T) {
		opts, _ := RunRequest{JPEGQuality: 70}.Options()
		if opts.JPEGQuality != 70 {
			t.Errorf("JPEGQuality = %d, want 70", opts.JPEGQuality)
		}
	})
	t.Run("max", func(t *testing.T) {
		opts, _ := RunRequest{JPEGQuality: 100}.Options()
		if opts.JPEGQuality != 100 {
			t.Errorf("JPEGQuality = %d, want 100", opts.JPEGQuality)
		}
	})
	t.Run("over_max_clamped", func(t *testing.T) {
		// A quality above 100 is clamped to 100 (max quality) rather than
		// silently resetting to the engine default of 85.
		opts, _ := RunRequest{JPEGQuality: 150}.Options()
		if opts.JPEGQuality != 100 {
			t.Errorf("JPEGQuality = %d, want 100 (clamped from 150)", opts.JPEGQuality)
		}
	})
}

// --- Options(): transcode string mapping ---

func TestOptionsTranscodeMapping(t *testing.T) {
	tests := []struct {
		in      string
		want    core.TranscodeMode
		wantErr bool
	}{
		{"", core.TranscodeNone, false},
		{"none", core.TranscodeNone, false},
		{"mp3-320", core.TranscodeMP3_320, false},
		{"aac-256", core.TranscodeAAC_256, false},
		{"bogus", core.TranscodeNone, true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			opts, err := RunRequest{Transcode: tc.in}.Options()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.in, err)
			}
			if opts.Transcode != tc.want {
				t.Errorf("Transcode = %v, want %v", opts.Transcode, tc.want)
			}
		})
	}
}

// --- Options(): booleans forwarded ---

func TestOptionsBooleansForwarded(t *testing.T) {
	r := RunRequest{
		Recursive: true, RenameStrayJPG: true, ResizeCoverJPG: true,
		ExtractCover: true, ResizeEmbedded: true, Backup: true, DryRun: true,
		SetGenre: true,
	}
	opts, _ := r.Options()
	if !opts.Recursive || !opts.RenameStrayJPG || !opts.ResizeCoverJPG ||
		!opts.ExtractCover || !opts.ResizeEmbedded || !opts.Backup || !opts.DryRun {
		t.Errorf("all booleans should forward true: %+v", opts)
	}
	if !opts.SetGenre {
		t.Error("SetGenre should forward true")
	}

	r2 := RunRequest{}
	opts2, _ := r2.Options()
	if opts2.Recursive || opts2.RenameStrayJPG || opts2.ResizeCoverJPG ||
		opts2.ExtractCover || opts2.ResizeEmbedded || opts2.Backup || opts2.DryRun ||
		opts2.SetGenre {
		t.Errorf("zero-value booleans should forward false: %+v", opts2)
	}
}

// --- Options(): genre forwarded ---

func TestOptionsGenreForwarded(t *testing.T) {
	opts, _ := RunRequest{SetGenre: true, Genre: "Jazz"}.Options()
	if !opts.SetGenre || opts.Genre != "Jazz" {
		t.Errorf("SetGenre/Genre = %v/%q, want true/Jazz", opts.SetGenre, opts.Genre)
	}
}

// --- Options(): composite round-trip (mirrors the original VAL-FORM composite) ---

func TestOptionsCompositeMapping(t *testing.T) {
	r := RunRequest{
		Dir:            "/tmp/composite-test",
		ArtSize:        256,
		JPEGQuality:    90,
		Recursive:      true,
		RenameStrayJPG: true,
		ResizeCoverJPG: true,
		ExtractCover:   true,
		ResizeEmbedded: true,
		Transcode:      "aac-256",
		Backup:         true,
		DryRun:         false,
	}
	opts, err := r.Options()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.Dir != "/tmp/composite-test" {
		t.Errorf("Dir = %q", opts.Dir)
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
	if !opts.ResizeEmbedded || !opts.Backup {
		t.Errorf("ResizeEmbedded/Backup should be true: %+v", opts)
	}
	if opts.DryRun {
		t.Error("DryRun should be false")
	}
}
