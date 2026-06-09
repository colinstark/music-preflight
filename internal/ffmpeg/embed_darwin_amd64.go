//go:build embed_ffmpeg && darwin && amd64

package ffmpeg

import _ "embed"

//go:embed bin/ffmpeg-darwin-amd64
var ffmpegBinary []byte

const exeSuffix = ""
