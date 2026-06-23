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
		codec     string
		bitrate   string
		aac       bool
	)
	switch o.Transcode {
	case TranscodeMP3_320:
		codec, bitrate, targetExt = "libmp3lame", "320k", ".mp3"
	case TranscodeMP3_256:
		codec, bitrate, targetExt = "libmp3lame", "256k", ".mp3"
	case TranscodeMP3_192:
		codec, bitrate, targetExt = "libmp3lame", "192k", ".mp3"
	case TranscodeAAC_320:
		codec, bitrate, targetExt, aac = "aac", "320k", ".m4a", true
	case TranscodeAAC_256:
		codec, bitrate, targetExt, aac = "aac", "256k", ".m4a", true
	case TranscodeAAC_192:
		codec, bitrate, targetExt, aac = "aac", "192k", ".m4a", true
	default:
		return nil
	}
	audioArgs := []string{"-c:a", codec, "-b:a", bitrate, "-c:v", "copy"}
	if aac {
		audioArgs = append(audioArgs, "-disposition:v:0", "attached_pic")
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
	// output always ends up correctly sized.
	if err := resizeOutputArt(o, outPath); err != nil {
		rep.fail(progress, "transcode", outPath, err)
		return nil
	}

	rep.inc(&rep.Transcoded)
	return nil
}

// resizeOutputArt resizes the embedded cover in a transcode output in place.
func resizeOutputArt(o Options, path string) error {
	var err error
	switch classifyAudio(path) {
	case audioMP3:
		_, err = resizeMP3Art(o, path)
	case audioM4A:
		_, err = resizeM4AArt(o, path)
	}
	return err
}
