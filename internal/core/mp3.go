package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
)

// readMP3Genre returns the file's genre tag (TCON), or "" if unset.
func readMP3Genre(path string) (string, error) {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return "", fmt.Errorf("open id3: %w", err)
	}
	defer tag.Close()
	return tag.Genre(), nil
}

// setMP3Genre writes o.Genre into the TCON frame in place. It returns whether
// the tag was changed (the new genre differs from the existing one). Backup and
// DryRun are honoured via o.
func setMP3Genre(o Options, path string) (bool, error) {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return false, fmt.Errorf("open id3: %w", err)
	}
	defer tag.Close()

	if tag.Genre() == o.Genre {
		return false, nil
	}
	if o.DryRun {
		return true, nil
	}

	// Delete any existing TCON so SetGenre's AddTextFrame doesn't leave a
	// duplicate frame alongside the old value.
	tag.DeleteFrames(tag.CommonID("Content type"))
	tag.SetGenre(o.Genre)
	if err := tag.Save(); err != nil {
		return false, fmt.Errorf("save id3: %w", err)
	}
	return true, nil
}

// readMP3TextTags returns the file's writable text tags, for change detection.
func readMP3TextTags(t *id3v2.Tag) textTags {
	rawTrck := t.GetTextFrame(t.CommonID("Track number/Position in set")).Text
	return textTags{
		Title:       t.Title(),
		Artist:      t.Artist(),
		AlbumArtist: t.GetTextFrame(t.CommonID("Band/Orchestra/Accompaniment")).Text,
		Album:       t.Album(),
		Genre:       t.Genre(),
		Year:        t.Year(),
		TrackNumber: parseLeadInt(rawTrck),
	}
}

// setMP3Tags writes the given text tags into the file's ID3v2 text frames in
// place: TIT2 (title), TPE1 (artist), TPE2 (album artist), TALB (album), TCON
// (genre), TYER/TDRC (year) and TRCK (track number). It returns whether any
// frame was changed. Backup and DryRun are honoured via o.
//
// Each frame is deleted before being (re-)added, matching the genre pass: the
// id3v2 Set*/AddTextFrame helpers are additive, so deleting first keeps every
// frame single-valued. A track number keeps any existing "/total" suffix.
func setMP3Tags(o Options, path string, want textTags) (bool, error) {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return false, fmt.Errorf("open id3: %w", err)
	}
	defer tag.Close()

	rawTrck := tag.GetTextFrame(tag.CommonID("Track number/Position in set")).Text
	cur := readMP3TextTags(tag)
	if cur == want {
		return false, nil
	}
	if o.DryRun {
		return true, nil
	}

	enc := tag.DefaultEncoding()
	setText := func(commonID, value string) {
		tag.DeleteFrames(commonID)
		tag.AddTextFrame(commonID, enc, value)
	}
	setText(tag.CommonID("Title"), want.Title)
	setText(tag.CommonID("Artist"), want.Artist)
	setText(tag.CommonID("Band/Orchestra/Accompaniment"), want.AlbumArtist)
	setText(tag.CommonID("Album/Movie/Show title"), want.Album)
	setText(tag.CommonID("Content type"), want.Genre)
	setText(tag.CommonID("Year"), want.Year)
	setText(tag.CommonID("Track number/Position in set"), formatTRCK(want.TrackNumber, rawTrck))

	if err := tag.Save(); err != nil {
		return false, fmt.Errorf("save id3: %w", err)
	}
	return true, nil
}

// formatTRCK renders an ID3v2 TRCK value, preserving any "/total" suffix from
// the existing frame (e.g. "3/12" becomes "5/12"). A non-positive number clears
// the frame entirely.
func formatTRCK(n int, prev string) string {
	if n <= 0 {
		return ""
	}
	if i := strings.IndexByte(prev, '/'); i >= 0 {
		return strconv.Itoa(n) + prev[i:]
	}
	return strconv.Itoa(n)
}

// setMP3AlbumArtist writes o.AlbumArtist into the TPE2 frame (album artist) in
// place, leaving TPE1 (track artist) untouched. Returns whether it changed.
// Backup and DryRun are honoured via o.
func setMP3AlbumArtist(o Options, path string) (bool, error) {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return false, fmt.Errorf("open id3: %w", err)
	}
	defer tag.Close()

	id := tag.CommonID("Band/Orchestra/Accompaniment")
	if tag.GetTextFrame(id).Text == o.AlbumArtist {
		return false, nil
	}
	if o.DryRun {
		return true, nil
	}

	tag.DeleteFrames(id)
	tag.AddTextFrame(id, tag.DefaultEncoding(), o.AlbumArtist)
	if err := tag.Save(); err != nil {
		return false, fmt.Errorf("save id3: %w", err)
	}
	return true, nil
}

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

	tag.DeleteFrames(apicID)
	for _, pf := range rebuilt {
		tag.AddAttachedPicture(pf)
	}
	if err := tag.Save(); err != nil {
		return false, fmt.Errorf("save id3: %w", err)
	}
	return true, nil
}
