package core

import (
	"encoding/base64"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/colinstark/coverfixer/internal/ffmpeg"
)

// Album is a display-only grouping of tracks for the GUI's library preview. It
// is read-only and never participates in an engine run. Artwork is a base64
// "data:image/jpeg;..." URL (empty when the album has no artwork).
type Album struct {
	Title   string  `json:"title"`
	Artist  string  `json:"artist"`
	Genre   string  `json:"genre"`
	Year    string  `json:"year"`
	Artwork string  `json:"artwork"`
	Tracks  []Track `json:"tracks"`
}

// Track is one audio file's display metadata for the preview.
type Track struct {
	Number   int     `json:"number"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"` // seconds
	Disc     int     `json:"disc"`     // 0 when unset
}

// thumbSize is the largest dimension of a preview artwork thumbnail, in pixels.
const thumbSize = 96

// ReadLibrary scans dir for audio files (respecting recursive), reads each
// file's metadata via ffmpeg, and groups the results by album for the GUI's idle
// preview. It never mutates files. An unreadable individual file is skipped; an
// unreadable root, or a missing ffmpeg, is returned as an error.
func ReadLibrary(dir string, recursive bool) ([]Album, error) {
	if _, err := ffmpeg.Path(); err != nil {
		return nil, err
	}

	var paths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == dir {
				return err
			}
			return nil
		}
		if d.IsDir() {
			if !recursive && path != dir {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), "._") {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".mp3", ".m4a", ".aac":
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	probed := make([]ffmpeg.ProbeResult, len(paths))
	{
		sem := make(chan struct{}, workerCount())
		var wg sync.WaitGroup
		for i, p := range paths {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, p string) {
				defer wg.Done()
				defer func() { <-sem }()
				if r, err := ffmpeg.Probe(p); err == nil {
					probed[i] = r
				}
			}(i, p)
		}
		wg.Wait()
	}

	// Group by (album, album artist), preserving the first non-empty tag
	// values and a representative art-bearing path per album.
	type group struct {
		album   *Album
		artPath string
	}
	groups := map[string]*group{}
	var order []string
	for i, p := range paths {
		r := probed[i]
		if r.Tags == nil && r.Duration == 0 {
			continue // ffmpeg could not read this file; skip it.
		}
		tags := r.Tags
		title := firstNonEmpty(tag(tags, "album"), "Unknown Album")
		artist := firstNonEmpty(tag(tags, "album_artist", "album artist", "albumartist", "band", "artist"), "Unknown Artist")
		key := title + "\x00" + artist
		g, ok := groups[key]
		if !ok {
			g = &group{album: &Album{Title: title, Artist: artist}}
			groups[key] = g
			order = append(order, key)
		}
		if g.album.Genre == "" {
			g.album.Genre = tag(tags, "genre")
		}
		if g.album.Year == "" {
			g.album.Year = yearOf(tag(tags, "date", "year"))
		}
		if g.artPath == "" && r.HasArt {
			g.artPath = p
		}
		trackTitle := firstNonEmpty(tag(tags, "title"), strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)))
		g.album.Tracks = append(g.album.Tracks, Track{
			Number:   parseLeadInt(tag(tags, "track")),
			Title:    trackTitle,
			Duration: r.Duration,
			Disc:     parseLeadInt(tag(tags, "disc", "part_of_a_set", "tpos")),
		})
	}

	// Attach one thumbnail per album (first art-bearing track).
	for _, key := range order {
		g := groups[key]
		if g.artPath == "" {
			continue
		}
		if thumb, _ := ffmpeg.ExtractThumb(g.artPath, thumbSize); len(thumb) > 0 {
			g.album.Artwork = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumb)
		}
	}

	out := make([]Album, 0, len(order))
	for _, key := range order {
		a := groups[key].album
		sort.SliceStable(a.Tracks, func(i, j int) bool {
			ti, tj := a.Tracks[i], a.Tracks[j]
			if di, dj := discKey(ti.Disc), discKey(tj.Disc); di != dj {
				return di < dj
			}
			if ti.Number != tj.Number {
				if ti.Number == 0 {
					return false
				}
				if tj.Number == 0 {
					return true
				}
				return ti.Number < tj.Number
			}
			return ti.Title < tj.Title
		})
		out = append(out, *a)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Artist != out[j].Artist {
			return out[i].Artist < out[j].Artist
		}
		return out[i].Title < out[j].Title
	})
	return out, nil
}

func workerCount() int {
	if n := runtime.GOMAXPROCS(0); n > 0 {
		return n
	}
	return 4
}

// tag returns the first non-empty value among the given lowercased tag keys.
func tag(tags map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(tags[k]); v != "" {
			return v
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return v
		}
	}
	return ""
}

// parseLeadInt parses a "N" or "N/M" tag (track or disc) into its leading int.
func parseLeadInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if i := strings.IndexAny(s, "/"); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// discKey folds an unset disc (0 or negative) to 1, so untagged tracks land in
// the same group as disc 1 rather than forming a spurious separate disc.
func discKey(d int) int {
	if d <= 0 {
		return 1
	}
	return d
}

// yearOf extracts a 4-digit year prefix from a date tag ("2020" or
// "2020-01-01"); empty when there is no parseable year.
func yearOf(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return ""
	}
	for i := 0; i < 4; i++ {
		if s[i] < '0' || s[i] > '9' {
			return ""
		}
	}
	return s[:4]
}

// FirstMetadata is the prefilled metadata read from the first audio file under a
// directory, used to seed the GUI's metadata fields on folder pick.
type FirstMetadata struct {
	Genre       string `json:"genre"`
	AlbumArtist string `json:"albumArtist"`
}

// ReadFirstMetadata probes the first audio file under dir (via ffmpeg) and
// returns its genre and album artist for prefilling the GUI. Empty values (and
// a missing ffmpeg) yield an empty FirstMetadata; the directory is never mutated.
func ReadFirstMetadata(dir string) FirstMetadata {
	var fm FirstMetadata
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if classifyAudio(path) == audioOther {
			return nil
		}
		r, perr := ffmpeg.Probe(path)
		if perr != nil {
			return nil // unreadable: try the next file
		}
		fm.Genre = tag(r.Tags, "genre")
		fm.AlbumArtist = tag(r.Tags, "album_artist", "album artist", "albumartist", "band", "artist")
		return errStopWalk
	})
	return fm
}
