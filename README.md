# Cursor Assistant

Cursor Assistant is a local desktop-app foundation for routing Cursor IDE traffic through a localhost bridge. It follows the Feishu technical document structure: Go backend, Vue/Vite frontend, local proxy, relay gateway, BYOK model adapters, certificate management, Cursor settings integration, JSONL observability, and a Wails-friendly service boundary.

## Current Scope

This repository contains a Wails desktop application and HTTP fallback server:

- Bridge API on `127.0.0.1:8080`
- Local proxy on `127.0.0.1:18080`
- HTTP forwarding and TLS MITM for CONNECT traffic after the generated CA is trusted
- Relay routing with `x-raw-cursor-server-url`
- BYOK JSON gateway for OpenAI-compatible and Anthropic-compatible adapters
- Local config file under the user config directory
- Local CA generation and per-host server certificate generation
- Cursor settings preview/apply API
- Vue management console

System-level actions such as installing a CA certificate or changing Cursor proxy settings are explicit API actions. The app does not mutate system trust or Cursor settings on startup.

## Requirements

- Go 1.25+
- Node.js 18+
- Wails v3 CLI when packaging as a native desktop app
- macOS SDK 13+ preferred for Wails v3 macOS packaging; SDK 12.x is handled by the packaging script's local compatibility patch
- Task CLI is optional; equivalent commands are shown below

The source document mentions Go 1.25+. The project now targets Go 1.25 because Wails v3 requires it.

## Run

```bash
go run . --addr 127.0.0.1:8080 --proxy-addr 127.0.0.1:18080
```

In another terminal:

```bash
cd frontend
npm install
npm run dev
```

Open the Vite URL and use the console against `http://127.0.0.1:8080`.

## Verify

```bash
GOTOOLCHAIN=go1.25.11 go test ./...
GOTOOLCHAIN=go1.25.11 go build ./...
cd frontend && npm run build
```

## Package

Windows amd64:

```bash
./scripts/package_windows.sh amd64
```

macOS:

```bash
./scripts/doctor_packaging.sh
./scripts/package_darwin.sh
```

GitHub Actions builds the release archives from `.github/workflows/package.yml`: macOS amd64, macOS arm64, and Windows amd64. Tags matching `v*` publish those zip files to a GitHub Release.

Check whether the repository is ready to trigger GitHub Actions:

```bash
./scripts/check_ci_ready.sh
```

After creating the GitHub repository:

```bash
git remote add origin https://github.com/<owner>/<repo>.git
git push -u origin master
./scripts/verify_github_actions.py <owner>/<repo>
```

This workspace has Xcode 13.2.1 / macOS SDK 12.1. The macOS packaging script applies a local Wails compatibility patch so the `.app` can still be built.
The local macOS script applies ad-hoc signing when `codesign` is available. Public distribution still needs Developer ID signing and notarization.

## Important Paths

- `internal/mitm`: local proxy server
- `internal/relay`: route decisions and BYOK adapter forwarding
- `internal/certs`: local CA and server certificate management
- `internal/cursor`: Cursor settings integration
- `internal/config`: persisted user configuration
- `internal/observability`: JSONL logs
- `internal/app`: bridge API and service composition
- `frontend`: Vue/Vite control console
