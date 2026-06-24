package core

import "path/filepath"

// textTags is the format-agnostic set of writable text tags for one file. The
// per-format setters write every field (empty clears it), so callers compose
// album-level + track-level values into one textTags before writing.
type textTags struct {
	Album       string
	AlbumArtist string
	Genre       string
	Year        string
	Title       string
	Artist      string
	TrackNumber int
}

// fileEdit is a single file's merged tag edit — the album-level fields shared
// across the album plus that file's own track-level fields — produced by
// editsByFolder and consumed by the tag-edit pass.
type fileEdit struct {
	path string
	want textTags
}

// setTags writes want into path (format-dispatched), recording the outcome on
// rep. Unsupported formats are a no-op. It mirrors setGenre's shape.
func setTags(o Options, path string, want textTags, rep *reportAccum, progress func(Event)) {
	var (
		changed bool
		err     error
	)
	switch classifyAudio(path) {
	case audioMP3:
		changed, err = setMP3Tags(o, path, want)
	case audioM4A:
		changed, err = setM4ATags(o, path, want)
	default:
		return
	}
	if err != nil {
		rep.fail(progress, "set-tags", path, err)
		return
	}
	if changed {
		rep.action(progress, "set-tags", path, want.Title)
		rep.inc(&rep.TagsEdited)
	}
}

// editsByFolder flattens o.TagEdits into per-folder file edits. Each track's
// absolute path resolves to its folder, and each entry carries the merged
// album-level + track-level text tags to write. The result lets the runner
// apply tag edits within the per-folder loop so ordering relative to the genre
// (before) and transcode (after) passes stays correct.
func editsByFolder(o Options) map[string][]fileEdit {
	if len(o.TagEdits) == 0 {
		return nil
	}
	by := map[string][]fileEdit{}
	for _, ed := range o.TagEdits {
		for _, tr := range ed.Tracks {
			if tr.Path == "" {
				continue
			}
			by[filepath.Dir(tr.Path)] = append(by[filepath.Dir(tr.Path)], fileEdit{
				path: tr.Path,
				want: textTags{
					Album:       ed.Album,
					AlbumArtist: ed.AlbumArtist,
					Genre:       ed.Genre,
					Year:        ed.Year,
					Title:       tr.Title,
					Artist:      tr.Artist,
					TrackNumber: tr.TrackNumber,
				},
			})
		}
	}
	return by
}
