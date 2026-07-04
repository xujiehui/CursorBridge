#!/usr/bin/env python3
from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / ".github" / "workflows" / "package.yml"


def main() -> int:
    text = WORKFLOW.read_text(encoding="utf-8")
    errors: list[str] = []
    matrix = parse_matrix(text)
    expected_targets = {
        "darwin-amd64": {
            "runner": "macos-15-intel",
            "script": "./scripts/package_darwin.sh amd64",
            "artifact_name": "CursorAssistant-darwin-amd64",
            "artifact_path": "bin/CursorAssistant-darwin-amd64.app.zip",
        },
        "darwin-arm64": {
            "runner": "macos-14",
            "script": "./scripts/package_darwin.sh arm64",
            "artifact_name": "CursorAssistant-darwin-arm64",
            "artifact_path": "bin/CursorAssistant-darwin-arm64.app.zip",
        },
        "windows-amd64": {
            "runner": "ubuntu-latest",
            "script": "./scripts/package_windows.sh amd64",
            "artifact_name": "CursorAssistant-windows-amd64",
            "artifact_path": "bin/CursorAssistant-windows-amd64.zip",
        },
    }
    expected_snippets = [
        "permissions:\n  contents: read",
        "GOTOOLCHAIN: go1.25.11",
        'GO_VERSION: "1.25.11"',
        'NODE_VERSION: "20"',
        "runs-on: ${{ matrix.runner }}",
        "run: ${{ matrix.script }}",
        "unzip -t \"${{ matrix.artifact_path }}\"",
        "uses: actions/upload-artifact@v4",
        "uses: softprops/action-gh-release@v2",
        "permissions:\n      contents: write",
    ]
    for snippet in expected_snippets:
        if snippet not in text:
            errors.append(f"missing workflow snippet: {snippet!r}")
    for target, expected in expected_targets.items():
        actual = matrix.get(target)
        if actual is None:
            errors.append(f"missing package matrix target: {target}")
            continue
        for key, value in expected.items():
            if actual.get(key) != value:
                errors.append(
                    f"{target}.{key} expected {value!r}, got {actual.get(key)!r}"
                )

    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    print("workflow package matrix verified")
    return 0


def parse_matrix(text: str) -> dict[str, dict[str, str]]:
    entries: dict[str, dict[str, str]] = {}
    current: dict[str, str] | None = None
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if line.startswith("- target: "):
            target = line.split(": ", 1)[1]
            current = {"target": target}
            entries[target] = current
            continue
        if current is None:
            continue
        if line.startswith("- "):
            current = None
            continue
        if ": " not in line:
            continue
        key, value = line.split(": ", 1)
        if key in {"runner", "script", "artifact_name", "artifact_path"}:
            current[key] = value
    return entries


if __name__ == "__main__":
    raise SystemExit(main())
