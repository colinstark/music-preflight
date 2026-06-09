package core

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder for embedded PNG artwork
	"math"

	"golang.org/x/image/draw"
)

// resizeArtwork decodes src (JPEG or PNG), fits it within maxSize×maxSize
// preserving aspect ratio without upscaling, and re-encodes it as a baseline
// JPEG at the given quality. Go's image/jpeg encoder only emits baseline
// (non-progressive) JPEGs, which is exactly what Rockbox-style players want.
func resizeArtwork(src []byte, maxSize, quality int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	b := img.Bounds()
	nw, nh := fitDimensions(b.Dx(), b.Dy(), maxSize)
	if nw == b.Dx() && nh == b.Dy() {
		return encodeBaselineJPEG(img, quality)
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return encodeBaselineJPEG(dst, quality)
}

// fitDimensions returns the largest w×h that fits within maxSize on both axes
// while preserving aspect ratio. It never upscales.
func fitDimensions(w, h, maxSize int) (int, int) {
	if w <= maxSize && h <= maxSize {
		return w, h
	}
	if w >= h {
		nh := int(math.Round(float64(h) * float64(maxSize) / float64(w)))
		if nh < 1 {
			nh = 1
		}
		return maxSize, nh
	}
	nw := int(math.Round(float64(w) * float64(maxSize) / float64(h)))
	if nw < 1 {
		nw = 1
	}
	return nw, maxSize
}

func encodeBaselineJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

// artworkNeedsWork reports whether src should be re-encoded: true if it is not a
// JPEG, if it is a progressive JPEG, or if either dimension exceeds maxSize.
func artworkNeedsWork(src []byte, maxSize int) (bool, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(src))
	if err != nil {
		return false, fmt.Errorf("decode image header: %w", err)
	}
	if format != "jpeg" {
		return true, nil
	}
	if cfg.Width > maxSize || cfg.Height > maxSize {
		return true, nil
	}
	return isProgressiveJPEG(src), nil
}

// isProgressiveJPEG scans the JPEG marker segments for an SOF marker. SOF2
// (0xFFC2) is progressive; SOF0/SOF1 (0xFFC0/0xFFC1) are baseline. The first SOF
// encountered is decisive and always precedes the entropy-coded scan data.
func isProgressiveJPEG(b []byte) bool {
	for i := 0; i+1 < len(b); i++ {
		if b[i] != 0xFF {
			continue
		}
		switch b[i+1] {
		case 0xC2:
			return true
		case 0xC0, 0xC1:
			return false
		}
	}
	return false
}
