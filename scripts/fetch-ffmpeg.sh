#!/usr/bin/env bash
# fetch-ffmpeg.sh GOOS GOARCH DEST_DIR
#
# Downloads a static ffmpeg build for the given platform and installs it as
# DEST_DIR/ffmpeg-<goos>-<goarch>[.exe], where the per-platform embed_*.go files
# expect it. These static builds are GPL (they include libmp3lame); a binary
# built with -tags embed_ffmpeg is therefore covered by the GPL.
#
# The upstream URLs below move over time. If a download 404s, update the URL for
# your platform — any static ffmpeg with libmp3lame + aac + mjpeg will do.
set -euo pipefail

GOOS="${1:?usage: fetch-ffmpeg.sh GOOS GOARCH DEST_DIR}"
GOARCH="${2:?missing GOARCH}"
DEST="${3:?missing DEST_DIR}"

key="${GOOS}-${GOARCH}"
out="${DEST}/ffmpeg-${key}"
[[ "$GOOS" == "windows" ]] && out="${out}.exe"

if [[ -f "$out" ]]; then
  echo "ffmpeg already present: $out"
  exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Fetching static ffmpeg for ${key}..."
case "$key" in
  darwin-arm64)
    url="https://www.osxexperts.net/ffmpeg71arm.zip"
    curl -fSL "$url" -o "$tmp/ff.zip"
    unzip -o -j "$tmp/ff.zip" -d "$tmp" >/dev/null
    cp "$tmp/ffmpeg" "$out"
    ;;
  darwin-amd64)
    url="https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip"
    curl -fSL "$url" -o "$tmp/ff.zip"
    unzip -o -j "$tmp/ff.zip" -d "$tmp" >/dev/null
    cp "$tmp/ffmpeg" "$out"
    ;;
  linux-amd64)
    url="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz"
    curl -fSL "$url" -o "$tmp/ff.tar.xz"
    tar -xf "$tmp/ff.tar.xz" -C "$tmp"
    cp "$tmp"/ffmpeg-*-amd64-static/ffmpeg "$out"
    ;;
  windows-amd64)
    url="https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
    curl -fSL "$url" -o "$tmp/ff.zip"
    unzip -o "$tmp/ff.zip" -d "$tmp" >/dev/null
    cp "$tmp"/ffmpeg-*-essentials_build/bin/ffmpeg.exe "$out"
    ;;
  *)
    echo "No download recipe for ${key}; add one to scripts/fetch-ffmpeg.sh" >&2
    exit 1
    ;;
esac

chmod +x "$out" 2>/dev/null || true
echo "Installed: $out"
