package core

import (
	"fmt"
	"strconv"

	mp4tag "github.com/Sorrow446/go-mp4tag"
)

// setM4ATags writes the given text tags into the file's MP4 atoms in place:
// ©nam (title), ©ART (artist), aART (album artist), ©alb (album), ©gen (genre),
// ©day (date/year) and trkn (track number, preserving any existing total). It
// returns whether any tag was changed. Backup and DryRun are honoured via o.
//
// Every managed atom is named in delStrings so a value (including an empty one,
// which clears) replaces rather than appends; all other tags — pictures, disc
// numbers, etc. — are merged through untouched. The ©day atom is special:
// go-mp4tag reads a numeric value back as Year (int32) and free text as Date, so
// we write whichever shape fits and clear both to avoid a stale leftover.
func setM4ATags(o Options, path string, want textTags) (bool, error) {
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return false, fmt.Errorf("open mp4: %w", err)
	}
	defer mp4.Close()

	cur, err := mp4.Read()
	if err != nil {
		return false, fmt.Errorf("read mp4 tags: %w", err)
	}

	if readM4ATextTags(cur) == want {
		return false, nil
	}
	if o.DryRun {
		return true, nil
	}

	out := &mp4tag.MP4Tags{
		Album:       want.Album,
		AlbumArtist: want.AlbumArtist,
		CustomGenre: want.Genre,
		Title:       want.Title,
		Artist:      want.Artist,
		TrackNumber: int16(want.TrackNumber),
	}
	if n, err := strconv.Atoi(want.Year); err == nil && want.Year != "" {
		out.Year = int32(n)
	} else {
		out.Date = want.Year
	}
	del := []string{"album", "albumartist", "customgenre", "year", "date", "title", "artist", "tracknumber"}
	if err := mp4.Write(out, del); err != nil {
		return false, fmt.Errorf("write mp4 tags: %w", err)
	}
	return true, nil
}

// readM4ATextTags projects a go-mp4tag MP4Tags onto the format-agnostic textTags
// for change detection. A numeric ©day is stored by the library as Year; an
// absent track number reads as -1 and is folded to 0.
func readM4ATextTags(cur *mp4tag.MP4Tags) textTags {
	year := cur.Date
	if cur.Year > 0 {
		year = strconv.Itoa(int(cur.Year))
	}
	track := int(cur.TrackNumber)
	if track < 0 {
		track = 0
	}
	return textTags{
		Title:       cur.Title,
		Artist:      cur.Artist,
		AlbumArtist: cur.AlbumArtist,
		Album:       cur.Album,
		Genre:       cur.CustomGenre,
		Year:        year,
		TrackNumber: track,
	}
}

// setM4AAlbumArtist writes o.AlbumArtist into the aART atom (album artist) in
// place, leaving ©ART (track artist) untouched. Returns whether it changed.
func setM4AAlbumArtist(o Options, path string) (bool, error) {
	mp4, err := mp4tag.Open(path)
	if err != nil {
		return false, fmt.Errorf("open mp4: %w", err)
	}
	defer mp4.Close()

	tags, err := mp4.Read()
	if err != nil {
		return false, fmt.Errorf("read mp4 tags: %w", err)
	}
	if tags.AlbumArtist == o.AlbumArtist {
		return false, nil
	}
	if o.DryRun {
		return true, nil
	}

	// "albumartist" clears the existing aART so we replace rather than append;
	// other tags (including ©ART) are merged through untouched.
	if err := mp4.Write(&mp4tag.MP4Tags{AlbumArtist: o.AlbumArtist}, []string{"albumartist"}); err != nil {
		return false, fmt.Errorf("write mp4 tags: %w", err)
	}
	return true, nil
}

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
