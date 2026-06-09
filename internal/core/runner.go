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

	var rep Report
	folders, err := scan(o.Dir, o.Recursive)
	if err != nil {
		return rep, err
	}

	for _, f := range folders {
		if err := ctx.Err(); err != nil {
			return rep, err
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

		// Pass 4: transcode audio. Runs after the embedded-art pass so the
		// (already-resized) cover stream is copied into the new file.
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
