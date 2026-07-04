#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WAILS="$ROOT/.tools/bin/wails3"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.11}"
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"

echo "Packaging doctor"
echo "================"
go version
node --version
npm --version
if [[ -x "$WAILS" ]]; then
  "$WAILS" version
else
  echo "wails3 missing at $WAILS"
fi

if command -v xcodebuild >/dev/null 2>&1; then
  xcodebuild -version | head -2
fi
if command -v xcrun >/dev/null 2>&1; then
  sdk_version="$(xcrun --sdk macosx --show-sdk-version 2>/dev/null || true)"
  echo "macOS SDK: ${sdk_version:-unknown}"
  major="${sdk_version%%.*}"
  if [[ -n "$sdk_version" && "$major" =~ ^[0-9]+$ && "$major" -lt 13 ]]; then
    echo "macOS SDK is older than Wails v3's SMAppService compile-time target; package_darwin.sh will apply a local compatibility patch."
  fi
fi
