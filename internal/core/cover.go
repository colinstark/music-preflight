package core

import (
	"os"
	"path/filepath"
	"strings"
)

// readEmbeddedArt returns the first embedded cover image for an audio file, or
// nil if the format is unsupported or the file has no artwork.
func readEmbeddedArt(path string) ([]byte, error) {
	switch classifyAudio(path) {
	case audioMP3:
		return readMP3Art(path)
	case audioM4A:
		return readM4AArt(path)
	default:
		return nil, nil
	}
}

// processJPGs renames a stray *.jpg to cover.jpg (when the folder has none) and
// resizes cover.jpg to the target baseline JPEG.
func processJPGs(o Options, f *albumFolder, rep *Report, progress func(Event)) {
	hasCover := f.hasCover
	for _, jpg := range f.jpgs {
		cur := jpg
		isCover := strings.EqualFold(filepath.Base(jpg), "cover.jpg")

		if !isCover && o.RenameStrayJPG && !hasCover {
			dst := coverPath(f.dir)
			rep.action(progress, "rename", jpg, "→ cover.jpg")
			if !o.DryRun {
				if err := os.Rename(jpg, dst); err != nil {
					rep.fail(progress, "rename", jpg, err)
					continue
				}
			}
			rep.Renamed++
			cur = dst
			hasCover = true
			isCover = true
		}

		if isCover && o.ResizeCoverJPG {
			resizeCoverFile(o, cur, rep, progress)
		}
	}
}

func resizeCoverFile(o Options, path string, rep *Report, progress func(Event)) {
	if !fileExists(path) {
		// Reachable only under DryRun after a simulated rename; count the
		// resize we would perform on the renamed cover.
		rep.action(progress, "resize-cover", path, "(after rename)")
		rep.CoversResized++
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		rep.fail(progress, "resize-cover", path, err)
		return
	}
	need, err := artworkNeedsWork(data, o.ArtSize)
	if err != nil {
		rep.fail(progress, "resize-cover", path, err)
		return
	}
	if !need {
		rep.skip(progress, "resize-cover", path, "already within size, baseline")
		return
	}

	resized, err := resizeArtwork(data, o.ArtSize, o.JPEGQuality)
	if err != nil {
		rep.fail(progress, "resize-cover", path, err)
		return
	}
	rep.action(progress, "resize-cover", path, "")
	if !o.DryRun {
		if err := maybeBackup(o, path); err != nil {
			rep.fail(progress, "resize-cover", path, err)
			return
		}
		if err := writeFileAtomic(path, resized); err != nil {
			rep.fail(progress, "resize-cover", path, err)
			return
		}
	}
	rep.CoversResized++
}

// extractCover writes cover.jpg into a folder from the first audio file that
// carries embedded artwork.
func extractCover(o Options, f *albumFolder, rep *Report, progress func(Event)) {
	dst := coverPath(f.dir)
	for _, a := range f.audio {
		art, err := readEmbeddedArt(a)
		if err != nil {
			rep.fail(progress, "extract", a, err)
			continue
		}
		if art == nil {
			continue
		}
		resized, err := resizeArtwork(art, o.ArtSize, o.JPEGQuality)
		if err != nil {
			rep.fail(progress, "extract", a, err)
			continue
		}
		rep.action(progress, "extract", dst, "from "+filepath.Base(a))
		if !o.DryRun {
			if err := writeFileAtomic(dst, resized); err != nil {
				rep.fail(progress, "extract", dst, err)
				return
			}
		}
		rep.Extracted++
		f.hasCover = true
		return
	}
}
