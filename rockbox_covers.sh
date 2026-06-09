#!/usr/bin/env bash
# rockbox_covers.sh
# Recursively processes a music folder:
#   1. Renames any .jpg not called cover.jpg to cover.jpg
#   2. Resizes all cover.jpg files to 360x360 baseline (non-progressive) JPGs
#   3. Extracts embedded artwork from mp3/aac/m4a where cover.jpg is missing
#
# Requirements: vips, vipsthumbnail, ffmpeg
#
# Usage:
#   chmod +x rockbox_covers.sh
#   ./rockbox_covers.sh [--dry-run] /path/to/music/folder

set -euo pipefail

# ── Argument handling ────────────────────────────────────────────────────────

DRY_RUN=false
MUSIC_DIR=""

for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=true ;;
    *) MUSIC_DIR="$arg" ;;
  esac
done

if [[ -z "$MUSIC_DIR" ]]; then
  echo "Usage: $0 [--dry-run] /path/to/music/folder"
  exit 1
fi

if [[ ! -d "$MUSIC_DIR" ]]; then
  echo "Error: '$MUSIC_DIR' is not a directory."
  exit 1
fi

if ! command -v vips &>/dev/null; then
  echo "Error: 'vips' is not installed or not in PATH."
  exit 1
fi

if ! command -v ffmpeg &>/dev/null; then
  echo "Warning: 'ffmpeg' not found — embedded artwork extraction will be skipped."
  HAS_FFMPEG=false
else
  HAS_FFMPEG=true
fi

# ── Counters ─────────────────────────────────────────────────────────────────

RENAMED=0
RESIZED=0
SKIPPED=0
EXTRACTED=0
FAILED=0

# ── Helper: resize a jpg to 360x360 baseline in-place ────────────────────────

resize_jpg() {
  local jpg="$1"
  local tmp
  tmp=$(mktemp "${jpg}.tmp.XXXXXX")
  if vipsthumbnail "$jpg" \
      --size 360x360 \
      --smartcrop none \
      -o "${tmp}.jpg[Q=85,interlace=false]" 2>/dev/null && \
      mv "${tmp}.jpg" "$tmp"; then
    mv "$tmp" "$jpg"
    return 0
  else
    rm -f "$tmp" "${tmp}.jpg"
    return 1
  fi
}

# ── Pass 1: rename + resize existing JPGs ────────────────────────────────────

echo "Scanning: $MUSIC_DIR"
$DRY_RUN && echo "(dry-run mode — no files will be modified)"
echo ""
echo "── Pass 1: JPG files ───────────────────────────────────────────────────────"

while IFS= read -r -d '' jpg; do
  dir=$(dirname "$jpg")
  base=$(basename "$jpg")
  base_lower=$(echo "$base" | tr '[:upper:]' '[:lower:]')

  # Rename to cover.jpg if not already
  if [[ "$base_lower" != "cover.jpg" ]]; then
    newpath="$dir/cover.jpg"
    echo "  rename  $jpg → $newpath"
    if ! $DRY_RUN; then
      mv "$jpg" "$newpath"
      ((RENAMED++))
    else
      ((RENAMED++))
    fi
    jpg="$newpath"
  fi

  # Check dimensions and interlace
  width=$(vipsheader -f width "$jpg" 2>/dev/null || echo 0)
  height=$(vipsheader -f height "$jpg" 2>/dev/null || echo 0)
  interlace=$(vipsheader -f jpeg-multiscan "$jpg" 2>/dev/null || echo 0)

  if [[ "$width" -eq 360 && "$height" -eq 360 && "$interlace" -eq 0 ]]; then
    echo "  skip    $jpg  (already 360×360, baseline)"
    ((SKIPPED++))
    continue
  fi

  echo "  resize  $jpg  (${width}×${height}, interlace=${interlace} → 360×360, baseline)"

  if ! $DRY_RUN; then
    if resize_jpg "$jpg"; then
      ((RESIZED++))
    else
      echo "  ERROR  $jpg"
      ((FAILED++))
    fi
  else
    ((RESIZED++))
  fi

done < <(find "$MUSIC_DIR" -type f -iname "*.jpg" ! -name "._*" -print0)

# ── Pass 2: extract embedded artwork from audio files ────────────────────────

echo ""
echo "── Pass 2: embedded artwork ────────────────────────────────────────────────"

if ! $HAS_FFMPEG; then
  echo "  skipped (ffmpeg not available)"
else
  while IFS= read -r -d '' audio; do
    dir=$(dirname "$audio")
    cover="$dir/cover.jpg"

    if [[ -f "$cover" ]]; then
      continue
    fi

    echo "  extract  $audio → $cover"

    if ! $DRY_RUN; then
      if ffmpeg -i "$audio" -an -vcodec copy "$cover" -y 2>/dev/null; then
        if resize_jpg "$cover"; then
          echo "    ok"
          ((EXTRACTED++))
        else
          echo "    ERROR resizing extracted artwork"
          rm -f "$cover"
          ((FAILED++))
        fi
      else
        echo "    no embedded artwork found"
        rm -f "$cover"
      fi
    else
      ((EXTRACTED++))
    fi

  done < <(find "$MUSIC_DIR" -type f \( -iname "*.mp3" -o -iname "*.aac" -o -iname "*.m4a" \) ! -name "._*" -print0)
fi

# ── Summary ──────────────────────────────────────────────────────────────────

echo ""
echo "Done."
$DRY_RUN && echo "  (dry-run — no changes made)"
echo "  Renamed   : $RENAMED"
echo "  Resized   : $RESIZED"
echo "  Extracted : $EXTRACTED"
echo "  Skipped   : $SKIPPED (already correct)"
[[ $FAILED -gt 0 ]] && echo "  Failed    : $FAILED"
