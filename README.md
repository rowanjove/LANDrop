# LAN Drop

Local network file, text, and clipboard sharing with a simple web UI and CLI.

## 中文

### 简介

LAN Drop 是一个基于 Go 的局域网分享工具，用来在同一网络内快速发送文件、文本和剪贴板内容。它同时提供：

- 一套响应式 Web UI，桌面和手机共用同一个页面
- 用于发送和接收的 CLI 命令
- 局域网设备发现
- 一次性下载链接
- PIN 保护和 HTTPS/TLS 支持

### 功能特性

- 文件分享：支持浏览器上传和 CLI 发送
- 文本分享：支持链接生成、预览和复制
- 剪贴板同步：Web UI 支持纯文本推送和拉取
- 设备发现：通过 mDNS 在局域网内发现其他节点
- 安全选项：支持 `--pin`、`--tls` 和 `--one-time`
- 接收增强：支持手动目标地址、断点续传和传输历史
- 跨平台：支持 Windows、macOS 和 Linux

### 快速开始

1. 启动服务

```bash
landrop serve
```

2. 在浏览器中打开输出的地址，或直接扫描二维码

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
landrop recv . --target 192.168.1.10:53217
landrop recv . --continue --target 192.168.1.10:53217

landrop history --limit 20
landrop history --clear
```

### 高级安全

如果你的局域网并不完全可信，可以同时开启 PIN 和 TLS 来保护传输：

```bash
landrop serve --pin 5231 --tls
```

启用 `--tls` 后，传输内容会进行端到端加密。

> 提示：当前 TLS 默认使用临时生成的自签名证书。如果你不想看到浏览器证书警告，可以使用 [`mkcert`](https://github.com/FiloSottile/mkcert) 为本地局域网 IP 生成受信任证书；或者在浏览器的高级提示页中手动继续访问。

### 一次性链接

如果你希望分享链接只允许成功下载一次，可以这样启动：

```bash
landrop serve --one-time
```

Token 会在真正的下载完成后被消费，不会因为打开预览页面而失效。

### CLI 额外能力

如果 mDNS 发现不可用，可以直接连接已知设备地址：

```bash
landrop recv . --target 192.168.1.10:53217
```

如果下载被中断，可以继续写入同一路径：

```bash
landrop recv . --continue --target 192.168.1.10:53217
```

查看或清理本地传输历史：

```bash
landrop history --limit 20
landrop history --clear
```

### 从源码运行

```bash
go test ./...
go build -o landrop .
```

### 发布说明

GitHub Releases 提供以下预编译二进制：

- Windows `amd64`
- macOS `amd64` / `arm64`
- Linux `amd64` / `arm64`

### 许可证

本项目基于 `MIT License` 开源，详见 [LICENSE](./LICENSE)。

## English

### Overview

LAN Drop is a Go-based LAN sharing tool for quickly sending files, text, and clipboard content across devices on the same network. It includes:

- A responsive web UI shared by desktop and mobile
- CLI commands for sending and receiving
- Local device discovery
- One-time download links
- PIN protection and HTTPS/TLS support

### Features

- File sharing from both browser and CLI
- Text sharing with preview and copy support
- Plain-text clipboard push and pull in the web UI
- mDNS-based device discovery on the local network
- Security options with `--pin`, `--tls`, and `--one-time`
- Enhanced receiving with manual targets, resume, and transfer history
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
landrop recv . --target 192.168.1.10:53217
landrop recv . --continue --target 192.168.1.10:53217

landrop history --limit 20
landrop history --clear
```

### Advanced Security

If your local network is not fully trusted, you can protect transfers with both PIN and TLS:

```bash
landrop serve --pin 5231 --tls
```

With `--tls`, transfers are encrypted end-to-end.

> Tip: TLS currently uses a temporary self-signed certificate. If you want to avoid browser warnings, you can use [`mkcert`](https://github.com/FiloSottile/mkcert) to generate a locally trusted certificate for your LAN IP, or proceed through the browser's advanced warning page.

### One-Time Links

To make a shared link valid for only one successful download:

```bash
landrop serve --one-time
```

The token is consumed when a real download completes, not when a preview page is opened.

### CLI Extras

If mDNS discovery is unavailable, connect directly to a known device:

```bash
landrop recv . --target 192.168.1.10:53217
```

To resume an interrupted download into the same local path:

```bash
landrop recv . --continue --target 192.168.1.10:53217
```

To inspect or clear the local transfer history:

```bash
landrop history --limit 20
landrop history --clear
```

### Build From Source

```bash
go test ./...
go build -o landrop .
```

### Releases

GitHub Releases include prebuilt binaries for:

- Windows `amd64`
- macOS `amd64` / `arm64`
- Linux `amd64` / `arm64`

### License

This project is released under the `MIT License`. See [LICENSE](./LICENSE).
