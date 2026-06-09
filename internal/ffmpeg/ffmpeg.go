// Package ffmpeg supplies a usable ffmpeg executable to the rest of coverfixer.
//
// ffmpeg is only needed for the optional transcode feature, so the dependency is
// resolved lazily. There are two build variants:
//
//   - default build: locate() finds ffmpeg on PATH (see system.go). Transcoding
//     works wherever the user has ffmpeg installed, and `go build`/tests never
//     require the large static binary.
//   - release build (-tags embed_ffmpeg): a static ffmpeg is compiled into the
//     binary via //go:embed and extracted to the user cache dir on first use
//     (see embed.go + the per-platform embed_*.go files). This is the
//     self-contained single-file distribution.
package ffmpeg

// Path returns the path to a runnable ffmpeg executable, or an error explaining
// how to make one available.
func Path() (string, error) {
	return locate()
}
