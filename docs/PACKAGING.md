# Packaging

This project targets Wails v3 and Go 1.25.

## Prerequisites

- Go 1.25+
- Node.js 18+
- npm
- Wails v3 CLI at `.tools/bin/wails3`
- macOS packaging: Xcode with macOS SDK 13+ preferred; SDK 12.x is supported by `scripts/package_darwin.sh` via a small local Wails compatibility patch
- Windows packaging: no Windows host is required for the current amd64 zip package; WebView2 is expected on target Windows machines

Install Wails CLI locally:

```bash
mkdir -p .tools/bin
GOBIN="$PWD/.tools/bin" GOTOOLCHAIN=go1.25.11 go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.103
```

Check packaging prerequisites:

```bash
./scripts/doctor_packaging.sh
```

## Windows

```bash
./scripts/package_windows.sh amd64
```

Output:

```text
bin/CursorAssistant-windows-amd64.zip
```

Verify:

```bash
file bin/windows-amd64/CursorAssistant.exe
unzip -t bin/CursorAssistant-windows-amd64.zip
```

## macOS

```bash
./scripts/package_darwin.sh
```

Output:

```text
bin/CursorAssistant-darwin-<arch>.app.zip
```

The macOS Wails desktop build must run on the target CPU architecture. Local packaging builds the current host architecture; GitHub Actions builds `darwin-amd64` on an Intel macOS runner and `darwin-arm64` on an Apple Silicon runner.

The local script applies ad-hoc signing (`codesign --sign -`) when `codesign` is available. Distribution outside your own machine still needs Developer ID signing and notarization.

Current workspace note: Xcode 13.2.1 provides macOS SDK 12.1, while the pinned Wails v3 alpha references `SMAppService` at compile time. `scripts/package_darwin.sh` applies a local module-cache patch that calls `SMAppService` dynamically so the app can still be built with SDK 12.x.

## GitHub Actions

The workflow at `.github/workflows/package.yml` builds and uploads platform archives.

Check whether the repository is ready to trigger the workflow:

```bash
./scripts/check_ci_ready.sh
```

Triggers:

- `workflow_dispatch`: manual package run from the Actions tab.
- `push` to `main` or `master`: verifies and uploads build artifacts.
- `push` of tags matching `v*`: verifies, uploads artifacts, and publishes a GitHub Release with zip assets.
- `pull_request` to `main` or `master`: runs verification only.

Artifacts:

```text
CursorAssistant-darwin-amd64.app.zip
CursorAssistant-darwin-arm64.app.zip
CursorAssistant-windows-amd64.zip
```

Runner mapping:

- `darwin-amd64`: `macos-15-intel`
- `darwin-arm64`: `macos-14`
- `windows-amd64`: `ubuntu-latest`

Release publishing uses `GITHUB_TOKEN` through `softprops/action-gh-release`; no signing credentials are required for the current ad-hoc packages. Public distribution should still add Developer ID/notarization for macOS and code signing for Windows.
