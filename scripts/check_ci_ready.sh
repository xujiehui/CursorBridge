#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
missing=0

check() {
  local label="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    printf "ok: %s\n" "$label"
  else
    printf "missing: %s\n" "$label" >&2
    missing=1
  fi
}

check "GitHub Actions workflow" test -f "$ROOT/.github/workflows/package.yml"
check "workflow package matrix" python3 "$ROOT/scripts/verify_workflow.py"
check "macOS package script" test -x "$ROOT/scripts/package_darwin.sh"
check "Windows package script" test -x "$ROOT/scripts/package_windows.sh"
check "release verifier" test -x "$ROOT/scripts/verify_release.sh"
check "GitHub Actions verifier" test -x "$ROOT/scripts/verify_github_actions.py"

if git -C "$ROOT" remote get-url origin >/dev/null 2>&1; then
  printf "ok: git remote origin (%s)\n" "$(git -C "$ROOT" remote get-url origin)"
else
  printf "missing: git remote origin\n" >&2
  missing=1
fi

if command -v gh >/dev/null 2>&1; then
  printf "ok: GitHub CLI (%s)\n" "$(gh --version | head -1)"
else
  printf "missing: GitHub CLI (optional, needed for local workflow dispatch/status checks)\n" >&2
fi

if git -C "$ROOT" rev-parse --verify HEAD >/dev/null 2>&1; then
  printf "ok: repository has at least one commit\n"
else
  printf "missing: repository has no commits yet\n" >&2
  missing=1
fi

exit "$missing"
