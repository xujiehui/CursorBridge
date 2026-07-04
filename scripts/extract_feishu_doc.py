#!/usr/bin/env python3
from __future__ import annotations

import json
import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RAW_PATH = ROOT / "docs" / "feishu" / "cursor-assistant-tech-doc.raw.json"
TEXT_PATH = ROOT / "docs" / "feishu" / "cursor-assistant-tech-doc.raw.txt"
MD_PATH = ROOT / "docs" / "feishu" / "cursor-assistant-tech-doc.md"
SNAPSHOTS_MD_PATH = ROOT / "docs" / "feishu" / "cursor-assistant-tech-doc.snapshots.md"

NOISE_LINES = {
    "Feishu Docs",
    "L",
    "leokun's Docs",
    "Shared With Me",
    "leokun",
    "Log In or Sign Up",
    "Comments (0)",
    "Comments",
    "Help Center",
    "Keyboard Shortcuts",
    "Copy",
    "Code block",
    "Plain Text",
    "Bash",
    "JSON",
    "Go",
    "TypeScript",
}

SECTION_HEADERS = [
    "Cursor助手 - 项目技术文档",
    "项目概述",
    "技术栈",
    "架构设计",
    "目录结构",
    "核心模块详解",
    "1. MITM 代理服务 (`internal/mitm`)",
    "2. Relay 网关 (`internal/relay`)",
    "3. BYOK 模型网关 (`internal/relay/self_implemented_*.go`)",
    "4. 证书管理 (`internal/certs`)",
    "5. Cursor 集成 (`internal/cursor`)",
    "6. 桥接服务 (`internal/bridge`)",
    "数据流详解",
    "请求处理流程",
    "BYOK 模式流程",
    "Protocol Buffer 协议",
    "构建与开发",
    "开发环境要求",
    "常用命令",
    "输出产物",
    "配置说明",
    "用户配置文件",
    "代理设置",
    "错误处理",
    "常见错误码",
    "日志与观测",
    "日志文件",
    "调试模式",
    "安全考虑",
    "依赖说明",
    "主要 Go 依赖",
    "主要前端依赖",
    "扩展与定制",
    "添加新的模型适配器",
    "添加新的路由",
    "版本历史",
    "许可证",
]

TOC_HEADERS = set(SECTION_HEADERS[1:])


def normalize_line(line: str) -> str:
    line = line.replace("\u200b", "")
    line = line.replace("\ufeff", "")
    return line.strip()


def clean_lines(text: str) -> list[str]:
    lines: list[str] = []
    for raw in text.splitlines():
        line = normalize_line(raw)
        if not line:
            continue
        if line in NOISE_LINES:
            continue
        if re.fullmatch(r"user \d+", line):
            continue
        if re.fullmatch(r"\d{1,2}:\d{2} [AP]M [A-Z][a-z]{2} \d{1,2}", line):
            continue
        if line == "Win x86版本 运行不了哎":
            continue
        lines.append(line)
    return lines


def longest_common_overlap(a: list[str], b: list[str], max_len: int = 80) -> int:
    limit = min(len(a), len(b), max_len)
    for size in range(limit, 0, -1):
        if a[-size:] == b[:size]:
            return size
    return 0


def merge_chunks(chunks: list[dict]) -> list[str]:
    merged: list[str] = []
    for chunk in chunks:
        lines = clean_lines(chunk["text"])
        if not lines:
            continue
        overlap = longest_common_overlap(merged, lines)
        if overlap:
            merged.extend(lines[overlap:])
            continue

        # Feishu keeps the page title and TOC in each virtualized chunk. Use
        # the last repeated document title or TOC marker as the real content
        # boundary for chunks that do not have a clean line overlap.
        start = 0
        for idx, line in enumerate(lines):
            if line in TOC_HEADERS:
                start = idx + 1
        for idx, line in enumerate(lines):
            if idx > start and line == "Cursor助手 - 项目技术文档":
                start = idx + 1
        candidate = lines[start:]
        overlap = longest_common_overlap(merged, candidate)
        merged.extend(candidate[overlap:] if overlap else candidate)
    return compact_repeated_runs(merged)


def compact_repeated_runs(lines: list[str]) -> list[str]:
    compact: list[str] = []
    for line in lines:
        if compact and compact[-1] == line:
            continue
        compact.append(line)
    return compact


def write_snapshots_markdown(data: dict) -> None:
    lines = [
        "---",
        f"source_url: {data['sourceUrl']}",
        f"captured_at: {data['capturedAt']}",
        f"title: {data['title']}",
        f"final_url: {data['finalUrl']}",
        "mode: virtual-scroll snapshots",
        "---",
        "",
        "# Cursor助手 - 项目技术文档（飞书滚动快照）",
        "",
        "> 飞书页面使用虚拟滚动。此文件按滚动顺序保留每次视口可见正文，作为本地忠实副本；相邻快照可能有重叠。",
        "",
    ]
    for chunk in data["chunks"]:
        chunk_lines = clean_lines(chunk["text"])
        if not chunk_lines:
            continue
        lines.append(f"## Snapshot {chunk['i']}")
        lines.append("")
        lines.append("```text")
        lines.extend(chunk_lines)
        lines.append("```")
        lines.append("")
    SNAPSHOTS_MD_PATH.write_text("\n".join(lines))


def line_to_markdown(line: str, index: int) -> str:
    if index == 0 and line == "Cursor助手 - 项目技术文档":
        return "# Cursor助手 - 项目技术文档"
    if line in SECTION_HEADERS[1:]:
        return f"## {line}"
    if re.match(r"^\d+\. .+`.+`", line):
        return f"### {line}"
    if line.startswith("•"):
        return f"- {line[1:].strip()}"
    return line


def main() -> None:
    data = json.loads(RAW_PATH.read_text())
    write_snapshots_markdown(data)
    lines = merge_chunks(data["chunks"])
    raw_text = "\n".join(lines) + "\n"
    TEXT_PATH.write_text(raw_text)

    markdown_lines = [
        "---",
        f"source_url: {data['sourceUrl']}",
        f"captured_at: {data['capturedAt']}",
        f"title: {data['title']}",
        f"final_url: {data['finalUrl']}",
        "---",
        "",
    ]
    markdown_lines.extend(line_to_markdown(line, index) for index, line in enumerate(lines))
    MD_PATH.write_text("\n".join(markdown_lines) + "\n")
    print(f"wrote {TEXT_PATH}")
    print(f"wrote {MD_PATH}")
    print(f"wrote {SNAPSHOTS_MD_PATH}")
    print(f"lines {len(lines)}")


if __name__ == "__main__":
    main()
