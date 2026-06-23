package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// ffmpegBin returns a system ffmpeg path or skips the test when none is present.
// These integration tests need real audio streams, which our pure-Go helpers
// cannot synthesise.
func ffmpegBin(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed; skipping integration test")
	}
	return p
}

// makeM4A creates a 1s silent AAC/M4A file with an embedded cover of the given size.
func makeM4A(t *testing.T, bin, dir string, coverPx int) string {
	t.Helper()
	cover := filepath.Join(dir, "src-cover.jpg")
	if err := os.WriteFile(cover, makeJPEG(t, coverPx, coverPx), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "track.m4a")
	// anullsrc is otherwise infinite; bounding it with d=1 (rather than a
	// trailing -t) guarantees ffmpeg reaches EOF and exits instead of hanging.
	// The context timeout is a belt-and-braces guard so a misbuilt command can
	// never stall the whole suite.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono:d=1",
		"-i", cover,
		"-map", "0:a", "-map", "1", "-c:a", "aac", "-c:v", "copy",
		"-disposition:v:0", "attached_pic", out)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build test m4a: %v: %s", err, outBytes)
	}
	return out
}

func TestResizeM4AArt(t *testing.T) {
	bin := ffmpegBin(t)
	dir := t.TempDir()
	path := makeM4A(t, bin, dir, 1400)

	o := DefaultOptions()
	o.Dir = dir
	changed, err := resizeM4AArt(o, path)
	if err != nil {
		t.Fatalf("resizeM4AArt: %v", err)
	}
	if !changed {
		t.Fatal("expected m4a art to be resized")
	}
	art, err := readM4AArt(path)
	if err != nil {
		t.Fatalf("readM4AArt: %v", err)
	}
	if w, h := jpegDimensions(t, art); w > 500 || h > 500 {
		t.Errorf("m4a art %dx%d, want <= 500", w, h)
	}
}

func TestTranscodeToMP3(t *testing.T) {
	bin := ffmpegBin(t)
	dir := t.TempDir()
	src := makeM4A(t, bin, dir, 800)

	o := DefaultOptions()
	o.Dir = dir
	o.RenameStrayJPG = false
	o.ResizeCoverJPG = false
	o.ExtractCover = false
	o.Transcode = TranscodeMP3_320
	o.Backup = true

	rep, err := Run(context.Background(), o, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Failed != 0 {
		t.Fatalf("%d transcode failures", rep.Failed)
	}
	if rep.Transcoded != 1 {
		t.Fatalf("Transcoded = %d, want 1", rep.Transcoded)
	}
	mp3 := filepath.Join(dir, "track.mp3")
	if !fileExists(mp3) {
		t.Error("expected track.mp3 output")
	}
	if fileExists(src) {
		t.Error("original .m4a should have been replaced")
	}
	// Backup is now a full-folder copy under <parent>/backup/<rootname>, not a
	// .bak sidecar, and it is taken before the run so it holds the original.
	backupDir := filepath.Join(filepath.Dir(dir), "backup", filepath.Base(dir))
	if !fileExists(filepath.Join(backupDir, filepath.Base(src))) {
		t.Error("expected original .m4a duplicated into the backup folder")
	}
	if fileExists(src + ".bak") {
		t.Error("backup must not create .bak sidecars")
	}

	// The 800px source cover is copied into the mp3 by ffmpeg; with ResizeEmbedded
	// off the transcode pass must still size it down to ArtSize on the output.
	art, err := readMP3Art(mp3)
	if err != nil {
		t.Fatalf("readMP3Art: %v", err)
	}
	if art == nil {
		t.Fatal("expected embedded cover carried into mp3")
	}
	if w, h := jpegDimensions(t, art); w > 500 || h > 500 {
		t.Errorf("transcoded mp3 art %dx%d, want <= 500", w, h)
	}
}
