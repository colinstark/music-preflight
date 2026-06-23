package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// makeTaggedTrack writes a short silent m4a with the given metadata (and an
// optional embedded cover) via ffmpeg. It needs a real audio stream so duration
// is non-zero, which the pure-Go helpers cannot synthesise. disc sets the disc
// number tag (pass "" to leave it unset).
func makeTaggedTrack(t *testing.T, bin, dir, name, album, artist, genre, title, track, disc string, withCover bool) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono:d=1",
	}
	if withCover {
		cover := filepath.Join(dir, "cov-"+name+".jpg")
		if err := os.WriteFile(cover, makeJPEG(t, 400, 400), 0o644); err != nil {
			t.Fatal(err)
		}
		args = append(args, "-i", cover)
	}
	args = append(args,
		"-map", "0:a",
		"-c:a", "aac",
		"-metadata", "title="+title,
		"-metadata", "album="+album,
		"-metadata", "artist="+artist,
		"-metadata", "album_artist="+artist,
		"-metadata", "genre="+genre,
		"-metadata", "track="+track,
	)
	if disc != "" {
		args = append(args, "-metadata", "disc="+disc)
	}
	if withCover {
		args = append(args, "-map", "1", "-c:v", "copy", "-disposition:v:0", "attached_pic")
	}
	out := filepath.Join(dir, name)
	args = append(args, out)
	if b, err := exec.CommandContext(ctx, bin, args...).CombinedOutput(); err != nil {
		t.Fatalf("build tagged m4a %q: %v: %s", name, err, b)
	}
	return out
}

func TestReadLibraryGroupsByAlbum(t *testing.T) {
	bin := ffmpegBin(t) // self-skips when ffmpeg is absent
	dir := t.TempDir()

	makeTaggedTrack(t, bin, dir, "a1.m4a", "Album One", "Alpha", "Jazz", "First", "1", "", true)
	makeTaggedTrack(t, bin, dir, "a2.m4a", "Album One", "Alpha", "Jazz", "Second", "2", "", false)
	makeTaggedTrack(t, bin, dir, "b1.m4a", "Album Two", "Beta", "Rock", "Third", "1", "", true)

	albums, err := ReadLibrary(dir, true)
	if err != nil {
		t.Fatalf("ReadLibrary: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("got %d albums, want 2: %+v", len(albums), albums)
	}

	// Sorted by artist then title: Alpha/Album One, Beta/Album Two.
	a, b := albums[0], albums[1]
	if a.Artist != "Alpha" || a.Title != "Album One" {
		t.Errorf("albums[0] = %q / %q, want Alpha / Album One", a.Artist, a.Title)
	}
	if b.Artist != "Beta" || b.Title != "Album Two" {
		t.Errorf("albums[1] = %q / %q, want Beta / Album Two", b.Artist, b.Title)
	}
	if a.Genre != "Jazz" {
		t.Errorf("album one genre = %q, want Jazz", a.Genre)
	}
	if len(a.Tracks) != 2 {
		t.Fatalf("album one tracks = %d, want 2", len(a.Tracks))
	}
	// Tracks ordered by track number (1 then 2).
	if a.Tracks[0].Number != 1 || a.Tracks[1].Number != 2 {
		t.Errorf("album one track order = %d,%d, want 1,2", a.Tracks[0].Number, a.Tracks[1].Number)
	}
	for _, tr := range a.Tracks {
		if tr.Duration <= 0 {
			t.Errorf("track %q duration = %v, want > 0", tr.Title, tr.Duration)
		}
	}
	// Each album took its art from its first cover-bearing track.
	if a.Artwork == "" || b.Artwork == "" {
		t.Errorf("expected both albums to have artwork; got %q / %q", a.Artwork, b.Artwork)
	}
}

func TestReadLibraryRespectsRecursive(t *testing.T) {
	bin := ffmpegBin(t)
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	makeTaggedTrack(t, bin, root, "top.m4a", "Top", "T", "G", "T1", "1", "", false)
	makeTaggedTrack(t, bin, sub, "deep.m4a", "Deep", "D", "G", "D1", "1", "", false)

	if albums, err := ReadLibrary(root, false); err != nil {
		t.Fatalf("ReadLibrary non-recursive: %v", err)
	} else if len(albums) != 1 {
		t.Errorf("non-recursive got %d albums, want 1 (top only)", len(albums))
	}
	if albums, err := ReadLibrary(root, true); err != nil {
		t.Fatalf("ReadLibrary recursive: %v", err)
	} else if len(albums) != 2 {
		t.Errorf("recursive got %d albums, want 2", len(albums))
	}
}

func TestReadLibrarySortsByDisc(t *testing.T) {
	bin := ffmpegBin(t)
	dir := t.TempDir()

	makeTaggedTrack(t, bin, dir, "d1t1.m4a", "Double", "Artist", "G", "A1", "1", "1", false)
	makeTaggedTrack(t, bin, dir, "d1t2.m4a", "Double", "Artist", "G", "A2", "2", "1", false)
	makeTaggedTrack(t, bin, dir, "d2t1.m4a", "Double", "Artist", "G", "B1", "1", "2", false)
	makeTaggedTrack(t, bin, dir, "d2t2.m4a", "Double", "Artist", "G", "B2", "2", "2", false)

	albums, err := ReadLibrary(dir, true)
	if err != nil {
		t.Fatalf("ReadLibrary: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("got %d albums, want 1 (one multi-disc album)", len(albums))
	}
	tr := albums[0].Tracks
	if len(tr) != 4 {
		t.Fatalf("got %d tracks, want 4", len(tr))
	}
	// Tracks are sorted disc-major: disc 1 tracks first, then disc 2.
	want := []int{1, 1, 2, 2}
	for i, w := range want {
		if got := discKey(tr[i].Disc); got != w {
			t.Errorf("track %d disc key = %d (raw %d), want %d", i, got, tr[i].Disc, w)
		}
	}
	// Within disc 2, track numbers ascend.
	if tr[2].Number != 1 || tr[3].Number != 2 {
		t.Errorf("disc 2 order = %d,%d, want 1,2", tr[2].Number, tr[3].Number)
	}
}
