#!/usr/bin/env bash
# Fetch the custom TTFs that `divoom push` installs on the Times Frame.
# Run once on the USB-attached host before the first `divoom push`. Files
# land in ./fonts/, which is gitignored.
#
# Two weights per the wallclock-scene design review: Archivo SemiBold
# (header/weather/room rows) and Archivo ExtraBold (clock only). Google
# Fonts no longer distributes Archivo as static per-weight files (only
# the variable Archivo[wdth,wght].ttf) — this codebase's font loader has
# no proven variable-font axis support, so we pin the two weights we need
# out of the variable font with `fonttools varLib.instancer`, producing
# real static TTFs.
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

if ! command -v fonttools >/dev/null 2>&1; then
  echo "fonttools not found — install it first (e.g. \`apt-get install -y fonttools\` or \`pip install fonttools\`)" >&2
  exit 1
fi

variable="fonts/Archivo-Variable.ttf"
fetch \
  "https://raw.githubusercontent.com/google/fonts/main/ofl/archivo/Archivo%5Bwdth,wght%5D.ttf" \
  "$variable"

instance() {
  local wght="$1" dst="$2"
  if [ -s "$dst" ]; then
    echo "  have $dst"
    return
  fi
  echo "  instance $dst (wght=$wght)"
  fonttools varLib.instancer -q -o "$dst" "$variable" "wght=$wght"
}

instance 600 fonts/Archivo-SemiBold.ttf
instance 800 fonts/Archivo-ExtraBold.ttf

echo "done — run \`go run ./cmd/divoom push\` to install on the frame"
