#!/usr/bin/env python3
from __future__ import annotations

import json
import os
import subprocess
import sys
import urllib.error
import urllib.request


WORKFLOW = "package.yml"
EXPECTED_ARTIFACTS = {
    "CursorAssistant-darwin-amd64",
    "CursorAssistant-darwin-arm64",
    "CursorAssistant-windows-amd64",
}


def main() -> int:
    repo = sys.argv[1] if len(sys.argv) > 1 else infer_repo()
    if not repo:
        print(
            "usage: scripts/verify_github_actions.py owner/repo",
            file=sys.stderr,
        )
        print(
            "or configure git remote origin to a GitHub repository",
            file=sys.stderr,
        )
        return 2

    token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")
    runs = api_json(
        f"https://api.github.com/repos/{repo}/actions/workflows/{WORKFLOW}/runs?per_page=10",
        token,
    )
    successful = [
        run
        for run in runs.get("workflow_runs", [])
        if run.get("conclusion") == "success"
        and run.get("status") == "completed"
        and run.get("event") != "pull_request"
    ]
    if not successful:
        print(f"no successful non-PR {WORKFLOW} run found for {repo}", file=sys.stderr)
        return 1

    run = successful[0]
    artifacts = api_json(
        f"https://api.github.com/repos/{repo}/actions/runs/{run['id']}/artifacts?per_page=100",
        token,
    )
    names = {artifact.get("name") for artifact in artifacts.get("artifacts", [])}
    missing = sorted(EXPECTED_ARTIFACTS - names)
    if missing:
        print(f"run {run['id']} is missing artifacts: {', '.join(missing)}", file=sys.stderr)
        return 1

    print(f"ok: {repo} {WORKFLOW} run {run['id']} completed successfully")
    for name in sorted(EXPECTED_ARTIFACTS):
        print(f"ok: artifact {name}")
    return 0


def infer_repo() -> str | None:
    try:
        remote = subprocess.check_output(
            ["git", "remote", "get-url", "origin"],
            text=True,
            stderr=subprocess.DEVNULL,
        ).strip()
    except subprocess.CalledProcessError:
        return None
    if remote.startswith("git@github.com:"):
        repo = strip_suffix(remote[len("git@github.com:") :], ".git")
        return repo or None
    marker = "github.com/"
    if marker in remote:
        repo = strip_suffix(remote.split(marker, 1)[1], ".git")
        return repo or None
    return None


def strip_suffix(value: str, suffix: str) -> str:
    if value.endswith(suffix):
        return value[: -len(suffix)]
    return value


def api_json(url: str, token: str | None) -> dict:
    request = urllib.request.Request(
        url,
        headers={
            "Accept": "application/vnd.github+json",
            "User-Agent": "CursorBridge-CI-Verifier",
        },
    )
    if token:
        request.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            return json.load(response)
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        print(f"GitHub API error {exc.code}: {body}", file=sys.stderr)
        raise SystemExit(1) from exc


if __name__ == "__main__":
    raise SystemExit(main())
