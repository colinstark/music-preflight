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

// isProgressiveJPEG walks the JPEG marker segments looking for the frame header.
// SOF2 (0xFFC2) is progressive; the other SOFn markers are sequential/baseline.
// The first SOFn is decisive and always precedes the entropy-coded scan data.
//
// Segments are skipped by their length field rather than scanning every byte, so
// arbitrary 0xFFCx bytes inside an APPn/DQT payload (e.g. an embedded EXIF
// thumbnail) cannot be mistaken for a frame marker.
func isProgressiveJPEG(b []byte) bool {
	if len(b) < 2 || b[0] != 0xFF || b[1] != 0xD8 { // require SOI
		return false
	}
	i := 2
	for i+1 < len(b) {
		if b[i] != 0xFF {
			return false // not aligned on a marker; malformed header
		}
		// Skip any 0xFF fill bytes preceding the marker code.
		j := i + 1
		for j < len(b) && b[j] == 0xFF {
			j++
		}
		if j >= len(b) {
			return false
		}
		marker := b[j]
		i = j + 1

		switch {
		case marker == 0xC2:
			return true // SOF2 = progressive
		case marker >= 0xC0 && marker <= 0xCF &&
			marker != 0xC4 && marker != 0xC8 && marker != 0xCC:
			return false // a non-progressive SOFn (C4=DHT, C8=JPG, CC=DAC are not SOF)
		case marker == 0x01 || (marker >= 0xD0 && marker <= 0xD9):
			continue // standalone markers (TEM, RSTn, SOI, EOI): no length payload
		case marker == 0xDA:
			return false // SOS: scan data begins; any SOF would have appeared already
		}
		// Marker segment with a 2-byte big-endian length (which includes itself).
		if i+1 >= len(b) {
			return false
		}
		length := int(b[i])<<8 | int(b[i+1])
		if length < 2 {
			return false
		}
		i += length
	}
	return false
}
