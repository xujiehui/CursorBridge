#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="Cursor助手"
TARGET_ARCH="${1:-${GOARCH:-$(go env GOARCH)}}"
case "$TARGET_ARCH" in
  amd64|arm64) ;;
  *)
    echo "unsupported darwin architecture: $TARGET_ARCH" >&2
    exit 1
    ;;
esac
HOST_ARCH="$(go env GOARCH)"
if [[ "$(go env GOOS)" == "darwin" && "$TARGET_ARCH" != "$HOST_ARCH" ]]; then
  echo "macOS Wails desktop builds must run on the target architecture; host is $HOST_ARCH, target is $TARGET_ARCH." >&2
  echo "Use GitHub Actions macOS runner matrix for cross-platform release builds." >&2
  exit 1
fi

APP_DIR="$ROOT/bin/darwin-${TARGET_ARCH}/${APP_NAME}.app"
CONTENTS="$APP_DIR/Contents"
MACOS="$CONTENTS/MacOS"
RESOURCES="$CONTENTS/Resources"
ZIP_PATH="$ROOT/bin/CursorAssistant-darwin-${TARGET_ARCH}.app.zip"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.11}"
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"

sdk_version="$(xcrun --sdk macosx --show-sdk-version 2>/dev/null || true)"
major="${sdk_version%%.*}"
if [[ -n "$sdk_version" && "$major" =~ ^[0-9]+$ && "$major" -lt 13 ]]; then
  echo "macOS SDK $sdk_version is older than Wails v3's SMAppService compile-time target; applying local compatibility patch."
  "$ROOT/scripts/patch_wails_old_macos_sdk.sh"
fi

(
  cd "$ROOT/frontend"
  npm run build
)
(
  cd "$ROOT"
  GOOS=darwin GOARCH="$TARGET_ARCH" go build -tags desktop -o "$ROOT/bin/cursor-assistant-desktop-darwin-${TARGET_ARCH}" .
)

rm -rf "$APP_DIR"
mkdir -p "$MACOS" "$RESOURCES"
cp "$ROOT/bin/cursor-assistant-desktop-darwin-${TARGET_ARCH}" "$MACOS/$APP_NAME"
chmod +x "$MACOS/$APP_NAME"

cat > "$CONTENTS/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>zh_CN</string>
  <key>CFBundleExecutable</key>
  <string>Cursor助手</string>
  <key>CFBundleIdentifier</key>
  <string>com.cursor-assistant.app</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>Cursor助手</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>0.1.0</string>
  <key>CFBundleVersion</key>
  <string>0.1.0</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

if command -v codesign >/dev/null 2>&1; then
  codesign --force --deep --sign - "$APP_DIR"
fi

rm -f "$ZIP_PATH"
(
  cd "$(dirname "$APP_DIR")"
  ditto -c -k --keepParent "$(basename "$APP_DIR")" "$ZIP_PATH"
)
echo "$APP_DIR"
echo "$ZIP_PATH"
