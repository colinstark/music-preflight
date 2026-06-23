package core

import (
	"fmt"

	mp4tag "github.com/Sorrow446/go-mp4tag"
)

// readM4AGenre returns the file's free-text genre (the ©gen atom), or "" if unset.
func readM4AGenre(path string) (string, error) {
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return "", fmt.Errorf("open mp4: %w", err)
	}
	defer mp4.Close()

	tags, err := mp4.Read()
	if err != nil {
		return "", fmt.Errorf("read mp4 tags: %w", err)
	}
	return tags.CustomGenre, nil
}

// setM4AGenre writes o.Genre into the ©gen atom in place, preserving the file's
// other tags. It returns whether the tag was changed.
func setM4AGenre(o Options, path string) (bool, error) {
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return false, fmt.Errorf("open mp4: %w", err)
	}
	defer mp4.Close()

	tags, err := mp4.Read()
	if err != nil {
		return false, fmt.Errorf("read mp4 tags: %w", err)
	}
	if tags.CustomGenre == o.Genre {
		return false, nil
	}
	if o.DryRun {
		return true, nil
	}

	// "customgenre" clears the existing ©gen so we replace rather than append;
	// other tags are left untouched (merged).
	if err := mp4.Write(&mp4tag.MP4Tags{CustomGenre: o.Genre}, []string{"customgenre"}); err != nil {
		return false, fmt.Errorf("write mp4 tags: %w", err)
	}
	return true, nil
}

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

	// go-mp4tag appends Pictures rather than replacing them, so clear the
	// existing covr atom with "allpictures" before writing the rebuilt set.
	// Other tags are left untouched (merged).
	if err := mp4.Write(&mp4tag.MP4Tags{Pictures: rebuilt}, []string{"allpictures"}); err != nil {
		return false, fmt.Errorf("write mp4 tags: %w", err)
	}
	return true, nil
}
