package core

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// makeJPEG returns a baseline JPEG of the given dimensions filled with a simple
// gradient (so the encoder produces non-trivial content).
func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	return buf.Bytes()
}

// jpegDimensions decodes the header of a JPEG and returns its dimensions.
func jpegDimensions(t *testing.T, data []byte) (int, int) {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode jpeg header: %v", err)
	}
	return cfg.Width, cfg.Height
}
