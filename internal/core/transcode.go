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
func transcodeFile(ctx context.Context, o Options, path string, rep *Report, progress func(Event)) error {
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
		rep.Transcoded++
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
	// When the extension changes, remove the original so the converted file
	// takes its place (the .bak copy, if any, is the safety net).
	if outPath != path {
		if err := os.Remove(path); err != nil {
			os.Remove(tmp)
			rep.fail(progress, "transcode", path, err)
			return nil
		}
	}
	if err := os.Rename(tmp, outPath); err != nil {
		os.Remove(tmp)
		rep.fail(progress, "transcode", path, err)
		return nil
	}
	rep.Transcoded++
	return nil
}
