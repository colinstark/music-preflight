package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunArtworkPasses(t *testing.T) {
	root := t.TempDir()

	// Album A: oversized embedded art + a stray (non-cover) jpg.
	albumA := filepath.Join(root, "Artist", "Album A")
	if err := os.MkdirAll(albumA, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMP3WithArt(t, filepath.Join(albumA, "01.mp3"), makeJPEG(t, 1500, 1500))
	if err := os.WriteFile(filepath.Join(albumA, "folder.jpg"), makeJPEG(t, 1200, 1200), 0o644); err != nil {
		t.Fatal(err)
	}

	// Album B: oversized embedded art, no jpg on disk.
	albumB := filepath.Join(root, "Artist", "Album B")
	if err := os.MkdirAll(albumB, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMP3WithArt(t, filepath.Join(albumB, "01.mp3"), makeJPEG(t, 900, 900))

	o := DefaultOptions()
	o.Dir = root
	o.ResizeEmbedded = true

	rep, err := Run(context.Background(), o, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Failed != 0 {
		t.Fatalf("Run reported %d failures", rep.Failed)
	}

	// Album A: stray jpg renamed to cover.jpg and resized.
	coverA := filepath.Join(albumA, "cover.jpg")
	if !fileExists(coverA) {
		t.Error("Album A cover.jpg not created from stray jpg")
	} else if w, h := jpegDimensions(t, readFile(t, coverA)); w > 500 || h > 500 {
		t.Errorf("Album A cover %dx%d, want <= 500", w, h)
	}
	if fileExists(filepath.Join(albumA, "folder.jpg")) {
		t.Error("Album A folder.jpg should have been renamed away")
	}

	// Album B: cover.jpg extracted from embedded art.
	coverB := filepath.Join(albumB, "cover.jpg")
	if !fileExists(coverB) {
		t.Error("Album B cover.jpg not extracted from embedded art")
	} else if w, h := jpegDimensions(t, readFile(t, coverB)); w > 500 || h > 500 {
		t.Errorf("Album B cover %dx%d, want <= 500", w, h)
	}

	// Embedded art in both tracks resized to within bounds.
	if rep.EmbeddedResized < 1 {
		t.Errorf("expected embedded art to be resized, got %d", rep.EmbeddedResized)
	}
	artA, _ := readMP3Art(filepath.Join(albumA, "01.mp3"))
	if w, h := jpegDimensions(t, artA); w > 500 || h > 500 {
		t.Errorf("Album A embedded art %dx%d, want <= 500", w, h)
	}
}

func TestRunDryRunMakesNoChanges(t *testing.T) {
	root := t.TempDir()
	album := filepath.Join(root, "Album")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMP3WithArt(t, filepath.Join(album, "01.mp3"), makeJPEG(t, 1500, 1500))

	o := DefaultOptions()
	o.Dir = root
	o.ResizeEmbedded = true
	o.DryRun = true

	rep, err := Run(context.Background(), o, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fileExists(filepath.Join(album, "cover.jpg")) {
		t.Error("dry-run created cover.jpg")
	}
	// Embedded art must remain oversized (untouched).
	art, _ := readMP3Art(filepath.Join(album, "01.mp3"))
	if w, _ := jpegDimensions(t, art); w <= 500 {
		t.Error("dry-run modified embedded art")
	}
	if rep.Extracted == 0 && rep.EmbeddedResized == 0 {
		t.Error("dry-run should still report intended actions")
	}
}

func TestRunEmitsFolderHeaders(t *testing.T) {
	root := t.TempDir()

	// Working folder: has a jpg the artwork passes will touch.
	work := filepath.Join(root, "Has Work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "cover.jpg"), makeJPEG(t, 1200, 1200), 0o644); err != nil {
		t.Fatal(err)
	}

	// Idle folder: a lone audio file with no art and only artwork passes enabled,
	// so no enabled pass acts on it.
	idle := filepath.Join(root, "Idle")
	if err := os.MkdirAll(idle, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMP3WithArt(t, filepath.Join(idle, "01.mp3"), nil)

	o := DefaultOptions()
	o.Dir = root
	o.ExtractCover = false // the idle folder's only candidate pass — keep it idle
	o.ResizeEmbedded = false

	var headers []string
	_, err := Run(context.Background(), o, func(e Event) {
		if e.Kind == EventInfo {
			headers = append(headers, e.Path)
		}
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(headers) != 1 || headers[0] != work {
		t.Errorf("folder headers = %v, want exactly [%q]", headers, work)
	}
}

// TestRenameThenExtractDoesNotOverwrite guards against a regression where a
// stray jpg renamed to cover.jpg in Pass 1 was overwritten by Pass 2's extract
// pass, because processJPGs failed to propagate the rename to f.hasCover.
//
// The stray jpg and the embedded art have distinguishable aspect ratios
// (landscape vs portrait), so the source of the resulting cover.jpg is
// unambiguous: it must be the renamed jpg, and Extracted must be 0.
func TestRenameThenExtractDoesNotOverwrite(t *testing.T) {
	root := t.TempDir()
	album := filepath.Join(root, "Album")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}

	// Stray jpg on disk: landscape 1200x600 → resizes to 500x250.
	if err := os.WriteFile(filepath.Join(album, "front.jpg"), makeJPEG(t, 1200, 600), 0o644); err != nil {
		t.Fatal(err)
	}
	// Embedded art: portrait 600x1200 → would resize to 250x500 if extracted.
	writeMP3WithArt(t, filepath.Join(album, "01.mp3"), makeJPEG(t, 600, 1200))

	o := DefaultOptions() // RenameStrayJPG + ResizeCoverJPG + ExtractCover all on
	o.Dir = root

	rep, err := Run(context.Background(), o, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	cover := filepath.Join(album, "cover.jpg")
	if !fileExists(cover) {
		t.Fatal("cover.jpg not created from renamed stray jpg")
	}
	w, h := jpegDimensions(t, readFile(t, cover))
	if w != 500 || h != 250 {
		t.Errorf("cover.jpg %dx%d, want 500x250 (from renamed front.jpg); "+
			"a portrait 250x500 would mean the extract pass overwrote it", w, h)
	}
	if rep.Extracted != 0 {
		t.Errorf("Extracted = %d, want 0 (folder is covered after rename)", rep.Extracted)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestScanSkipsUnreadableSubdir verifies that one locked subdirectory does not
// abort the whole run: the unreadable dir is recorded as a failure and the
// remaining readable folders are still processed.
func TestScanSkipsUnreadableSubdir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; chmod would be ignored")
	}
	root := t.TempDir()

	// Readable album whose cover will be processed.
	album := filepath.Join(root, "Album")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(album, "cover.jpg"), makeJPEG(t, 900, 900), 0o644); err != nil {
		t.Fatal(err)
	}

	// Locked subdir: its contents must not abort the run.
	locked := filepath.Join(root, "Locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "cover.jpg"), makeJPEG(t, 900, 900), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o755) // restore so t.TempDir cleanup can remove it

	o := DefaultOptions()
	o.Dir = root

	var sawErr bool
	rep, err := Run(context.Background(), o, func(e Event) {
		if e.Kind == EventError {
			sawErr = true
		}
	})
	if err != nil {
		t.Fatalf("an unreadable subdir aborted the entire run: %v", err)
	}
	if !sawErr || rep.Failed == 0 {
		t.Error("expected the unreadable subdir to be recorded as a failure")
	}
	if rep.CoversResized < 1 {
		t.Error("expected the readable album to still be processed")
	}
}
