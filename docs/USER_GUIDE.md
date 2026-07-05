# Cursor助手配置与使用教程

本文面向最终使用者和本地开发调试者，说明如何安装、启动、配置模型渠道、写入 Cursor 代理设置、安装本地 CA 证书，以及如何排查常见问题。

## 1. 软件用途

Cursor助手是一个运行在本机的 Cursor IDE 辅助工具。它会在本机启动一个 HTTP/HTTPS 代理，让 Cursor 的请求先经过本地桥接服务，再按配置转发到原始 Cursor 服务或你自己的模型渠道。

核心能力：

- 本地代理默认监听 `127.0.0.1:18080`。
- 桥接 API 默认监听 `127.0.0.1:8080`。
- 支持 OpenAI Compatible 和 Anthropic Compatible 模型渠道。
- 支持导入 JSON、base64 JSON 或导入链接形式的第三方中转站配置。
- 可以一键写入 Cursor 的 `http.proxy` 和 `http.proxyStrictSSL` 设置。
- 生成本地 CA 证书，并给出系统信任证书的安装命令。
- 提供状态诊断、路由预览和 JSONL 观测日志。

## 2. 推荐使用流程

普通用户按下面顺序操作即可：

1. 启动 Cursor助手。
2. 在“一键导入”里粘贴中转站配置，或在“手动配置”里填写模型渠道。
3. 点击“导入并应用”或“保存并应用”。
4. 在“已配置”区域确认模型显示为 `byok/<模型ID>`。
5. 点击“应用到 Cursor”或“自动准备”，让软件启动本地桥接并写入 Cursor 代理设置。
6. 在“高级设置与诊断”中复制并安装本地 CA 证书。
7. 重启 Cursor。
8. 让 Cursor 发出的模型请求携带 `byok/<模型ID>`，对应请求会命中本地 BYOK 渠道。

完成后，状态区域应显示：

- 模型配置：至少 1 个已启用。
- 本地桥接：运行中。
- Cursor：已应用。
- 当前渠道：显示你配置的渠道名称。

## 3. 启动方式

### 3.1 使用打包后的桌面应用

macOS 包名格式：

```text
CursorAssistant-darwin-<arch>.app.zip
```

Windows 包名格式：

```text
CursorAssistant-windows-amd64.zip
```

macOS：

1. 解压 zip。
2. 打开 `Cursor助手.app`。
3. 如果系统提示来自未知开发者，需要在系统设置中允许打开。

Windows：

1. 解压 zip。
2. 运行 `CursorAssistant.exe`。
3. 确认系统已安装 WebView2 Runtime；大多数新版 Windows 已自带。

桌面应用会直接打开管理界面，不需要手动启动前端开发服务器。

### 3.2 从源码运行

要求：

- Go 1.25+
- Node.js 18+
- npm

启动后端桥接 API：

```bash
go run . --addr 127.0.0.1:8080 --proxy-addr 127.0.0.1:18080
```

另开一个终端启动前端：

```bash
cd frontend
npm install
npm run dev
```

打开 Vite 显示的地址，通常是：

```text
http://127.0.0.1:5173
```

开发模式下，前端会把 `/api` 和 `/health` 请求代理到 `http://127.0.0.1:8080`。

如果使用 Task CLI，也可以执行：

```bash
task dev
task dev:frontend
```

## 4. 配置文件与数据目录

软件会把配置保存到用户配置目录下的 `Cursor助手/config.json`。

常见位置：

- macOS：`~/Library/Application Support/Cursor助手/config.json`
- Windows：通常在 `%APPDATA%/Cursor助手/config.json`
- Linux：通常在 `~/.config/Cursor助手/config.json`

界面底部会显示当前数据目录，“高级设置与诊断”里的“运行参数”区域会显示完整配置路径。

配置结构大致如下：

```json
{
  "baseURL": "http://127.0.0.1:8080",
  "licenseCode": "",
  "proxyURL": "http://127.0.0.1:18080",
  "observabilityLogEnabled": false,
  "modelAdapters": [
    {
      "id": "example-gpt-4o",
      "displayName": "Example Relay gpt-4o",
      "type": "openai",
      "baseURL": "https://relay.example.com/v1",
      "apiKey": "sk-xxx",
      "modelID": "gpt-4o",
      "enabled": true
    }
  ]
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `baseURL` | 本地桥接 API 地址，默认 `http://127.0.0.1:8080` |
| `proxyURL` | 写入 Cursor 的本地代理地址，默认 `http://127.0.0.1:18080` |
| `observabilityLogEnabled` | 是否启用 JSONL 观测日志 |
| `modelAdapters` | 模型渠道列表 |
| `modelAdapters[].type` | `openai` 或 `anthropic` |
| `modelAdapters[].baseURL` | 第三方模型或中转站的 API Base URL |
| `modelAdapters[].apiKey` | 渠道 API Key |
| `modelAdapters[].modelID` | 上游真实模型 ID，界面和请求中会显示为 `byok/<modelID>` |
| `modelAdapters[].enabled` | 是否启用该渠道 |

保存配置时，前端看到的密钥会被显示为 `********`。如果编辑已有渠道但不填写新的 API Key，后端会保留原密钥。

## 5. 配置模型渠道

### 5.1 一键导入中转站配置

进入首页的“一键导入”区域，把中转站配置粘贴到“配置内容”。

支持的输入包括：

- JSON 文档。
- base64 或 base64url 编码后的 JSON。
- 带 query 参数的导入链接。
- query、path、fragment 中包含 JSON payload 的导入链接。

最简单的 JSON 示例：

```json
{
  "name": "Example Relay",
  "type": "openai",
  "baseURL": "https://relay.example.com/v1",
  "apiKey": "sk-xxx",
  "models": ["gpt-4o", "gpt-4.1"]
}
```

导入链接示例：

```text
cursorbridge://import?name=Example%20Relay&baseURL=https%3A%2F%2Frelay.example.com%2Fv1&apiKey=sk-xxx&models=gpt-4o%2Cgpt-4.1
```

操作步骤：

1. 粘贴配置。
2. 点击“预览”。
3. 检查预览中的渠道名称、类型、Base URL 和 `byok/<模型ID>`。
4. 点击“导入并应用”。

导入时支持多个模型。如果 `models` 中有多个模型，软件会为每个模型生成一个适配器。

### 5.2 手动配置 OpenAI Compatible 渠道

在“手动配置”区域填写：

| 表单项 | 示例 |
| --- | --- |
| 渠道名称 | `我的中转站` |
| 类型 | `OpenAI Compatible` |
| Base URL | `https://api.openai.com/v1` 或中转站地址 |
| API Key | `sk-...` |
| 模型 ID | `gpt-4o` |
| 配置 ID | 可手填，也可使用占位建议 |
| 启用这个模型渠道 | 勾选 |

点击“保存并应用”。

OpenAI Compatible 请求会转发到：

```text
<Base URL>/chat/completions
```

并使用：

```text
Authorization: Bearer <API Key>
```

如果你填写的 Base URL 已经带有 `/chat/completions`，后端会在保存时自动规整为基础路径，避免重复拼接。

### 5.3 手动配置 Anthropic Compatible 渠道

在“类型”中选择 `Anthropic Compatible`。

常见填写：

| 表单项 | 示例 |
| --- | --- |
| Base URL | `https://api.anthropic.com/v1` 或兼容中转站地址 |
| API Key | 你的 Anthropic 或中转站 key |
| 模型 ID | `claude-3-5-sonnet-latest` |

Anthropic Compatible 请求会转发到：

```text
<Base URL>/messages
```

并使用：

```text
x-api-key: <API Key>
anthropic-version: 2023-06-01
```

如果你填写的 Base URL 已经带有 `/messages`，后端会自动规整为基础路径。

## 6. 应用到 Cursor

模型保存后，点击：

```text
应用到 Cursor
```

或在“下一步”区域点击：

```text
自动准备
```

这个动作会尝试完成两件事：

1. 启动本地代理。
2. 写入 Cursor 设置。

Cursor 设置会被写入下面的位置：

| 平台 | Cursor 设置文件 |
| --- | --- |
| macOS | `~/Library/Application Support/Cursor/User/settings.json` |
| Windows | `%APPDATA%/Cursor/User/settings.json` |
| Linux | `~/.config/Cursor/User/settings.json` |

写入内容包括：

```json
{
  "http.proxy": "http://127.0.0.1:18080",
  "http.proxyStrictSSL": false
}
```

写入后建议重启 Cursor，让代理设置完整生效。

## 7. 安装本地 CA 证书

Cursor助手会在数据目录下生成本地 CA 证书，用于 HTTPS MITM 代理。证书文件通常位于：

```text
<数据目录>/certs/cursor-assistant-ca.pem
```

在“高级设置与诊断”里的“本地 CA”区域可以看到：

- 证书文件路径。
- 证书指纹。
- 有效期。
- 当前平台的安装命令。

macOS 安装命令格式：

```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "<证书路径>"
```

Windows 安装命令格式：

```powershell
certutil -addstore -f Root "<证书路径>"
```

安装 CA 是系统级操作，软件不会在启动时自动安装。你需要确认命令和证书路径后手动执行。

安装完成后，重启 Cursor。

## 8. 让 Cursor 请求命中 BYOK 模型

保存渠道后，软件会把每个模型显示为：

```text
byok/<模型ID>
```

例如你配置的模型 ID 是：

```text
gpt-4o
```

那么本地路由使用的模型名是：

```text
byok/gpt-4o
```

当 Cursor 发出的请求模型名是 `byok/gpt-4o`，并且本地配置中存在启用的 `gpt-4o` 适配器时，请求会进入 BYOK 网关，转发到你配置的上游模型渠道。

如果请求没有匹配到启用的 BYOK 适配器，软件会按路由决策继续转发到原始目标或 Relay 目标。

注意：当前软件负责启动本地代理、写入 Cursor 代理设置、识别并转发 `byok/<模型ID>` 请求；它不会自动修改 Cursor 的模型列表。如果 Cursor 侧没有发出 `byok/<模型ID>` 这个模型名，请求就不会进入对应 BYOK 渠道。可以用“路由预览”先确认某个模型名是否能命中本地适配器。

## 9. 高级设置与诊断

点击首页底部的：

```text
高级设置与诊断
```

可以看到以下区域。

### 9.1 本地桥接

显示：

- 代理监听地址。
- Cursor 设置文件路径。
- Cursor 当前代理。

可执行操作：

- “启动桥接”或“停止桥接”。
- “写入 Cursor 设置”。
- “复制路径”。

如果要修改 `Proxy URL`，需要先停止桥接。后端会拒绝在代理运行中修改监听地址。

### 9.2 运行参数

可配置：

- `Bridge Base URL`
- `Proxy URL`
- 是否启用 JSONL 观测日志

`Proxy URL` 必须满足：

- 使用 `http` scheme。
- 包含 host 和 port。
- host 必须是 `localhost` 或回环地址。
- 不能包含用户名、密码、路径、query 或 fragment。

合法示例：

```text
http://127.0.0.1:18080
http://localhost:18080
```

不合法示例：

```text
https://127.0.0.1:18080
http://0.0.0.0:18080
http://127.0.0.1:18080/proxy
```

### 9.3 本地 CA

显示本地 CA 信息和安装命令。需要 HTTPS 代理正常工作时，应安装并信任该证书。

### 9.4 路由预览

输入：

- 模型，例如 `byok/gpt-4o`
- 路径，例如 `/v1/chat/completions`
- Raw Cursor Target，可选

点击“预览”后，会显示当前请求将如何路由。这个功能适合排查某个模型是否会命中 BYOK 渠道。

### 9.5 系统状态

显示诊断项：

- 桥接服务。
- 本地代理。
- 本地 CA。
- Cursor 设置。
- BYOK 适配器。

如果某项不是健康状态，界面的“下一步”区域会给出建议操作。

## 10. 日志

默认不会写入观测日志。需要时，在“高级设置与诊断”里的“运行参数”勾选：

```text
启用 JSONL 观测日志
```

日志路径会显示在界面中：

```text
<数据目录>/run-usage.jsonl
<数据目录>/channel-calls.jsonl
```

含义：

- `run-usage.jsonl`：记录代理启动、停止、转发等运行事件。
- `channel-calls.jsonl`：记录 BYOK 渠道调用结果、耗时、错误等事件。

这些日志适合排查“代理是否被调用”“请求是否进入 BYOK”“上游是否返回错误”。

## 11. 常见问题

### 11.1 界面显示“模型配置：未配置”

原因：

- 没有导入或保存模型渠道。
- 模型渠道被停用。
- 导入内容缺少 `baseURL`、`apiKey` 或 `model/models`。

处理：

1. 在“一键导入”点击“预览”，确认能解析出模型。
2. 或在“手动配置”填写完整字段。
3. 确认勾选“启用这个模型渠道”。
4. 点击“导入并应用”或“保存并应用”。

### 11.2 本地桥接未启动

处理：

1. 点击“自动准备”。
2. 或展开“高级设置与诊断”，点击“启动桥接”。
3. 如果端口被占用，修改 `Proxy URL` 的端口，或停止占用 `18080` 的其他程序。

### 11.3 Cursor 显示“待应用”

原因：

- Cursor 的 `settings.json` 还没有写入代理配置。
- Cursor 设置文件路径异常。
- 写入后 Cursor 尚未重启。

处理：

1. 点击“应用到 Cursor”或“写入 Cursor 设置”。
2. 检查界面显示的 Cursor 设置路径。
3. 重启 Cursor。

### 11.4 HTTPS 请求失败或证书错误

原因：

- 本地 CA 尚未被系统信任。
- Cursor 尚未重启。
- 系统证书安装到了错误的证书存储。

处理：

1. 展开“高级设置与诊断”。
2. 在“本地 CA”区域复制安装命令。
3. 手动执行命令并确认安装。
4. 重启 Cursor。

### 11.5 BYOK 模型没有命中

检查：

- Cursor 请求里的模型名是否是 `byok/<模型ID>`。
- 配置里的模型渠道是否启用。
- `模型 ID` 是否和 `byok/` 后面的部分完全一致。

可以用“路由预览”输入模型名，例如：

```text
byok/gpt-4o
```

如果没有命中，编辑对应渠道或重新导入。

### 11.6 上游返回 429 或 5xx

含义：

- 429：上游模型渠道限流。
- 5xx：上游模型渠道不可用或中转站异常。

处理：

1. 检查 API Key 是否有效。
2. 检查 Base URL 是否正确。
3. 换用其他模型渠道。
4. 开启 JSONL 观测日志，查看 `channel-calls.jsonl`。

## 12. 开发者常用命令

后端测试：

```bash
GOTOOLCHAIN=go1.25.11 go test ./...
```

后端构建：

```bash
GOTOOLCHAIN=go1.25.11 go build ./...
```

前端构建：

```bash
cd frontend
npm run build
```

完整校验：

```bash
task verify
```

打包前检查：

```bash
./scripts/doctor_packaging.sh
```

macOS 打包：

```bash
./scripts/package_darwin.sh
```

Windows 打包：

```bash
./scripts/package_windows.sh amd64
```

更多打包细节见 `docs/PACKAGING.md`。

## 13. 安全提示

- 本地代理默认只监听回环地址，不应改成公网可访问地址。
- API Key 保存在本机配置文件中，请保护好用户配置目录权限。
- 本地 CA 私钥位于数据目录的 `certs` 子目录，泄露后会影响 HTTPS 拦截安全，应妥善保管。
- 安装 CA 证书前，请确认路径来自 Cursor助手的数据目录。
- Cursor 设置和系统信任证书都属于显式操作，应用不会在启动时静默修改。
