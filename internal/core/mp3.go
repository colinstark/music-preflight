package core

import (
	"fmt"

	"github.com/bogem/id3v2/v2"
)

// readMP3Art returns the bytes of the first embedded APIC picture, or nil if the
// file has no embedded artwork.
func readMP3Art(path string) ([]byte, error) {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return nil, fmt.Errorf("open id3: %w", err)
	}
	defer tag.Close()

	for _, f := range tag.GetFrames(tag.CommonID("Attached picture")) {
		if pf, ok := f.(id3v2.PictureFrame); ok {
			return pf.Picture, nil
		}
	}
	return nil, nil
}

// resizeMP3Art resizes every embedded APIC picture that exceeds the target size
// (or is not a baseline JPEG) and rewrites the tag in place. It returns whether
// anything was changed. Backup and DryRun are honoured via o.
func resizeMP3Art(o Options, path string) (bool, error) {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return false, fmt.Errorf("open id3: %w", err)
	}
	defer tag.Close()

	apicID := tag.CommonID("Attached picture")
	frames := tag.GetFrames(apicID)
	if len(frames) == 0 {
		return false, nil
	}

	var rebuilt []id3v2.PictureFrame
	changed := false
	for _, f := range frames {
		pf, ok := f.(id3v2.PictureFrame)
		if !ok {
			continue
		}
		need, err := artworkNeedsWork(pf.Picture, o.ArtSize)
		if err != nil {
			return false, err
		}
		if need {
			resized, err := resizeArtwork(pf.Picture, o.ArtSize, o.JPEGQuality)
			if err != nil {
				return false, err
			}
			pf.Picture = resized
			pf.MimeType = "image/jpeg"
			changed = true
		}
		rebuilt = append(rebuilt, pf)
	}
	if !changed {
		return false, nil
	}
	if o.DryRun {
		return true, nil
	}

	if err := maybeBackup(o, path); err != nil {
		return false, err
	}
	tag.DeleteFrames(apicID)
	for _, pf := range rebuilt {
		tag.AddAttachedPicture(pf)
	}
	if err := tag.Save(); err != nil {
		return false, fmt.Errorf("save id3: %w", err)
	}
	return true, nil
}
