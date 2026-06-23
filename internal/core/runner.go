package core

import "context"

// Run processes o.Dir according to the enabled passes and returns a Report.
// progress, if non-nil, receives a live Event for each unit of work; the CLI
// prints these and the GUI renders them. Per-file failures are recorded in the
// Report and do not abort the run; only engine-level failures (a missing scan
// root, or ffmpeg being unavailable for a requested transcode) return an error.
func Run(ctx context.Context, o Options, progress func(Event)) (Report, error) {
	if progress == nil {
		progress = func(Event) {}
	}
	o.applyDefaults()

	var rep Report
	folders, err := scan(o.Dir, o.Recursive, &rep, progress)
	if err != nil {
		return rep, err
	}

	for _, f := range folders {
		if err := ctx.Err(); err != nil {
			return rep, err
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
			for _, a := range f.audio {
				resizeEmbedded(o, a, &rep, progress)
			}
		}

		// Pass 4: set the genre tag. Runs before transcode so the tag is
		// carried onto the output by ffmpeg's -map_metadata 0.
		if o.SetGenre && o.Genre != "" && len(f.audio) > 0 {
			for _, a := range f.audio {
				setGenre(o, a, &rep, progress)
			}
		}

		// Pass 5: transcode audio. Runs after the embedded-art and genre
		// passes so the (already-resized) cover and genre carry into the new
		// file via -map_metadata 0.
		if o.Transcode != TranscodeNone {
			for _, a := range f.audio {
				if err := transcodeFile(ctx, o, a, &rep, progress); err != nil {
					return rep, err
				}
			}
		}
	}
	return rep, nil
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

func resizeEmbedded(o Options, path string, rep *Report, progress func(Event)) {
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
		rep.EmbeddedResized++
	}
}
