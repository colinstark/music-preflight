//go:build !embed_ffmpeg

package ffmpeg

import (
	"fmt"
	"os/exec"
)

// locate finds a system-installed ffmpeg. Used for development builds and any
// build not made with -tags embed_ffmpeg.
func locate() (string, error) {
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("ffmpeg not found on PATH; install ffmpeg, " +
		"or build coverfixer with -tags embed_ffmpeg to bundle it")
}
