# Cursor助手项目技术方案

## 1. 文档说明

本文档根据本地抓取的飞书技术文档 `docs/feishu/cursor-assistant-tech-doc.snapshots.md` 整理，并结合当前仓库实现现状形成项目技术方案。

源文档信息：

- 飞书链接：<https://dcne38qm5vlg.feishu.cn/wiki/K2YHwSbAjilCZ6k3ywQcHnxFn7e>
- 飞书文档更新时间：2026-06-16
- 本地抓取文件：`docs/feishu/cursor-assistant-tech-doc.snapshots.md`
- 当前项目实现：Go + Wails v3 + Vue/Vite 的桌面应用，另保留 HTTP fallback server

## 2. 项目背景与目标

Cursor助手是一个面向 Cursor IDE 的本地桌面辅助应用。项目通过本机代理服务承接 Cursor IDE 的 HTTP/HTTPS 请求，并在本地完成请求路由、必要的请求转换、证书管理、Cursor 设置集成与 BYOK 模型转发。

核心目标：

- 在本机启动代理服务，默认监听 `127.0.0.1:18080`。
- 通过 MITM 代理拦截 Cursor IDE 的 HTTPS 请求。
- 按路由策略将请求分为直接转发、Relay 网关处理、自实现模型网关处理。
- 支持 BYOK（Bring Your Own Key），允许用户配置 OpenAI 或 Anthropic 兼容模型 API Key。
- 通过 Wails v3 提供跨平台桌面 UI，支持 macOS 与 Windows。
- 提供配置、证书、代理状态、Cursor 设置和日志观测能力。

## 3. 范围与边界

### 3.1 本期范围

- 桌面应用壳：Wails v3 原生桌面入口。
- 前端控制台：Vue 3 + Vite 管理界面。
- 本地代理：HTTP 转发、HTTPS CONNECT TLS MITM。
- Relay 网关：统一路由判断与上游转发。
- BYOK 网关：OpenAI/Anthropic 兼容接口转发。
- 证书管理：本地 CA 生成、服务端证书动态生成、安装指引。
- Cursor 集成：Cursor 设置路径识别、代理配置预览与写入。
- 配置持久化：用户目录下保存运行配置和模型适配器。
- 日志观测：JSONL 记录运行与渠道调用事件。
- 打包发布：macOS `.app`/zip 与 Windows amd64 zip。

### 3.2 当前边界

- 飞书原文提到 Protocol Buffers 完整编解码与 Cursor 私有协议文件；当前仓库保留协议扩展方向，但未内置完整 Cursor 私有 proto 文件。
- 公开分发 macOS 应用仍需要 Developer ID 签名与 notarization；当前本地脚本使用 ad-hoc signing。
- CA 安装与 Cursor 设置修改属于系统级动作，当前设计为显式用户操作，不在应用启动时自动执行。

## 4. 技术栈

| 层级 | 技术选型 | 说明 |
| --- | --- | --- |
| 桌面壳 | Wails v3 | Go 服务与 Web 前端集成，支持 macOS/Windows |
| 后端 | Go 1.25 | 代理、Relay、证书、配置、Cursor 集成 |
| 前端 | Vue 3 + Vite 7 | 本地控制台 UI |
| 代理 | Go `net/http` + 自定义 TLS 证书 | 当前实现未直接依赖 goproxy，使用标准库完成代理/MITM |
| 模型网关 | OpenAI/Anthropic 兼容 HTTP API | BYOK 适配器 |
| 配置 | JSON 文件 | 存储在用户配置目录 |
| 日志 | JSONL | `run-usage.jsonl`、`channel-calls.jsonl` |
| 打包 | Shell + Go/Wails | macOS `.app`/zip、Windows amd64 zip |

## 5. 总体架构

```text
┌──────────────────────────────────────────────────────────────┐
│                         Cursor IDE                           │
└──────────────────────────────┬───────────────────────────────┘
                               │ HTTP/HTTPS
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                   MITM Proxy 127.0.0.1:18080                 │
│        HTTP proxy / CONNECT TLS MITM / request handoff       │
└──────────────────────────────┬───────────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
┌────────────────┐    ┌────────────────┐    ┌────────────────┐
│ Direct Forward │    │ Relay Gateway  │    │ BYOK Gateway   │
└───────┬────────┘    └───────┬────────┘    └───────┬────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌────────────────┐    ┌────────────────┐    ┌────────────────┐
│ Cursor Server  │    │ Cursor Server  │    │ OpenAI/Claude  │
└────────────────┘    └────────────────┘    └────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                  Wails Desktop / Vue Console                 │
│  proxy control / config / certs / Cursor settings / status   │
└──────────────────────────────────────────────────────────────┘
```

应用内部采用组合式服务结构：

- `internal/app` 负责组装配置、证书、Cursor 集成、Relay 与代理服务，并暴露 HTTP API。
- `internal/desktop` 将 Go 服务绑定给 Wails 前端。
- `server_main.go` 提供 HTTP fallback server。
- `desktop_main.go` 提供 Wails 桌面入口并嵌入前端产物。

## 6. 目录结构方案

当前仓库核心目录如下：

```text
.
├── desktop_main.go              # Wails 桌面入口
├── server_main.go               # HTTP fallback server 入口
├── frontend/                    # Vue/Vite 前端
├── internal/
│   ├── app/                     # 服务组合与 HTTP API
│   ├── apperrors/               # 统一错误码与响应结构
│   ├── certs/                   # CA 与动态证书管理
│   ├── config/                  # 用户配置读写与校验
│   ├── cursor/                  # Cursor 设置集成
│   ├── desktop/                 # Wails 服务绑定
│   ├── mitm/                    # 本地代理与 TLS MITM
│   ├── observability/           # JSONL 日志
│   └── relay/                   # 路由、转发与 BYOK
├── scripts/                     # 抓取整理、打包、校验脚本
├── docs/
│   ├── feishu/                  # 飞书原文抓取副本
│   ├── PACKAGING.md             # 打包说明
│   └── TECHNICAL_SOLUTION.md    # 本技术方案
└── Taskfile.yml                 # 常用任务入口
```

## 7. 核心模块设计

### 7.1 应用组合层 `internal/app`

职责：

- 初始化配置存储、证书管理器、Cursor 集成、Relay 网关、MITM 代理。
- 暴露状态、配置保存、模型适配器管理、代理启停、证书信息、Cursor 设置等服务能力。
- 提供 HTTP API，使前端在 Vite 开发模式和 fallback server 模式下可复用同一套业务能力。

关键能力：

- `Status()` 返回健康状态、配置路径、数据目录、代理状态、Cursor 集成状态。
- `StartProxy()` / `StopProxy()` 控制本地代理。
- `SaveConfig()` / `UpsertAdapter()` / `DeleteAdapter()` 管理用户配置。
- `CAInfo()` / `CAInstallPlan()` 提供本地 CA 信息与安装命令。
- `CursorPlan()` / `ApplyCursorSettings()` 预览或写入 Cursor 代理设置。
- `Decision()` 复用 Relay 路由判断逻辑，为前端提供路由预览。

### 7.2 桌面绑定层 `internal/desktop`

职责：

- 将 `internal/app` 的服务方法以 Wails service 形式暴露给前端。
- 让桌面模式无需启动独立 HTTP server 即可完成状态查询、配置修改、代理控制和路由预览。

前端 API 客户端优先调用 Wails 绑定；当不在 Wails 环境时降级到 HTTP API。

### 7.3 MITM 代理 `internal/mitm`

职责：

- 默认监听 `127.0.0.1:18080`。
- 处理普通 HTTP 代理请求。
- 处理 HTTPS `CONNECT` 请求：返回 `200 Connection Established` 后，使用本地 CA 动态签发服务端证书，完成 TLS server handshake。
- 将解密后的 HTTP 请求交给 Relay 网关处理。

关键设计：

- 仅监听本机回环地址，避免外部访问。
- TLS 最低版本为 TLS 1.2。
- 若未提供证书管理器，可退化为 CONNECT 隧道模式。
- 通过 `singleConnListener` 将单条 TLS 连接适配给 `http.Server`，复用标准库 HTTP 请求解析。

### 7.4 Relay 网关 `internal/relay`

职责：

- 统一读取请求体、判断路由模式、转发请求或交给 BYOK 网关。
- 识别 `x-raw-cursor-server-url` 头作为上游目标。
- 支持从 header、query、JSON body 中识别模型 ID。

路由模式：

| 模式 | 触发条件 | 处理方式 |
| --- | --- | --- |
| `routeDirect` | 默认路径 | 按目标地址直接转发 |
| `routeRelay` | 存在原始 Cursor 目标头，或 BYOK adapter 未匹配 | 通过本地 Relay 转发 |
| `selfImplemented` | 模型 ID 为 `byok/<model>` 且匹配启用适配器 | 交给 BYOK 网关 |

### 7.5 BYOK 模型网关

职责：

- 支持用户配置自己的模型渠道。
- 将 `byok/<model>` 请求转换为对应第三方 API 请求。
- 目前支持 OpenAI 兼容与 Anthropic 兼容接口。

适配器配置：

```go
type ModelAdapter struct {
    ID          string
    DisplayName string
    Type        AdapterType // openai / anthropic
    BaseURL     string
    APIKey      string
    ModelID     string
    Enabled     bool
}
```

OpenAI 路径：

- Endpoint：`{baseURL}/chat/completions`
- Header：`Authorization: Bearer <apiKey>`

Anthropic 路径：

- Endpoint：`{baseURL}/messages`
- Header：`x-api-key: <apiKey>`、`anthropic-version: 2023-06-01`

错误处理：

- 上游 429 映射为 `ErrByokChannelRateLimited`。
- 上游 5xx 或请求失败映射为 `ErrByokChannelNotAvailable`。
- 无效 JSON 请求映射为 `ErrInvalidBidiAppendPayload`。

### 7.6 证书管理 `internal/certs`

职责：

- 在应用数据目录下维护本地 CA 证书和私钥。
- 动态生成目标 host 的服务端证书。
- 输出 CA 证书信息和平台安装计划。

安全策略：

- CA 私钥权限为 `0600`。
- 证书目录权限为 `0700`。
- 系统证书安装不自动执行，只返回用户可确认的命令。

### 7.7 Cursor 集成 `internal/cursor`

职责：

- 识别 Cursor 用户设置文件路径。
- 生成代理设置写入计划。
- 显式写入 `http.proxy` 与 `http.proxyStrictSSL`。

平台路径：

| 平台 | 设置文件 |
| --- | --- |
| macOS | `~/Library/Application Support/Cursor/User/settings.json` |
| Windows | `%APPDATA%/Cursor/User/settings.json` |
| Linux fallback | `~/.config/Cursor/User/settings.json` |

### 7.8 配置管理 `internal/config`

职责：

- 管理用户配置文件 `config.json`。
- 校验 Base URL、Proxy URL、模型适配器字段。
- 保存配置时保留密钥掩码，避免前端的 `********` 覆盖真实 API Key 或 license。

默认配置：

- `baseURL`: `http://127.0.0.1:8080`
- `proxyURL`: `http://127.0.0.1:18080`
- `modelAdapters`: `[]`

### 7.9 日志与观测 `internal/observability`

职责：

- 在启用观测日志时写入 JSONL。
- `run-usage.jsonl` 记录代理启停、Relay 转发等运行事件。
- `channel-calls.jsonl` 记录 BYOK 渠道调用事件。

日志字段包含事件名、UTC 时间戳、状态码、耗时、适配器 ID、错误信息等。

## 8. 前端设计

前端位于 `frontend/`，使用 Vue 3 + Vite。当前控制台提供：

- 服务状态与代理状态展示。
- 运行配置编辑。
- 代理启动/停止。
- Cursor 设置写入。
- CA 证书信息与安装命令展示。
- BYOK 模型适配器新增、编辑、删除。
- 路由决策预览。

API 调用策略：

- 桌面模式：优先调用 `window.wails.Services` 暴露的 Go service。
- Web 开发模式：使用 HTTP API 调用 `127.0.0.1:8080`。

## 9. 请求处理流程

### 9.1 普通请求流程

```text
1. Cursor IDE 发起 HTTP/HTTPS 请求
2. Cursor 将请求发送到本地代理 127.0.0.1:18080
3. MITM Proxy 接收请求
   - HTTP：直接解析
   - HTTPS CONNECT：动态证书 + TLS MITM
4. Proxy 将解密后的 HTTP 请求交给 Relay Gateway
5. Relay Gateway 判断路由模式
6. 按路由转发到 Cursor Server 或 BYOK Gateway
7. 响应回写给 Cursor IDE
```

### 9.2 BYOK 请求流程

```text
1. 用户在配置中添加模型适配器
2. Cursor 请求中的模型 ID 使用 byok/<model>
3. Relay 从 header/query/body 识别模型 ID
4. Relay 匹配启用的 ModelAdapter
5. BYOK Gateway 转换请求为 OpenAI/Anthropic 兼容格式
6. 调用第三方模型 API
7. 原样流式或普通响应返回给 Cursor IDE
```

## 10. Protocol Buffer 设计

飞书原文定义项目将使用 Protocol Buffers 进行 Cursor AI 请求/响应编解码，主要协议文件包括：

- `aiserver_v1.proto`：AI 服务完整协议定义。
- `agent_v1_transport_pseudo.proto`：Agent 传输协议。
- `dashboard_usage_patch.proto`：使用量统计补丁。
- `from_extensions/`：从扩展提取的协议文件。

当前仓库尚未纳入完整 proto 文件与生成代码。本方案保留 `protoc`/`protoc-gen-go` 后续接入方向：

1. 将 proto 文件放入 `proto/`。
2. 生成 Go 结构体与编解码代码。
3. 在 `internal/relay` 中增加 Cursor Proto 与第三方模型协议之间的转换层。
4. 对 `run_sse`、`bidi_append` 等接口补充端到端测试。

## 11. 配置方案

用户配置存放在应用数据目录下：

- macOS：`~/Library/Application Support/Cursor助手/`
- Windows：`%APPDATA%/Cursor助手/`

配置结构：

```json
{
  "baseURL": "http://127.0.0.1:8080",
  "licenseCode": "",
  "proxyURL": "http://127.0.0.1:18080",
  "observabilityLogEnabled": false,
  "modelAdapters": [
    {
      "id": "openai-main",
      "displayName": "GPT-4o",
      "type": "openai",
      "baseURL": "https://api.openai.com/v1",
      "apiKey": "sk-xxx",
      "modelID": "gpt-4o",
      "enabled": true
    }
  ]
}
```

配置安全注意事项：

- 当前 API Key 以明文保存在本地配置文件。
- 文件权限限制为当前用户可读写。
- 后续可演进为 macOS Keychain / Windows Credential Manager 存储。

## 12. 错误处理方案

统一错误结构：

```json
{
  "code": "ErrInvalidSystemSetting",
  "message": "system setting is invalid",
  "status": 500
}
```

主要错误码：

| 错误码 | 说明 | HTTP 状态码 |
| --- | --- | --- |
| `ErrInvalidSystemSetting` | 系统配置无效 | 500 |
| `ErrCursorAccountUnavailable` | Cursor 账号不可用 | 503 |
| `ErrByokChannelRateLimited` | BYOK 渠道限流 | 429 |
| `ErrByokChannelNotAvailable` | BYOK 渠道不可用 | 503 |
| `ErrInvalidBidiAppendPayload` | 无效的双向流负载 | 400 |
| `ErrInvalidRequest` | 请求参数无效 | 400 |
| `ErrNotFound` | 资源不存在 | 404 |
| `ErrProxyNotRunning` | 代理未运行 | 503 |
| `ErrUpstream` | 上游请求失败 | 502 |

## 13. 安全方案

安全约束：

- 代理仅绑定 `127.0.0.1`，不接受外部连接。
- CA 安装和 Cursor 设置修改均要求用户显式触发。
- CA 私钥保存在本机应用数据目录，权限为 `0600`。
- API Key 当前为本地明文保存，应提示用户保护本机配置目录。
- 前端不会直接展示真实 API Key，后端快照中使用掩码。
- CORS 仅允许本地开发源 `127.0.0.1:5173` / `localhost:5173`。
- 生产分发需补充代码签名、macOS notarization 与 Windows 签名。

## 14. 构建与发布方案

开发要求：

- Go 1.25+
- Node.js 18+
- Wails v3 CLI
- npm

常用命令：

```bash
# 后端测试
GOTOOLCHAIN=go1.25.11 go test ./...

# 前端构建
cd frontend && npm run build

# HTTP fallback server
go run . --addr 127.0.0.1:8080 --proxy-addr 127.0.0.1:18080

# Windows amd64 包
./scripts/package_windows.sh amd64

# macOS 包
./scripts/doctor_packaging.sh
./scripts/package_darwin.sh
```

输出产物：

| 平台 | 产物 |
| --- | --- |
| macOS amd64 | `bin/darwin-amd64/Cursor助手.app` |
| macOS amd64 zip | `bin/CursorAssistant-darwin-amd64.app.zip` |
| macOS arm64 | `bin/darwin-arm64/Cursor助手.app` |
| macOS arm64 zip | `bin/CursorAssistant-darwin-arm64.app.zip` |
| Windows amd64 | `bin/windows-amd64/CursorAssistant.exe` |
| Windows amd64 zip | `bin/CursorAssistant-windows-amd64.zip` |

macOS 当前使用 ad-hoc signing，公开分发前需使用 Developer ID 签名并 notarize。

GitHub Actions 工作流位于 `.github/workflows/package.yml`，用于持续构建多平台发布包：

- `workflow_dispatch`：手动触发完整打包。
- `push` 到 `main` / `master`：验证并上传多平台构建产物。
- `pull_request` 到 `main` / `master`：只执行验证，不上传发布产物。
- `v*` 标签：构建三个 zip 产物并发布到 GitHub Release。

CI 打包矩阵：

| Target | Runner | Script | Artifact |
| --- | --- | --- | --- |
| `darwin-amd64` | `macos-15-intel` | `./scripts/package_darwin.sh amd64` | `bin/CursorAssistant-darwin-amd64.app.zip` |
| `darwin-arm64` | `macos-14` | `./scripts/package_darwin.sh arm64` | `bin/CursorAssistant-darwin-arm64.app.zip` |
| `windows-amd64` | `ubuntu-latest` | `./scripts/package_windows.sh amd64` | `bin/CursorAssistant-windows-amd64.zip` |

## 15. 可测试性与质量保障

当前测试覆盖：

- `internal/app`：健康检查、状态接口、证书接口、路由决策接口。
- `internal/config`：配置保存与密钥掩码保留。
- `internal/mitm`：代理状态基础测试。
- `internal/relay`：BYOK 路由、body 模型识别、目标 URL 拼接。

建议质量门禁：

```bash
GOTOOLCHAIN=go1.25.11 go test ./...
cd frontend && npm run build
./scripts/package_windows.sh amd64
./scripts/package_darwin.sh
./scripts/verify_release.sh
python3 scripts/verify_workflow.py
```

## 16. 扩展方案

### 16.1 新增模型适配器

1. 在 `internal/config` 扩展 `AdapterType`。
2. 在 `internal/relay/byok.go` 中扩展 `encodeAdapterRequest()` 和 `applyAuthHeaders()`。
3. 在前端适配器表单增加类型选项。
4. 增加适配器保存、路由决策和上游请求测试。

### 16.2 新增路由

1. 在 `internal/relay/types.go` 中扩展路由判断条件。
2. 在 `Gateway.ServeHTTP()` 中接入新的处理分支。
3. 为目标路径、请求头、请求体模型识别增加测试。
4. 在前端路由预览中暴露新路由结果。

### 16.3 接入完整 Proto 编解码

1. 引入 `proto/` 目录和代码生成任务。
2. 增加 `internal/protocodec` 或等价模块。
3. 对 Cursor 请求体做结构化解码。
4. 在 BYOK 转换层中实现 Cursor Proto 与 OpenAI/Anthropic payload 的双向转换。

## 17. 风险与后续工作

| 风险/待办 | 影响 | 建议 |
| --- | --- | --- |
| Cursor 私有协议未完整落地 | BYOK 与 Cursor 原生接口兼容深度有限 | 补齐 proto 文件、生成代码和端到端样例 |
| API Key 明文存储 | 本机配置泄露会暴露密钥 | 接入系统凭据存储 |
| CA 信任链操作敏感 | 用户误操作可能影响系统信任 | 继续保持显式安装与清晰提示 |
| macOS 未 notarize | 公开分发可能被 Gatekeeper 拦截 | Developer ID 签名与 notarization |
| Windows 未签名 | 用户运行时可能看到 SmartScreen 提示 | 增加代码签名证书 |
| 流式响应转换能力有限 | SSE/bidi 场景可能不完整 | 针对 `run_sse`、`bidi_append` 增加协议测试 |

## 18. 结论

本项目采用本地 MITM 代理 + Relay 网关 + BYOK 适配器的架构，为 Cursor IDE 提供本地可控的模型网关能力。Wails v3 将 Go 后端能力和 Vue 控制台整合为跨平台桌面应用，当前已经具备代理启停、配置管理、证书管理、Cursor 设置集成、BYOK 转发和 macOS/Windows 打包能力。

后续重点应放在 Cursor Protocol Buffer 协议完整接入、密钥安全存储、生产签名与更多模型渠道适配上。
