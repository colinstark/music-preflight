package core

import (
	"fmt"

	mp4tag "github.com/Sorrow446/go-mp4tag"
)

// readM4AArt returns the bytes of the first embedded cover (covr) picture, or
// nil if the file has no embedded artwork.
func readM4AArt(path string) ([]byte, error) {
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open mp4: %w", err)
	}
	defer mp4.Close()

	tags, err := mp4.Read()
	if err != nil {
		return nil, fmt.Errorf("read mp4 tags: %w", err)
	}
	if len(tags.Pictures) == 0 {
		return nil, nil
	}
	return tags.Pictures[0].Data, nil
}

// resizeM4AArt resizes every embedded cover that exceeds the target size (or is
// not a baseline JPEG) and rewrites the covr atom in place, preserving the
// file's other tags. It returns whether anything was changed.
func resizeM4AArt(o Options, path string) (bool, error) {
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return false, fmt.Errorf("open mp4: %w", err)
	}
	defer mp4.Close()

	tags, err := mp4.Read()
	if err != nil {
		return false, fmt.Errorf("read mp4 tags: %w", err)
	}
	if len(tags.Pictures) == 0 {
		return false, nil
	}

	rebuilt := make([]*mp4tag.MP4Picture, 0, len(tags.Pictures))
	changed := false
	for _, p := range tags.Pictures {
		need, err := artworkNeedsWork(p.Data, o.ArtSize)
		if err != nil {
			return false, err
		}
		if need {
			resized, err := resizeArtwork(p.Data, o.ArtSize, o.JPEGQuality)
			if err != nil {
				return false, err
			}
			rebuilt = append(rebuilt, &mp4tag.MP4Picture{Format: mp4tag.ImageTypeJPEG, Data: resized})
			changed = true
		} else {
			rebuilt = append(rebuilt, p)
		}
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
	// go-mp4tag appends Pictures rather than replacing them, so clear the
	// existing covr atom with "allpictures" before writing the rebuilt set.
	// Other tags are left untouched (merged).
	if err := mp4.Write(&mp4tag.MP4Tags{Pictures: rebuilt}, []string{"allpictures"}); err != nil {
		return false, fmt.Errorf("write mp4 tags: %w", err)
	}
	return true, nil
}
