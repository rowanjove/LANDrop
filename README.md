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
- 剪贴板同步：Web UI 支持纯文本推送与拉取
- 设备发现：通过 mDNS 在局域网内发现其他节点
- 安全选项：支持 `--pin`、`--tls`、`--one-time`
- 接收增强：支持手动目标地址、断点续传和传输历史
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
landrop recv . --target 192.168.1.10:53217
landrop recv . --continue --target 192.168.1.10:53217

landrop history --limit 20
landrop history --clear
```

### 高级功能：TLS 与 PIN 保护

如果局域网存在不可信节点（例如咖啡厅公开 Wi-Fi），你可以启用传输保护：

```bash
# 启动 4 位 PIN 和 TLS 保护
landrop serve --pin 5231 --tls
```

启用 `--tls` 后传输将完全加密。

> **提示**：目前启用 TLS 使用的是临时生成的自签名证书。如果不希望浏览器提示证书不安全：
>
> **手动安装信任根证书**：建议使用 [`mkcert`](https://github.com/FiloSottile/mkcert) 安装自签名的根证书并为本地网络 IP 发放信任的 CA 证书替换项目内置生成。或者在浏览器弹出安全警告时，点击高级，选择继续访问（不安全）即可正常打开。

### 一次性分享与阅后即焚

为了安全控制大文件只被受众下载一次，不占用本地带宽，可以加上一次性标志：

```bash
# 生成阅后即焚链接（Token 仅在触发真实读取下载流时核销并失效）
landrop serve --one-time
```

### CLI 增强功能

当 mDNS 发现失败时，可以手动指定目标设备地址：

```bash
landrop recv . --target 192.168.1.10:53217
```

如果下载中断，可以在同一路径上尝试续传：

```bash
landrop recv . --continue --target 192.168.1.10:53217
```

还可以查看或清理本地传输历史：

```bash
landrop history --limit 20
landrop history --clear
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

### 许可证

本项目基于 `MIT License` 开源，详见 [LICENSE](./LICENSE)。

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

GitHub Releases will include prebuilt binaries for:

- Windows `amd64`
- macOS `amd64` / `arm64`
- Linux `amd64` / `arm64`

### License

This project is released under the `MIT License`. See [LICENSE](./LICENSE).
