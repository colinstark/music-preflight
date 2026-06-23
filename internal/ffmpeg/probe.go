package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProbeResult holds the read-only metadata ffmpeg reports for a file.
type ProbeResult struct {
	Tags     map[string]string // lowercased tag name -> value (title, artist, album, genre, track, ...)
	Duration float64           // seconds; 0 if ffmpeg could not determine it
	HasArt   bool              // an embedded picture (video stream) was detected
}

// probeTimeout caps a single per-file probe so one pathological file cannot
// stall the whole library scan.
const probeTimeout = 15 * time.Second

// Probe reads a file's format tags, duration, and presence of embedded artwork
// using ffmpeg. It writes the ffmetadata block to stdout (parsed into Tags) and
// reads the "Duration:" line and stream list from stderr. The file is never
// modified. Probe is best-effort: a file ffmpeg cannot parse returns an error
// that the caller is expected to skip.
func Probe(path string) (ProbeResult, error) {
	bin, err := Path()
	if err != nil {
		return ProbeResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin,
		"-hide_banner", "-nostdin",
		"-i", path,
		"-f", "ffmetadata", "pipe:1",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return ProbeResult{}, fmt.Errorf("ffmpeg probe %q: %w", path, err)
	}

	return ProbeResult{
		Tags:     parseFFMetadata(stdout.Bytes()),
		Duration: parseDuration(stderr.String()),
		HasArt:   hasVideoStream(stderr.String()),
	}, nil
}

// ExtractThumb extracts and down-scales the first embedded picture to a small
// JPEG, returned as raw bytes (suitable for a data:image/jpeg URL). It returns
// nil, nil when the file has no artwork or extraction fails — callers render a
// placeholder in that case.
func ExtractThumb(path string, size int) ([]byte, error) {
	bin, err := Path()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, bin,
		"-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", path,
		"-map", "0:v:0",
		"-frames:v", "1",
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", size, size),
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-q:v", "4",
		"pipe:1",
	)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, nil
	}
	if stdout.Len() == 0 {
		return nil, nil
	}
	return stdout.Bytes(), nil
}

// parseFFMetadata parses ffmpeg's "ffmetadata" output (key=value lines, ;-comments,
// [stream]/[chapter] section headers) into a lowercased-key map. Only the
// format-level tags before the first section header are kept, which is the
// album/artist/title/genre/track set we care about.
func parseFFMetadata(b []byte) map[string]string {
	tags := map[string]string{}
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line[0] == ';' {
			continue
		}
		if line[0] == '[' {
			break // format-level tags end at the first [STREAM]/[CHAPTER] section
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(line[:eq]))
		if k != "" {
			tags[k] = strings.TrimSpace(line[eq+1:])
		}
	}
	return tags
}

var (
	durationRe    = regexp.MustCompile(`Duration:\s*(\d+):(\d{2}):(\d{2}(?:\.\d+)?)`)
	videoStreamRe = regexp.MustCompile(`Stream #\d+:\d+[^ ]*:\s*Video:`)
)

func parseDuration(stderr string) float64 {
	m := durationRe.FindStringSubmatch(stderr)
	if m == nil {
		return 0
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	sec, _ := strconv.ParseFloat(m[3], 64)
	return float64(h)*3600 + float64(min)*60 + sec
}

func hasVideoStream(stderr string) bool { return videoStreamRe.MatchString(stderr) }
