//go:build embed_ffmpeg

package ffmpeg

// ffmpegVersion keys the extracted binary in the cache dir. Bump it when you
// refresh the bundled static ffmpeg (via `make fetch-ffmpeg`) so users pick up
// the new binary instead of a stale cached copy.
const ffmpegVersion = "7"
