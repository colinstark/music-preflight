package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/colinstark/coverfixer/internal/ffmpeg"
)

// transcodeFile converts a single audio file to the requested format in place,
// carrying over metadata and any (already-resized) embedded cover. The original
// is replaced; with o.Backup a <file>.bak copy is kept first. A returned error
// is engine-level (ffmpeg unavailable) and aborts the run; per-file conversion
// failures are recorded in the Report and return nil.
func transcodeFile(ctx context.Context, o Options, path string, rep *reportAccum, progress func(Event)) error {
	var (
		targetExt string
		audioArgs []string
	)
	switch o.Transcode {
	case TranscodeMP3_320:
		targetExt = ".mp3"
		audioArgs = []string{"-c:a", "libmp3lame", "-b:a", "320k", "-c:v", "copy"}
	case TranscodeAAC_256:
		targetExt = ".m4a"
		audioArgs = []string{"-c:a", "aac", "-b:a", "256k", "-c:v", "copy", "-disposition:v:0", "attached_pic"}
	default:
		return nil
	}

	outPath := strings.TrimSuffix(path, filepath.Ext(path)) + targetExt

	rep.action(progress, "transcode", path, "→ "+o.Transcode.String())
	if o.DryRun {
		rep.inc(&rep.Transcoded)
		return nil
	}

	bin, err := ffmpeg.Path()
	if err != nil {
		return fmt.Errorf("transcode requested but ffmpeg unavailable: %w", err)
	}

	tmp := outPath + ".coverfixer.tmp" + targetExt
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", path,
		"-map", "0:a:0", "-map", "0:v:0?", "-map_metadata", "0",
	}
	args = append(args, audioArgs...)
	args = append(args, tmp)

	cmd := exec.CommandContext(ctx, bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmp)
		rep.fail(progress, "transcode", path, fmt.Errorf("ffmpeg: %v: %s", err, strings.TrimSpace(string(out))))
		return nil
	}

	if err := maybeBackup(o, path); err != nil {
		os.Remove(tmp)
		rep.fail(progress, "transcode", path, err)
		return nil
	}

	// Install the converted file. When the extension changes the original has a
	// different name and must be removed for the output to take its place; move
	// it aside first rather than deleting outright, so a failed install can be
	// rolled back and there is never a window where the source is gone and the
	// output is not yet in place. Same-extension output is an atomic overwrite.
	if outPath != path {
		aside := path + ".coverfixer.orig"
		if err := os.Rename(path, aside); err != nil {
			os.Remove(tmp)
			rep.fail(progress, "transcode", path, err)
			return nil
		}
		if err := os.Rename(tmp, outPath); err != nil {
			os.Rename(aside, path) // restore the original
			os.Remove(tmp)
			rep.fail(progress, "transcode", path, err)
			return nil
		}
		os.Remove(aside)
	} else if err := os.Rename(tmp, outPath); err != nil {
		os.Remove(tmp)
		rep.fail(progress, "transcode", path, err)
		return nil
	}

	// ffmpeg copies the source cover stream verbatim (-c:v copy), so when the
	// standalone embedded-art pass wasn't requested the new file would otherwise
	// keep full-size art. Resize the carried-over cover in place so transcoded
	// output always ends up correctly sized. Backup is forced off here: outPath
	// is a freshly written file, not a user original.
	if err := resizeOutputArt(o, outPath); err != nil {
		rep.fail(progress, "transcode", outPath, err)
		return nil
	}

	rep.inc(&rep.Transcoded)
	return nil
}

// resizeOutputArt resizes the embedded cover in a transcode output in place,
// without writing a .bak sidecar.
func resizeOutputArt(o Options, path string) error {
	o.Backup = false
	var err error
	switch classifyAudio(path) {
	case audioMP3:
		_, err = resizeMP3Art(o, path)
	case audioM4A:
		_, err = resizeM4AArt(o, path)
	}
	return err
}
