# LAN Drop

Local network file, text, and clipboard sharing with a simple web UI and CLI.

## 中文

### 简介

LAN Drop 是一个基于 Go 的局域网分享工具，用来在同一网络内快速发送文件、文本和剪贴板内容。它同时提供：

- 浏览器网页端，自动适配桌面和手机
- 命令行发送与接收
- 局域网设备发现
- 一次性链接分享
- PIN 保护与 HTTPS/TLS

### 功能特性

- 文件分享：支持浏览器上传和 CLI 发送
- 文本分享：可直接生成链接，网页端可预览和复制
- 剪贴板同步：支持推送纯文本到接收端
- 设备发现：通过 mDNS 在局域网内发现其他节点
- 安全选项：支持 `--pin`、`--tls`、`--one-time`
- 跨平台：Windows、macOS、Linux

### 快速开始

1. 启动服务

```bash
landrop serve
```

2. 打开浏览器访问输出中的地址，或扫描二维码

3. 也可以直接使用 CLI

```bash
landrop send ./photo.jpg
landrop send --text "hello from LAN Drop"
landrop recv .
```

### 常用命令

```bash
landrop serve --port 53217
landrop serve --pin 1234
landrop serve --tls
landrop serve --one-time

landrop send ./file.zip
landrop send --text "hello"

landrop recv .
landrop recv . --pin 1234
landrop recv . --tls
```

### 从源码运行

```bash
go test ./...
go build -o landrop.exe .
```

### 发布说明

GitHub Releases 中会提供预编译版本：

- Windows `amd64`
- macOS `amd64` / `arm64`
- Linux `amd64` / `arm64`

## English

### Overview

LAN Drop is a Go-based LAN sharing tool for quickly sending files, text, and clipboard content across devices on the same network. It includes:

- A browser-based UI that adapts to desktop and mobile
- CLI commands for sending and receiving
- Local device discovery
- One-time download links
- PIN protection and HTTPS/TLS support

### Features

- File sharing from both browser and CLI
- Text sharing with preview and copy support
- Clipboard push for plain text
- mDNS-based device discovery on the local network
- Security options with `--pin`, `--tls`, and `--one-time`
- Cross-platform support for Windows, macOS, and Linux

### Quick Start

1. Start the server

```bash
landrop serve
```

2. Open the printed URL in a browser, or scan the QR code

3. Or use the CLI directly

```bash
landrop send ./photo.jpg
landrop send --text "hello from LAN Drop"
landrop recv .
```

### Common Commands

```bash
landrop serve --port 53217
landrop serve --pin 1234
landrop serve --tls
landrop serve --one-time

landrop send ./file.zip
landrop send --text "hello"

landrop recv .
landrop recv . --pin 1234
landrop recv . --tls
```

### Build From Source

```bash
go test ./...
go build -o landrop .
```

### Releases

GitHub Releases will include prebuilt binaries for:

- Windows `amd64`
- macOS `amd64` / `arm64`
- Linux `amd64` / `arm64`
