package core

import (
	"errors"
	"io/fs"
	"path/filepath"
)

// errStopWalk is a sentinel returned from a WalkDir callback to stop the walk
// early once a result is found; WalkDir's caller ignores it.
var errStopWalk = errors.New("stop walk")

// readGenre returns the genre tag of an audio file, or "" for unsupported or
// untagged files. It mirrors readEmbeddedArt's format dispatch.
func readGenre(path string) (string, error) {
	switch classifyAudio(path) {
	case audioMP3:
		return readMP3Genre(path)
	case audioM4A:
		return readM4AGenre(path)
	default:
		return "", nil
	}
}

// setGenre writes o.Genre into an audio file's tag in place (format-dispatched),
// recording the outcome on rep. Unsupported formats are a no-op.
func setGenre(o Options, path string, rep *reportAccum, progress func(Event)) {
	var (
		changed bool
		err     error
	)
	switch classifyAudio(path) {
	case audioMP3:
		changed, err = setMP3Genre(o, path)
	case audioM4A:
		changed, err = setM4AGenre(o, path)
	default:
		return
	}
	if err != nil {
		rep.fail(progress, "set-genre", path, err)
		return
	}
	if changed {
		rep.action(progress, "set-genre", path, o.Genre)
		rep.inc(&rep.GenresSet)
	}
}

// setAlbumArtist writes o.AlbumArtist into an audio file's ALBUM-ARTIST tag in
// place (format-dispatched), recording the outcome on rep. Only the album-artist
// frame is written; per-track artist tags are never touched. Unsupported formats
// are a no-op.
func setAlbumArtist(o Options, path string, rep *reportAccum, progress func(Event)) {
	var (
		changed bool
		err     error
	)
	switch classifyAudio(path) {
	case audioMP3:
		changed, err = setMP3AlbumArtist(o, path)
	case audioM4A:
		changed, err = setM4AAlbumArtist(o, path)
	default:
		return
	}
	if err != nil {
		rep.fail(progress, "set-album-artist", path, err)
		return
	}
	if changed {
		rep.action(progress, "set-album-artist", path, o.AlbumArtist)
		rep.inc(&rep.AlbumArtistsSet)
	}
}

// ReadFirstGenre returns the genre tag of the first audio file found under dir
// (the first file classifyAudio recognises, even if its genre is empty), or ""
// if there is no audio. It is used to prefill the GUI's genre field. It never
// touches ffmpeg.
func ReadFirstGenre(dir string) string {
	var genre string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if classifyAudio(path) == audioOther {
			return nil
		}
		g, _ := readGenre(path)
		genre = g
		return errStopWalk
	})
	return genre
}
