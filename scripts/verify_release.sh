#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
found=0
for zip_path in "$ROOT"/bin/CursorAssistant-*.zip; do
  if [[ ! -f "$zip_path" ]]; then
    continue
  fi
  found=1
  unzip -t "$zip_path" >/dev/null
done
if [[ "$found" -eq 0 ]]; then
  echo "no release zip artifacts found in $ROOT/bin" >&2
  exit 1
fi

for arch in amd64 arm64; do
  app_bin="$ROOT/bin/darwin-$arch/Cursor助手.app/Contents/MacOS/Cursor助手"
  zip_path="$ROOT/bin/CursorAssistant-darwin-$arch.app.zip"
  if [[ -e "$app_bin" || -e "$zip_path" ]]; then
    test -f "$app_bin"
    test -f "$zip_path"
  fi
done

for arch in amd64 arm64; do
  exe_path="$ROOT/bin/windows-$arch/CursorAssistant.exe"
  zip_path="$ROOT/bin/CursorAssistant-windows-$arch.zip"
  if [[ -e "$exe_path" || -e "$zip_path" ]]; then
    test -f "$exe_path"
    test -f "$zip_path"
  fi
done
echo "release artifacts verified"
