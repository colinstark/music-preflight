package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bogem/id3v2/v2"
)

// writeMP3WithArt creates a file containing only an ID3v2 tag with one APIC
// picture. That is enough to exercise the artwork read/resize paths without a
// real audio stream.
func writeMP3WithArt(t *testing.T, path string, art []byte) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		t.Fatal(err)
	}
	tag.AddAttachedPicture(id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Picture:     art,
	})
	if err := tag.Save(); err != nil {
		t.Fatal(err)
	}
	tag.Close()
}

func TestResizeMP3Art(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.mp3")
	writeMP3WithArt(t, path, makeJPEG(t, 1500, 1500))

	o := DefaultOptions()
	o.Dir = dir
	changed, err := resizeMP3Art(o, path)
	if err != nil {
		t.Fatalf("resizeMP3Art: %v", err)
	}
	if !changed {
		t.Fatal("expected art to be resized")
	}

	art, err := readMP3Art(path)
	if err != nil {
		t.Fatalf("readMP3Art: %v", err)
	}
	w, h := jpegDimensions(t, art)
	if w > 500 || h > 500 {
		t.Errorf("embedded art %dx%d, want <= 500", w, h)
	}

	// Second pass should be a no-op now that art is within size.
	changed, err = resizeMP3Art(o, path)
	if err != nil {
		t.Fatalf("resizeMP3Art second pass: %v", err)
	}
	if changed {
		t.Error("expected no change on already-correct art")
	}
}

func TestResizeMP3ArtNoArt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.mp3")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := resizeMP3Art(DefaultOptions(), path)
	if err != nil {
		t.Fatalf("resizeMP3Art: %v", err)
	}
	if changed {
		t.Error("expected no change for file without art")
	}
}

func TestSetMP3Genre(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.mp3")
	writeMP3WithArt(t, path, makeJPEG(t, 200, 200))

	o := DefaultOptions()
	o.Genre = "Jazz"

	changed, err := setMP3Genre(o, path)
	if err != nil {
		t.Fatalf("setMP3Genre: %v", err)
	}
	if !changed {
		t.Fatal("expected genre to be set")
	}
	if g, err := readMP3Genre(path); err != nil {
		t.Fatalf("readMP3Genre: %v", err)
	} else if g != "Jazz" {
		t.Errorf("genre = %q, want Jazz", g)
	}

	// Second pass is a no-op (already that genre).
	changed, err = setMP3Genre(o, path)
	if err != nil {
		t.Fatalf("setMP3Genre second pass: %v", err)
	}
	if changed {
		t.Error("expected no change when genre already set")
	}
}

func TestSetMP3GenreDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.mp3")
	writeMP3WithArt(t, path, makeJPEG(t, 200, 200))

	o := DefaultOptions()
	o.Genre = "Rock"
	o.DryRun = true

	changed, err := setMP3Genre(o, path)
	if err != nil {
		t.Fatalf("setMP3Genre: %v", err)
	}
	if !changed {
		t.Fatal("dry-run should report intended change")
	}
	if g, _ := readMP3Genre(path); g == "Rock" {
		t.Error("dry-run must not write the tag")
	}
}
