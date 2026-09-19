#!/usr/bin/env bash
# Fetch the custom TTF that `divoom push` installs on the Times
# Frame. Run once on the USB-attached host before the first `divoom push`.
# Files land in ./fonts/, which is gitignored.
#
# Single family per the Modernist-pairing design review: Archivo Black,
# a standalone static weight in Google Fonts' distribution (not an
# instance pulled out of the variable Archivo[wdth,wght].ttf — this
# codebase's font loader has no proven variable-font axis support, so a
# real static file is the safe choice). Used at multiple sizes rather
# than multiple weights.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p fonts

fetch() {
  local url="$1" dst="$2"
  if [ -s "$dst" ]; then
    echo "  have $dst"
    return
  fi
  echo "  fetch $dst"
  curl -fsSL --retry 3 -o "$dst" "$url"
}

fetch \
  "https://raw.githubusercontent.com/google/fonts/main/ofl/archivoblack/ArchivoBlack-Regular.ttf" \
  fonts/ArchivoBlack-Regular.ttf

echo "done — run \`go run ./cmd/divoom push\` to install on the frame"
