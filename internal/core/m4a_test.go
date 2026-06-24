package core

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	mp4tag "github.com/Sorrow446/go-mp4tag"
)

// m4aFixtureB64 is a tiny but valid M4A (1s silence + a 600x600 JPEG cover)
// built once with ffmpeg and embedded here so the covr read/resize/write path
// can be exercised headlessly — no ffmpeg at test time, no committed binary
// asset. It targets the riskiest engine path (pure-Go covr rewrite via
// go-mp4tag, including the stco offset patching on Write), which the
// ffmpeg-gated TestResizeM4AArt does not cover in CI.
const m4aFixtureB64 = "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AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAH/2QAAAAhmcmVlAAAAzW1kYXTeAgBMYXZjNjIuMjguMTAyAAIwQA4BGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBwEYIAcBGCAHARggBw=="

// writeM4AFixture decodes the embedded M4A fixture to path.
func writeM4AFixture(t *testing.T, path string) {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(m4aFixtureB64)
	if err != nil {
		t.Fatalf("decode m4a fixture: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestResizeM4AArtHeadless exercises the pure-Go covr read/resize/rewrite path
// (go-mp4tag) without requiring ffmpeg, including the stco offset patching on
// Write. It is the headless counterpart of the ffmpeg-gated TestResizeM4AArt.
func TestResizeM4AArtHeadless(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.m4a")
	writeM4AFixture(t, path)

	// Fixture integrity: the embedded cover is oversized (600x600).
	art, err := readM4AArt(path)
	if err != nil {
		t.Fatalf("readM4AArt: %v", err)
	}
	if art == nil {
		t.Fatal("fixture has no embedded cover")
	}
	if w, h := jpegDimensions(t, art); w != 600 || h != 600 {
		t.Fatalf("fixture cover %dx%d, want 600x600", w, h)
	}

	o := DefaultOptions()
	o.Dir = dir

	changed, err := resizeM4AArt(o, path)
	if err != nil {
		t.Fatalf("resizeM4AArt: %v", err)
	}
	if !changed {
		t.Fatal("expected m4a art to be resized")
	}

	art2, err := readM4AArt(path)
	if err != nil {
		t.Fatalf("readM4AArt after resize: %v", err)
	}
	w, h := jpegDimensions(t, art2)
	if w > 500 || h > 500 {
		t.Errorf("resized m4a art %dx%d, want <= 500", w, h)
	}
	// The carried cover must be baseline JPEG, not progressive.
	if isProgressiveJPEG(art2) {
		t.Error("resized m4a cover is progressive; want baseline")
	}

	// Second pass is a no-op now that the cover is within size and baseline.
	changed, err = resizeM4AArt(o, path)
	if err != nil {
		t.Fatalf("resizeM4AArt second pass: %v", err)
	}
	if changed {
		t.Error("expected no change on already-correct m4a art")
	}
}

// TestReadM4AArtHeadless sanity-checks the pure-Go covr reader on the fixture.
func TestReadM4AArtHeadless(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.m4a")
	writeM4AFixture(t, path)

	art, err := readM4AArt(path)
	if err != nil {
		t.Fatalf("readM4AArt: %v", err)
	}
	if art == nil {
		t.Fatal("expected embedded cover")
	}
	if w, h := jpegDimensions(t, art); w != 600 || h != 600 {
		t.Errorf("cover %dx%d, want 600x600", w, h)
	}
}

// TestSetM4AGenreHeadless exercises the pure-Go ©gen write path headlessly.
func TestSetM4AGenreHeadless(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.m4a")
	writeM4AFixture(t, path)

	if g, _ := readM4AGenre(path); g != "" {
		t.Fatalf("fixture genre = %q, want empty", g)
	}

	o := DefaultOptions()
	o.Genre = "Ambient"

	changed, err := setM4AGenre(o, path)
	if err != nil {
		t.Fatalf("setM4AGenre: %v", err)
	}
	if !changed {
		t.Fatal("expected genre to be set")
	}
	if g, _ := readM4AGenre(path); g != "Ambient" {
		t.Errorf("genre = %q, want Ambient", g)
	}

	// The cover must still be intact after the ©gen rewrite.
	art, err := readM4AArt(path)
	if err != nil {
		t.Fatalf("readM4AArt after genre set: %v", err)
	}
	if art == nil {
		t.Fatal("cover lost after genre write")
	}

	// Second pass is a no-op (already that genre).
	changed, err = setM4AGenre(o, path)
	if err != nil {
		t.Fatalf("setM4AGenre second pass: %v", err)
	}
	if changed {
		t.Error("expected no change when genre already set")
	}
}

// readM4ATagsHeadless reads a file's writable text tags for test verification.
func readM4ATagsHeadless(t *testing.T, path string) textTags {
	t.Helper()
	mp4, err := mp4tag.Open(path)
	if err != nil {
		t.Fatalf("open mp4: %v", err)
	}
	defer mp4.Close()
	cur, err := mp4.Read()
	if err != nil {
		t.Fatalf("read mp4 tags: %v", err)
	}
	return readM4ATextTags(cur)
}

func TestSetM4ATagsHeadless(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.m4a")
	writeM4AFixture(t, path)

	want := textTags{
		Title:       "Svefn-g-englar",
		Artist:      "Sigur Rós",
		AlbumArtist: "Sigur Rós",
		Album:       "Ágætis byrjun",
		Genre:       "Post-Rock",
		Year:        "1999",
		TrackNumber: 1,
	}
	o := DefaultOptions()
	changed, err := setM4ATags(o, path, want)
	if err != nil {
		t.Fatalf("setM4ATags: %v", err)
	}
	if !changed {
		t.Fatal("expected tags to be written")
	}
	if got := readM4ATagsHeadless(t, path); got != want {
		t.Errorf("tags = %+v, want %+v", got, want)
	}

	// The cover must survive the text-tag rewrite.
	if art, err := readM4AArt(path); err != nil || art == nil {
		t.Errorf("cover lost after tag write (art=%v, err=%v)", art != nil, err)
	}

	// A second pass with the same values is a no-op.
	changed, err = setM4ATags(o, path, want)
	if err != nil {
		t.Fatalf("setM4ATags second pass: %v", err)
	}
	if changed {
		t.Error("expected no change when tags already match")
	}
}

func TestSetM4ATagsClearsHeadless(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.m4a")
	writeM4AFixture(t, path)

	o := DefaultOptions()
	if _, err := setM4ATags(o, path, textTags{Title: "X", Artist: "Y", Album: "Z", Year: "2020", TrackNumber: 2}); err != nil {
		t.Fatal(err)
	}
	clear := textTags{}
	if _, err := setM4ATags(o, path, clear); err != nil {
		t.Fatal(err)
	}
	if got := readM4ATagsHeadless(t, path); got != clear {
		t.Errorf("after clear, tags = %+v, want all zero", got)
	}
}

func TestSetM4ATagsDryRunHeadless(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.m4a")
	writeM4AFixture(t, path)

	o := DefaultOptions()
	o.DryRun = true
	changed, err := setM4ATags(o, path, textTags{Title: "Dry", TrackNumber: 3})
	if err != nil {
		t.Fatalf("setM4ATags: %v", err)
	}
	if !changed {
		t.Fatal("dry-run should report intended change")
	}
	if got := readM4ATagsHeadless(t, path); got.Title == "Dry" {
		t.Error("dry-run must not write tags")
	}
}
