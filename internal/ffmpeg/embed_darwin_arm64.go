//go:build embed_ffmpeg && darwin && arm64

package ffmpeg

import _ "embed"

//go:embed bin/ffmpeg-darwin-arm64
var ffmpegBinary []byte

const exeSuffix = ""
