package core

import "testing"

func TestFitDimensions(t *testing.T) {
	cases := []struct {
		name         string
		w, h, max    int
		wantW, wantH int
	}{
		{"already small", 100, 100, 500, 100, 100},
		{"exact", 500, 500, 500, 500, 500},
		{"landscape downscale", 1000, 500, 500, 500, 250},
		{"portrait downscale", 500, 1000, 500, 250, 500},
		{"square downscale", 2000, 2000, 500, 500, 500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gw, gh := fitDimensions(c.w, c.h, c.max)
			if gw != c.wantW || gh != c.wantH {
				t.Errorf("fitDimensions(%d,%d,%d) = %d,%d; want %d,%d",
					c.w, c.h, c.max, gw, gh, c.wantW, c.wantH)
			}
		})
	}
}

func TestResizeArtworkDownscales(t *testing.T) {
	src := makeJPEG(t, 1200, 800)
	out, err := resizeArtwork(src, 500, 85)
	if err != nil {
		t.Fatalf("resizeArtwork: %v", err)
	}
	w, h := jpegDimensions(t, out)
	if w > 500 || h > 500 {
		t.Errorf("resized to %dx%d, want both <= 500", w, h)
	}
	if w != 500 {
		t.Errorf("long edge = %d, want 500", w)
	}
	if isProgressiveJPEG(out) {
		t.Error("output is progressive; want baseline")
	}
}

func TestIsProgressiveJPEG(t *testing.T) {
	// Minimal hand-built marker streams (image/jpeg only emits baseline, so we
	// can't synthesize a real progressive file via the encoder).
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{
			// SOF2 frame header → progressive.
			name: "progressive sof2",
			data: []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x02, 0xFF, 0xC2, 0x00, 0x11},
			want: true,
		},
		{
			// APP0 payload contains the bytes FF C2 (a trap for a naive byte
			// scanner), but the real frame header is SOF0 → baseline.
			name: "baseline with ffc2 in app0 payload",
			data: []byte{
				0xFF, 0xD8, // SOI
				0xFF, 0xE0, 0x00, 0x06, 0xFF, 0xC2, 0x00, 0x00, // APP0, payload holds FF C2
				0xFF, 0xC0, 0x00, 0x11, // SOF0
			},
			want: false,
		},
		{
			name: "not a jpeg",
			data: []byte{0x89, 0x50, 0x4E, 0x47},
			want: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isProgressiveJPEG(c.data); got != c.want {
				t.Errorf("isProgressiveJPEG = %v, want %v", got, c.want)
			}
		})
	}
}

func TestArtworkNeedsWork(t *testing.T) {
	big := makeJPEG(t, 1000, 1000)
	if need, err := artworkNeedsWork(big, 500); err != nil || !need {
		t.Errorf("oversized art: need=%v err=%v, want need=true", need, err)
	}
	small := makeJPEG(t, 300, 300)
	if need, err := artworkNeedsWork(small, 500); err != nil || need {
		t.Errorf("small baseline art: need=%v err=%v, want need=false", need, err)
	}
}
