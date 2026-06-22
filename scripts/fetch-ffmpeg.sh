#!/usr/bin/env bash
# fetch-ffmpeg.sh GOOS GOARCH DEST_DIR
#
# Downloads a static ffmpeg build for the given platform and installs it as
# DEST_DIR/ffmpeg-<goos>-<goarch>[.exe], where the per-platform embed_*.go files
# expect it. These static builds are GPL (they include libmp3lame); a binary
# built with -tags embed_ffmpeg is therefore covered by the GPL.
#
# Integrity: each fetched binary is verified against a pinned SHA-256 in
# expected_sha256() before it is accepted. The release build embeds this binary
# and later execs it, so an unverified download is a supply-chain risk. When a
# digest is not yet pinned, the script prints the fetched digest and exits
# non-zero so you pin it from a trusted upstream source; set
# COVERFIXER_NO_FFMPEG_VERIFY=1 only for a conscious, trusted re-pin.
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

# Expected SHA-256 of the fetched ffmpeg per platform, from a trusted source
# (each upstream publishes digests). Empty = not pinned.
expected_sha256() {
	case "$1" in
		darwin-arm64)  echo "" ;;
		darwin-amd64)  echo "" ;;
		linux-amd64)   echo "" ;;
		windows-amd64) echo "" ;;
	esac
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1"
	else
		shasum -a 256 "$1"
	fi | awk '{print $1}'
}

# verify checks the binary at $1 against the pinned digest, or prints the
# fetched digest and returns non-zero when no digest is pinned (unless the
# caller set COVERFIXER_NO_FFMPEG_VERIFY=1).
verify() {
	local actual expected
	actual="$(sha256 "$1")"
	expected="$(expected_sha256 "$key")"
	if [[ -n "$expected" ]]; then
		if [[ "$actual" != "$expected" ]]; then
			echo "SHA-256 mismatch for ${key}:" >&2
			echo "  expected $expected" >&2
			echo "  got      $actual" >&2
			echo "  refusing to install/embed a binary that fails the integrity check." >&2
			return 1
		fi
		echo "Verified SHA-256 (${key}): $actual"
		return 0
	fi
	if [[ "${COVERFIXER_NO_FFMPEG_VERIFY:-}" == "1" ]]; then
		echo "WARNING: COVERFIXER_NO_FFMPEG_VERIFY=1; skipping integrity check." >&2
		echo "  fetched SHA-256 (${key}): $actual  — pin this in expected_sha256()" >&2
		return 0
	fi
	echo "No SHA-256 pinned for ${key}." >&2
	echo "  fetched SHA-256: $actual" >&2
	echo "  Pin this in expected_sha256() from a trusted upstream source, or re-run" >&2
	echo "  with COVERFIXER_NO_FFMPEG_VERIFY=1 to accept an unverified binary." >&2
	return 1
}

mkdir -p "$DEST"

if [[ -f "$out" ]]; then
	echo "ffmpeg already present: $out"
	verify "$out" || exit 1
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

verify "$out" || { rm -f "$out"; exit 1; }

chmod +x "$out" 2>/dev/null || true
echo "Installed: $out"
