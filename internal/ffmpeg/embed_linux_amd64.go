//go:build embed_ffmpeg && linux && amd64

package ffmpeg

import _ "embed"

//go:embed bin/ffmpeg-linux-amd64
var ffmpegBinary []byte

const exeSuffix = ""
