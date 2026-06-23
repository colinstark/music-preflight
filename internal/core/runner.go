package core

import "context"

// Run processes o.Dir according to the enabled passes and returns a Report.
// progress, if non-nil, receives a live Event for each unit of work; the CLI
// prints these and the future GUI will render them. Per-file failures are
// recorded in the Report and do not abort the run; only engine-level failures
// (a missing scan root, or ffmpeg being unavailable for a requested transcode)
// return an error.
func Run(ctx context.Context, o Options, progress func(Event)) (Report, error) {
	if progress == nil {
		progress = func(Event) {}
	}
	o.applyDefaults()

	var rep reportAccum
	folders, err := scan(o.Dir, o.Recursive, &rep, progress)
	if err != nil {
		return rep.Report, err
	}

	for _, f := range folders {
		if err := ctx.Err(); err != nil {
			return rep.Report, err
		}

		// Announce the folder once, before its passes run, so front-ends can
		// group the per-file events that follow under a header. Skipped for
		// folders no enabled pass will touch, to avoid empty headers.
		if folderHasWork(o, f) {
			rep.info(progress, "scan", f.dir, "")
		}

		// Pass 1: rename stray jpgs + resize cover.jpg.
		if o.RenameStrayJPG || o.ResizeCoverJPG {
			processJPGs(o, f, &rep, progress)
		}

		// Pass 2: write cover.jpg from embedded art when the folder lacks one.
		if o.ExtractCover && !f.hasCover && len(f.audio) > 0 {
			extractCover(o, f, &rep, progress)
		}

		// Pass 3: resize artwork embedded in audio files, in place.
		if o.ResizeEmbedded {
			if err := forEachParallel(ctx, f.audio, func(_ context.Context, a string) error {
				resizeEmbedded(o, a, &rep, progress)
				return nil
			}); err != nil {
				return rep.Report, err
			}
		}

		// Pass 4: set the genre tag. Runs before transcode so the tag is
		// carried onto the output by ffmpeg's -map_metadata 0.
		if o.SetGenre && o.Genre != "" && len(f.audio) > 0 {
			if err := forEachParallel(ctx, f.audio, func(_ context.Context, a string) error {
				setGenre(o, a, &rep, progress)
				return nil
			}); err != nil {
				return rep.Report, err
			}
		}

		// Pass 5: transcode audio. Runs after the embedded-art and genre
		// passes so the (already-resized) cover and genre carry into the new
		// file via -map_metadata 0. Each transcode is an independent ffmpeg
		// subprocess, so this pass benefits most from the worker pool.
		if o.Transcode != TranscodeNone {
			if err := forEachParallel(ctx, f.audio, func(ctx context.Context, a string) error {
				return transcodeFile(ctx, o, a, &rep, progress)
			}); err != nil {
				return rep.Report, err
			}
		}
	}
	return rep.Report, nil
}

// folderHasWork reports whether any enabled pass will act on f, mirroring the
// pass gates below. Used to suppress folder-header events for folders that will
// produce no further output.
func folderHasWork(o Options, f *albumFolder) bool {
	switch {
	case (o.RenameStrayJPG || o.ResizeCoverJPG) && len(f.jpgs) > 0:
		return true
	case o.ExtractCover && !f.hasCover && len(f.audio) > 0:
		return true
	case o.ResizeEmbedded && len(f.audio) > 0:
		return true
	case o.SetGenre && o.Genre != "" && len(f.audio) > 0:
		return true
	case o.Transcode != TranscodeNone && len(f.audio) > 0:
		return true
	default:
		return false
	}
}

func resizeEmbedded(o Options, path string, rep *reportAccum, progress func(Event)) {
	var (
		changed bool
		err     error
	)
	switch classifyAudio(path) {
	case audioMP3:
		changed, err = resizeMP3Art(o, path)
	case audioM4A:
		changed, err = resizeM4AArt(o, path)
	default:
		return
	}
	if err != nil {
		rep.fail(progress, "resize-embedded", path, err)
		return
	}
	if changed {
		rep.action(progress, "resize-embedded", path, "")
		rep.inc(&rep.EmbeddedResized)
	}
}
