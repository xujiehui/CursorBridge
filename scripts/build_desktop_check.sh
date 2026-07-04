#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.11}"
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"

if [[ "$(go env GOOS)" == "darwin" ]] && command -v xcrun >/dev/null 2>&1; then
  sdk_version="$(xcrun --sdk macosx --show-sdk-version 2>/dev/null || true)"
  major="${sdk_version%%.*}"
  if [[ -n "$sdk_version" && "$major" =~ ^[0-9]+$ && "$major" -lt 13 ]]; then
    "$ROOT/scripts/patch_wails_old_macos_sdk.sh"
  fi
fi

cd "$ROOT"
go build -tags desktop ./...
