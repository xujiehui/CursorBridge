---
source_url: https://dcne38qm5vlg.feishu.cn/wiki/K2YHwSbAjilCZ6k3ywQcHnxFn7e
captured_at: 2026-07-03T16:22:31.892Z
title: Cursor助手 - 项目技术文档 - Feishu Docs
final_url: https://dcne38qm5vlg.feishu.cn/wiki/K2YHwSbAjilCZ6k3ywQcHnxFn7e
mode: virtual-scroll snapshots
---

# Cursor助手 - 项目技术文档（飞书滚动快照）

> 飞书页面使用虚拟滚动。此文件按滚动顺序保留每次视口可见正文，作为本地忠实副本；相邻快照可能有重叠。

## Snapshot 0

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
Cursor助手 - 项目技术文档
非常感谢 @白帽酱 大佬的协议研究  https://rce.moe/  https://rce.moe/2026/01/31/cursor-reverse-notes-1/
助手作者后续在大佬的研究之上，继续深入研究了数月才小有成果。
cursor提示词与工具调用.zip
82.23KB
项目概述
Cursor助手 是一个基于 Wails v3 构建的跨平台桌面应用程序，为 Cursor IDE 提供本地代理服务。它的核心功能是通过 MITM（中间人代理）拦截 Cursor 的 API 请求，并支持 BYOK（Bring Your Own Key）自定义API模式，允许用户使用自己的 AI 模型 API 密钥。
技术栈
层级
技术选型
后端
Go 1.25 + Wails v3
前端
Vue 3 + Vite 7 + Tailwind CSS 4
代理
goproxy + 自定义 TLS 证书管理
协议
Protocol Buffers (Connect)
数据库
SQLite (modernc.org/sqlite)
```

## Snapshot 1

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
项目概述
Cursor助手 是一个基于 Wails v3 构建的跨平台桌面应用程序，为 Cursor IDE 提供本地代理服务。它的核心功能是通过 MITM（中间人代理）拦截 Cursor 的 API 请求，并支持 BYOK（Bring Your Own Key）自定义API模式，允许用户使用自己的 AI 模型 API 密钥。
技术栈
层级
技术选型
后端
Go 1.25 + Wails v3
前端
Vue 3 + Vite 7 + Tailwind CSS 4
代理
goproxy + 自定义 TLS 证书管理
协议
Protocol Buffers (Connect)
数据库
SQLite (modernc.org/sqlite)
架构设计
┌─────────────────────────────────────────────────────────────────┐
│                        Cursor IDE                               │
└─────────────────────────┬───────────────────────────────────────┘
│ HTTP/HTTPS 请求
▼
┌─────────────────────────────────────────────────────────────────┐
│                    MITM Proxy Server                            │
│                  (127.0.0.1:18080)                              │
└─────────────────────────┬───────────────────────────────────────┘
│
┌───────────────┼───────────────┐
│               │               │
▼               ▼               ▼
┌────────────┐  ┌────────────┐  ┌────────────┐
│  Direct    │  │   Relay    │  │  Self-Impl │
│  Forward   │  │  Gateway   │  │   Model    │
└────────────┘  └────────────┘  └────────────┘
│               │               │
▼               ▼               ▼
┌────────────┐  ┌────────────┐  ┌────────────┐
│  Cursor    │  │  Cursor    │  │  OpenAI /  │
│  Server    │  │  Server    │  │  Anthropic │
└────────────┘  └────────────┘  └────────────┘
```

## Snapshot 2

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
┌─────────────────────────────────────────────────────────────────┐
│                        Cursor IDE                               │
└─────────────────────────┬───────────────────────────────────────┘
│ HTTP/HTTPS 请求
▼
┌─────────────────────────────────────────────────────────────────┐
│                    MITM Proxy Server                            │
│                  (127.0.0.1:18080)                              │
└─────────────────────────┬───────────────────────────────────────┘
│
┌───────────────┼───────────────┐
│               │               │
▼               ▼               ▼
┌────────────┐  ┌────────────┐  ┌────────────┐
│  Direct    │  │   Relay    │  │  Self-Impl │
│  Forward   │  │  Gateway   │  │   Model    │
└────────────┘  └────────────┘  └────────────┘
│               │               │
▼               ▼               ▼
┌────────────┐  ┌────────────┐  ┌────────────┐
│  Cursor    │  │  Cursor    │  │  OpenAI /  │
│  Server    │  │  Server    │  │  Anthropic │
└────────────┘  └────────────┘  └────────────┘
目录结构
cursor-client/
├── main.go                      # 应用入口
├── go.mod                       # Go 模块定义
├── Taskfile.yml                 # 构建任务配置
├── build/                       # 构建配置
│   ├── config.yml              # Wails 开发配置
│   ├── darwin/                 # macOS 构建脚本
│   └── windows/                # Windows 构建脚本
├── frontend/                    # Vue.js 前端
│   ├── src/                    # 前端源码
│   ├── bindings/               # Wails 自动生成的绑定
│   └── package.json            # 前端依赖
├── internal/                    # 内部包
│   ├── app/                    # 应用运行器
│   ├── bridge/                 # Wails 服务桥接
│   ├── certs/                  # TLS 证书管理
│   ├── clientruntime/          # 客户端运行时服务
│   ├── cursor/                 # Cursor IDE 集成
│   ├── mitm/                   # MITM 代理服务
│   ├── protocodec/             # Protocol Buffer 编解码
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
负责拦截 Cursor IDE 的 HTTPS 请求。
```

## Snapshot 3

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
目录结构
cursor-client/
├── main.go                      # 应用入口
├── go.mod                       # Go 模块定义
├── Taskfile.yml                 # 构建任务配置
├── build/                       # 构建配置
│   ├── config.yml              # Wails 开发配置
│   ├── darwin/                 # macOS 构建脚本
│   └── windows/                # Windows 构建脚本
├── frontend/                    # Vue.js 前端
│   ├── src/                    # 前端源码
│   ├── bindings/               # Wails 自动生成的绑定
│   └── package.json            # 前端依赖
├── internal/                    # 内部包
│   ├── app/                    # 应用运行器
│   ├── bridge/                 # Wails 服务桥接
│   ├── certs/                  # TLS 证书管理
│   ├── clientruntime/          # 客户端运行时服务
│   ├── cursor/                 # Cursor IDE 集成
│   ├── mitm/                   # MITM 代理服务
│   ├── protocodec/             # Protocol Buffer 编解码
│   ├── relay/                  # Relay 网关核心
│   └── runtime/                # 本地运行时依赖
└── proto/                       # Protocol Buffer 定义
├── aiserver_v1.proto       # AI 服务协议
├── agent_v1_transport_pseudo.proto
└── from_extensions/        # 从扩展提取的协议
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
负责拦截 Cursor IDE 的 HTTPS 请求。
核心组件:
•
ProxyServer: 主代理服务器，监听 127.0.0.1:18080
•
支持动态证书生成
•
可配置上游服务器 URL
关键功能:
•
HTTPS 流量拦截
•
请求头注入 (x-raw-cursor-server-url, x-raw-cursor-user-key, x-raw-cursor-device-id)
•
透明代理转发
2. Relay 网关 (`internal/relay`)
```

## Snapshot 4

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
└── from_extensions/        # 从扩展提取的协议
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
负责拦截 Cursor IDE 的 HTTPS 请求。
核心组件:
•
ProxyServer: 主代理服务器，监听 127.0.0.1:18080
•
支持动态证书生成
•
可配置上游服务器 URL
关键功能:
•
HTTPS 流量拦截
•
请求头注入 (x-raw-cursor-server-url, x-raw-cursor-user-key, x-raw-cursor-device-id)
•
透明代理转发
2. Relay 网关 (`internal/relay`)
核心请求处理引擎，负责路由和转换 Cursor API 请求。
路由模式:
模式
说明
routeDirect
直接转发到 Cursor 服务器
routeRelay
通过本地 Relay 处理
selfImplemented
使用自有 API Key 调用模型
关键文件:
•
gateway_handler.go: HTTP 请求入口
•
gateway_forward.go: 上游转发逻辑
•
self_implemented_*.go: BYOK 模型实现
•
proto.go: Protocol Buffer 解析
支持的 AI 接口:
•
run_sse: Server-Sent Events 流式响应
•
bidi_append: 双向流式追加
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
```

## Snapshot 5

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
核心请求处理引擎，负责路由和转换 Cursor API 请求。
路由模式:
模式
说明
routeDirect
直接转发到 Cursor 服务器
routeRelay
通过本地 Relay 处理
selfImplemented
使用自有 API Key 调用模型
关键文件:
•
gateway_handler.go: HTTP 请求入口
•
gateway_forward.go: 上游转发逻辑
•
self_implemented_*.go: BYOK 模型实现
•
proto.go: Protocol Buffer 解析
支持的 AI 接口:
•
run_sse: Server-Sent Events 流式响应
•
bidi_append: 双向流式追加
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
允许用户配置自己的 OpenAI 或 Anthropic API Key。
支持的平台:
•
OpenAI (GPT-4, GPT-4o 等)
•
Anthropic (Claude 系列)
配置结构:
type ModelAdapterConfig struct {
DisplayName string  // 显示名称
Type        string  // "openai" 或 "anthropic"
BaseURL     string  // API 端点
APIKey      string  // API 密钥
ModelID     string  // 模型标识
}
4. 证书管理 (`internal/certs`)
管理 MITM 所需的 CA 证书。
功能:
•
内置 CA 证书（编译时嵌入）
•
动态生成服务器证书
•
支持 macOS/Windows 系统证书安装
5. Cursor 集成 (`internal/cursor`)
与 Cursor IDE 宿主环境交互。
功能:
•
读取/写入 Cursor 设置
•
设备 ID 管理
•
系统代理设置
•
状态数据库访问
6. 桥接服务 (`internal/bridge`)
```

## Snapshot 6

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
支持的平台:
•
OpenAI (GPT-4, GPT-4o 等)
•
Anthropic (Claude 系列)
配置结构:
type ModelAdapterConfig struct {
DisplayName string  // 显示名称
Type        string  // "openai" 或 "anthropic"
BaseURL     string  // API 端点
APIKey      string  // API 密钥
ModelID     string  // 模型标识
}
4. 证书管理 (`internal/certs`)
管理 MITM 所需的 CA 证书。
功能:
•
内置 CA 证书（编译时嵌入）
•
动态生成服务器证书
•
支持 macOS/Windows 系统证书安装
5. Cursor 集成 (`internal/cursor`)
与 Cursor IDE 宿主环境交互。
功能:
•
读取/写入 Cursor 设置
•
设备 ID 管理
•
系统代理设置
•
状态数据库访问
6. 桥接服务 (`internal/bridge`)
向 Wails 前端暴露 Go 服务。
暴露的服务:
•
ProxyService: 代理控制
•
WindowService: 窗口管理
前端可调用方法:
// 启动代理
StartProxy(): Promise<ProxyState>
数据流详解
```

## Snapshot 7

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
与 Cursor IDE 宿主环境交互。
功能:
•
读取/写入 Cursor 设置
•
设备 ID 管理
•
系统代理设置
•
状态数据库访问
6. 桥接服务 (`internal/bridge`)
向 Wails 前端暴露 Go 服务。
暴露的服务:
•
ProxyService: 代理控制
•
WindowService: 窗口管理
前端可调用方法:
// 启动代理
StartProxy(): Promise<ProxyState>
// 停止代理
StopProxy(): Promise<ProxyState>
// 获取状态
GetState(): ProxyState
// 设置上游 URL
SetBaseURL(url: string): Promise<ProxyState>
// 加载用户配置
LoadUserConfig(): Promise<UserConfig>
// 保存用户配置
SaveUserConfig(cfg: UserConfig): Promise<void>
// 激活许可证
ActivateLicense(req: LicenseActionRequest): Promise<LicenseAPIResult>
数据流详解
请求处理流程
1. Cursor IDE 发起请求
│
▼
2. MITM Proxy 拦截
│
├── 检查目标 URL
BYOK 模式流程
```

## Snapshot 8

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
SetBaseURL(url: string): Promise<ProxyState>
// 加载用户配置
LoadUserConfig(): Promise<UserConfig>
// 保存用户配置
SaveUserConfig(cfg: UserConfig): Promise<void>
// 激活许可证
ActivateLicense(req: LicenseActionRequest): Promise<LicenseAPIResult>
数据流详解
请求处理流程
1. Cursor IDE 发起请求
│
▼
2. MITM Proxy 拦截
│
├── 检查目标 URL
│
├── 提取自定义头 (x-raw-cursor-server-url)
│
▼
3. Relay Gateway 路由判断
│
├── 匹配路由规则
│
├── 判断处理模式
│   │
│   ├── 直接转发 → Cursor Server
│   │
│   ├── Relay 转发 → Cursor Server (带认证)
│   │
│   └── 自实现模型 → OpenAI/Anthropic API
│
▼
4. 响应返回给 Cursor IDE
BYOK 模式流程
1. 用户配置模型适配器 (BaseURL + APIKey)
│
▼
2. 请求到达 Relay Gateway
│
├── 检测模型 ID 前缀 "byok/"
│
```

## Snapshot 9

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
├── 匹配路由规则
│
├── 判断处理模式
│   │
│   ├── 直接转发 → Cursor Server
│   │
│   ├── Relay 转发 → Cursor Server (带认证)
│   │
│   └── 自实现模型 → OpenAI/Anthropic API
│
▼
4. 响应返回给 Cursor IDE
BYOK 模式流程
1. 用户配置模型适配器 (BaseURL + APIKey)
│
▼
2. 请求到达 Relay Gateway
│
├── 检测模型 ID 前缀 "byok/"
│
├── 匹配模型适配器配置
│
▼
3. 转换请求格式
│
├── Cursor Proto → OpenAI/Anthropic 格式
│
▼
4. 调用第三方 API
│
▼
5. 转换响应格式
│
├── OpenAI/Anthropic → Cursor Proto 格式
│
▼
6. 流式返回给 Cursor IDE
Protocol Buffer 协议
项目使用 Protocol Buffers 进行 AI 请求/响应的编解码。
主要协议文件:
文件
用途
aiserver_v1.proto
AI 服务完整协议定义
agent_v1_transport_pseudo.proto
Agent 传输协议
dashboard_usage_patch.proto
使用量统计补丁
```

## Snapshot 10

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
│
▼
4. 调用第三方 API
│
▼
5. 转换响应格式
│
├── OpenAI/Anthropic → Cursor Proto 格式
│
▼
6. 流式返回给 Cursor IDE
Protocol Buffer 协议
项目使用 Protocol Buffers 进行 AI 请求/响应的编解码。
主要协议文件:
文件
用途
aiserver_v1.proto
AI 服务完整协议定义
agent_v1_transport_pseudo.proto
Agent 传输协议
dashboard_usage_patch.proto
使用量统计补丁
运行时加载顺序:
1.
from_extensions/aiserver_v1.proto
2.
from_extensions/agent_v1.proto
3.
dashboard_usage_patch.proto
构建与开发
开发环境要求
•
Go 1.25+
•
Node.js 18+
•
Wails v3 CLI
常用命令
输出产物
```

## Snapshot 11

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
项目概述
技术栈
架构设计
目录结构
核心模块详解
1. MITM 代理服务 (`internal/mitm`)
2. Relay 网关 (`internal/relay`)
3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)
4. 证书管理 (`internal/certs`)
5. Cursor 集成 (`internal/cursor`)
6. 桥接服务 (`internal/bridge`)
数据流详解
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
dashboard_usage_patch.proto
使用量统计补丁
运行时加载顺序:
1.
from_extensions/aiserver_v1.proto
2.
from_extensions/agent_v1.proto
3.
dashboard_usage_patch.proto
构建与开发
开发环境要求
•
Go 1.25+
•
Node.js 18+
•
Wails v3 CLI
常用命令
# 启动开发模式
task dev
# 后台启动开发模式
task dev:bg
# 停止开发模式
task dev:stop
# 构建 macOS arm64
task build:darwin:arm64
# 构建 Windows amd64
task build:windows:amd64
# 打包发布版本
task package:release
输出产物
平台
格式
输出路径
macOS arm64
.app / .dmg
bin/Cursor助手-darwin-arm64.dmg
macOS amd64
.app / .dmg
bin/Cursor助手-darwin-amd64.dmg
Windows amd64
.exe / .zip
bin/Cursor助手-windows-amd64.zip
```

## Snapshot 12

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
许可证
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
# 后台启动开发模式
task dev:bg
# 停止开发模式
task dev:stop
# 构建 macOS arm64
task build:darwin:arm64
# 构建 Windows amd64
task build:windows:amd64
# 打包发布版本
task package:release
输出产物
平台
格式
输出路径
macOS arm64
.app / .dmg
bin/Cursor助手-darwin-arm64.dmg
macOS amd64
.app / .dmg
bin/Cursor助手-darwin-amd64.dmg
Windows amd64
.exe / .zip
bin/Cursor助手-windows-amd64.zip
配置说明
用户配置文件
配置文件存储在用户目录：
•
macOS: ~/Library/Application Support/Cursor助手/
•
Windows: %APPDATA%/Cursor助手/
**配置结构 (`UserConfig`):**
{
"baseURL": "http://127.0.0.1:8080",
"licenseCode": "",
"modelAdapters": [
{
"displayName": "GPT-4o",
"type": "openai",
"baseURL": "https://api.openai.com/v1",
代理设置
应用会自动配置 Cursor IDE 的代理设置：
•
代理地址: http://127.0.0.1:18080
•
CA 证书: 自动安装到系统信任存储
错误处理
常见错误码
```

## Snapshot 13

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
许可证
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
配置说明
用户配置文件
配置文件存储在用户目录：
•
macOS: ~/Library/Application Support/Cursor助手/
•
Windows: %APPDATA%/Cursor助手/
**配置结构 (`UserConfig`):**
{
"baseURL": "http://127.0.0.1:8080",
"licenseCode": "",
"modelAdapters": [
{
"displayName": "GPT-4o",
"type": "openai",
"baseURL": "https://api.openai.com/v1",
"apiKey": "sk-xxx",
"modelID": "gpt-4o"
}
]
}
代理设置
应用会自动配置 Cursor IDE 的代理设置：
•
代理地址: http://127.0.0.1:18080
•
CA 证书: 自动安装到系统信任存储
错误处理
常见错误码
错误
说明
HTTP 状态码
ErrInvalidSystemSetting
系统配置无效
500
ErrCursorAccountUnavailable
Cursor 账号不可用
503
ErrByokChannelRateLimited
BYOK 渠道限流
429
ErrByokChannelNotAvailable
BYOK 渠道不可用
503
ErrInvalidBidiAppendPayload
无效的双向流负载
400
```

## Snapshot 14

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
许可证
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
代理设置
应用会自动配置 Cursor IDE 的代理设置：
•
代理地址: http://127.0.0.1:18080
•
CA 证书: 自动安装到系统信任存储
错误处理
常见错误码
错误
说明
HTTP 状态码
ErrInvalidSystemSetting
系统配置无效
500
ErrCursorAccountUnavailable
Cursor 账号不可用
503
ErrByokChannelRateLimited
BYOK 渠道限流
429
ErrByokChannelNotAvailable
BYOK 渠道不可用
503
ErrInvalidBidiAppendPayload
无效的双向流负载
400
日志与观测
日志文件
日志存储在应用数据目录：
•
run-usage.jsonl: 运行使用记录
•
channel-calls.jsonl: 渠道调用记录
调试模式
通过配置启用详细日志：
RuntimeConfigSnapshot{
ObservabilityLogEnabled: true,
}
安全考虑
1. CA 证书: 使用内置 CA 证书，每次编译时嵌入
2. API Key 存储: 以明文形式存储在本地配置文件，用户需自行保护
3. 代理监听: 仅绑定 `127.0.0.1`，不接受外部连接
4. 单实例模式: 应用通过 `com.cursor-assistant.single-instance` 保证单实例运行
```

## Snapshot 15

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
许可证
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
日志与观测
日志文件
日志存储在应用数据目录：
•
run-usage.jsonl: 运行使用记录
•
channel-calls.jsonl: 渠道调用记录
调试模式
通过配置启用详细日志：
RuntimeConfigSnapshot{
ObservabilityLogEnabled: true,
}
安全考虑
1. CA 证书: 使用内置 CA 证书，每次编译时嵌入
2. API Key 存储: 以明文形式存储在本地配置文件，用户需自行保护
3. 代理监听: 仅绑定 `127.0.0.1`，不接受外部连接
4. 单实例模式: 应用通过 `com.cursor-assistant.single-instance` 保证单实例运行
依赖说明
主要 Go 依赖
依赖
用途
github.com/wailsapp/wails/v3
桌面应用框架
github.com/elazarl/goproxy
HTTP 代理实现
google.golang.org/protobuf
Protocol Buffers
modernc.org/sqlite
纯 Go SQLite
github.com/google/uuid
UUID 生成
```

## Snapshot 16

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
许可证
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
安全考虑
1. CA 证书: 使用内置 CA 证书，每次编译时嵌入
2. API Key 存储: 以明文形式存储在本地配置文件，用户需自行保护
3. 代理监听: 仅绑定 `127.0.0.1`，不接受外部连接
4. 单实例模式: 应用通过 `com.cursor-assistant.single-instance` 保证单实例运行
依赖说明
主要 Go 依赖
依赖
用途
github.com/wailsapp/wails/v3
桌面应用框架
github.com/elazarl/goproxy
HTTP 代理实现
google.golang.org/protobuf
Protocol Buffers
modernc.org/sqlite
纯 Go SQLite
github.com/google/uuid
UUID 生成
主要前端依赖
依赖
用途
vue
前端框架
vue-router
路由管理
tailwindcss
CSS 框架
@wailsio/runtime
Wails 运行时
扩展与定制
添加新的模型适配器
```

## Snapshot 17

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
许可证
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
modernc.org/sqlite
纯 Go SQLite
github.com/google/uuid
UUID 生成
主要前端依赖
依赖
用途
vue
前端框架
vue-router
路由管理
tailwindcss
CSS 框架
@wailsio/runtime
Wails 运行时
扩展与定制
添加新的模型适配器
1.
在 internal/relay/self_implemented_*.go 中实现模型接口
2.
在 ModelAdapterConfig 中添加新类型支持
3.
更新 normalizeModelAdapterType() 函数
添加新的路由
1.
在 internal/runtime/local_runtime.go 中定义路由
2.
在 gateway_handler.go 的 handleRoute() 中添加处理逻辑
版本历史
v0.0.1 (当前版本)
•
初始版本
•
支持 macOS (arm64/amd64) 和 Windows (amd64)
•
实现 BYOK 模式
•
支持 OpenAI 和 Anthropic 模型
许可证
© 2026, Cursor助手
```

## Snapshot 18

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
许可证
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
1.
在 internal/relay/self_implemented_*.go 中实现模型接口
2.
在 ModelAdapterConfig 中添加新类型支持
3.
更新 normalizeModelAdapterType() 函数
添加新的路由
1.
在 internal/runtime/local_runtime.go 中定义路由
2.
在 gateway_handler.go 的 handleRoute() 中添加处理逻辑
版本历史
v0.0.1 (当前版本)
•
初始版本
•
支持 macOS (arm64/amd64) 和 Windows (amd64)
•
实现 BYOK 模式
•
支持 OpenAI 和 Anthropic 模型
许可证
© 2026, Cursor助手
```

## Snapshot 19

```text
Cursor助手 - 项目技术文档
Cursor助手 - 项目技术文档
Last updated: Jun 16
请求处理流程
BYOK 模式流程
Protocol Buffer 协议
构建与开发
开发环境要求
常用命令
输出产物
配置说明
用户配置文件
代理设置
错误处理
常见错误码
日志与观测
日志文件
调试模式
安全考虑
依赖说明
主要 Go 依赖
主要前端依赖
扩展与定制
添加新的模型适配器
添加新的路由
版本历史
许可证
Cursor助手 - 项目技术文档
Modified June 16
cursor提示词与工具调用.zip
82.23KB
1.
在 internal/relay/self_implemented_*.go 中实现模型接口
2.
在 ModelAdapterConfig 中添加新类型支持
3.
更新 normalizeModelAdapterType() 函数
添加新的路由
1.
在 internal/runtime/local_runtime.go 中定义路由
2.
在 gateway_handler.go 的 handleRoute() 中添加处理逻辑
版本历史
v0.0.1 (当前版本)
•
初始版本
•
支持 macOS (arm64/amd64) 和 Windows (amd64)
•
实现 BYOK 模式
•
支持 OpenAI 和 Anthropic 模型
许可证
© 2026, Cursor助手
```
