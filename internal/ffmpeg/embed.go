//go:build embed_ffmpeg

package ffmpeg

import (
	"fmt"
	"os"
	"path/filepath"
)

// ffmpegBinary, ffmpegVersion and exeSuffix are provided by the per-platform
// embed_<goos>_<goarch>.go file selected by build constraints.

// locate writes the embedded static ffmpeg to the user cache dir on first use
// and returns its path. The binary is keyed by ffmpegVersion so it is only
// written once per version.
func locate() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		cache = os.TempDir()
	}
	dir := filepath.Join(cache, "coverfixer")
	target := filepath.Join(dir, "ffmpeg-"+ffmpegVersion+exeSuffix)

	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		return target, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create ffmpeg cache dir: %w", err)
	}

	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, ffmpegBinary, 0o755); err != nil {
		return "", fmt.Errorf("write bundled ffmpeg: %w", err)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("install bundled ffmpeg: %w", err)
	}
	return target, nil
}
