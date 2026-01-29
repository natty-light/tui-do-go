#!/usr/bin/env bash
set -euo pipefail

# Cross-compile tui-do for macOS and Windows
# Usage:
#   scripts/build.sh [version]
# - version: Optional. A label for the release folder (e.g., v1.0.0). If omitted,
#            we'll try `git describe --tags --always` and fall back to a timestamp.

# Determine version label
if [[ ${1-} ]]; then
  VERSION="$1"
else
  if command -v git >/dev/null 2>&1 && git rev-parse --git-dir >/dev/null 2>&1; then
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || date +%Y%m%d-%H%M%S)
  else
    VERSION=$(date +%Y%m%d-%H%M%S)
  fi
fi

APP_NAME="tui-do"
OUT_DIR="dist/${VERSION}"
mkdir -p "$OUT_DIR"

# Disable CGO for portable static-ish builds
export CGO_ENABLED=0

# List of target tuples: OS ARCH EXT
# Note: On Windows we add .exe extension
TARGETS=(
  "darwin amd64"
  "darwin arm64"
  "windows amd64 .exe"
  "windows arm64 .exe"
)

build_one() {
  local os="$1" arch="$2" ext="${3-}"
  local out_name="${APP_NAME}-${os}-${arch}${ext}"
  echo "Building ${out_name}..."
  GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-s -w" -o "${OUT_DIR}/${out_name}" ./
}

for t in "${TARGETS[@]}"; do
  # shellcheck disable=SC2086
  build_one $t
done

echo "\nArtifacts:"; ls -l "$OUT_DIR" || true

# Optionally create ZIP archives if 'zip' is available
if command -v zip >/dev/null 2>&1; then
  echo "\nZipping artifacts..."
  ( cd "$OUT_DIR" && for f in *; do
      # Create per-file zip alongside the binary for easy GH upload
      # Skip already-zipped files
      [[ "$f" == *.zip ]] && continue
      zip -q "${f}.zip" "$f" || true
    done )
  echo "Done."
else
  echo "\nTip: 'zip' not found; skipping archives. Install zip to auto-generate .zip files."
fi

echo "\nAll done. Binaries are in ${OUT_DIR}"
