#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_ARCH="${1:-${GOARCH:-amd64}}"
case "$TARGET_ARCH" in
  amd64|arm64) ;;
  *)
    echo "unsupported windows architecture: $TARGET_ARCH" >&2
    exit 1
    ;;
esac

OUT_DIR="$ROOT/bin/windows-${TARGET_ARCH}"
ZIP_PATH="$ROOT/bin/CursorAssistant-windows-${TARGET_ARCH}.zip"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.11}"
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"

mkdir -p "$OUT_DIR"
(
  cd "$ROOT/frontend"
  npm run build
)
(
  cd "$ROOT"
  GOOS=windows GOARCH="$TARGET_ARCH" CGO_ENABLED=0 go build -tags desktop -ldflags=-H=windowsgui -o "$OUT_DIR/CursorAssistant.exe" .
)
cat > "$OUT_DIR/README.txt" <<'TXT'
Cursor Assistant for Windows

Run CursorAssistant.exe. Configure Cursor to use http://127.0.0.1:18080 as the proxy after installing the generated local CA certificate from the app.
TXT

rm -f "$ZIP_PATH"
cd "$OUT_DIR"
zip -q -r "$ZIP_PATH" .
echo "$ZIP_PATH"
