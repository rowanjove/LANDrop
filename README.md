# LAN Drop — 局域网文件与文本分享

[简体中文](README.md) | [English](README.en.md)

LAN Drop 是一个 Go 编写的局域网分享工具，用于在同一网络内传送文件、文本和剪贴板内容。它提供桌面与手机共用的 Web 界面，以及发送、接收、设备发现和历史记录命令，不需要云端中转服务。

[下载 v2.0.1](https://github.com/rowanjove/LANDrop/releases/tag/v2.0.1) · [查看更新说明](docs/releases/v2.0.1.md) · [报告问题](https://github.com/rowanjove/LANDrop/issues)

## 界面截图

桌面端：

![LANDrop 2.0 桌面端首页](docs/screenshots/landrop-2.0-desktop.jpg)

移动端：

<img src="docs/screenshots/landrop-2.0-mobile.jpg" alt="LANDrop 2.0 移动端首页" width="390">

## 2.0 更新

- 使用 Preact、TypeScript 和模块化 CSS 重建 Web 界面，桌面端与移动端共用一套功能。
- 文件选择改为发送队列，可继续添加、移除或清空，确认后才开始上传。
- 多文件传输支持逐个下载，也可按需下载 ZIP。
- 新增独立接收页面，文件和文本可以预览、复制或下载。
- 完善传输进度、取消、过期清理、断点续传和一次性链接生命周期。
- 设备名、主题和语言由后端持久化，刷新与重启后保留。
- mDNS 会携带 HTTP／HTTPS 协议，设备跳转和二维码使用正确地址。
- 双击 `landrop.exe` 或运行 `landrop serve` 后自动打开浏览器，终端输出以中文为主。

完整内容见 [v2.0.0 更新说明](docs/releases/v2.0.0.md)，本次隐私与文档清理见 [v2.0.1 更新说明](docs/releases/v2.0.1.md)。

## 安装与开始分享

Release 提供 Windows amd64、macOS amd64／arm64、Linux amd64／arm64 压缩包。下载对应平台版本并解压。

Windows：

```powershell
.\landrop.exe
```

macOS／Linux：

```bash
chmod +x landrop
./landrop
```

程序启动后会自动打开本机浏览器，同时在终端显示局域网访问地址和二维码。手机可在同一局域网内扫描二维码进入页面。请允许必要的局域网防火墙访问，不要把服务端口转发到公网。

## 支持的操作

- 在浏览器中选择或拖入单个、多个文件。
- 创建文本分享，保留中文、Emoji、代码和换行。
- 在 Web 界面推送和读取纯文本剪贴板内容。
- 通过 mDNS 发现同一局域网内的设备，也可手动指定地址。
- 使用 PIN、HTTPS/TLS 和一次性下载链接控制访问。
- 查看传输进度、状态和本地历史记录。
- 使用 CLI 发送、接收和断点续传文件。

## 常用命令

```bash
landrop serve --port 53217
landrop send ./photo.jpg ./documents/
landrop send --text "来自 LAN Drop 的消息"
landrop recv . --target 192.168.1.10:53217
landrop recv . --continue --target 192.168.1.10:53217
landrop devices
landrop history --limit 20
```

启用 PIN、TLS 或一次性下载：

```bash
landrop serve --pin 5231 --tls --one-time
landrop recv . --pin 5231 --tls --target 192.168.1.10:53217
```

查看全部命令：

```bash
landrop --help
```

## 一次性链接与安全边界

一次性下载令牌在真实下载完成后消耗。打开预览页、发送 HEAD 请求或中断未完成的下载不会提前消耗令牌。

TLS 用于客户端与 LAN Drop 服务之间的传输加密。当前服务生成临时自签名证书，浏览器会提示证书不受信任；部分 CLI HTTPS 请求跳过证书验证，因此不能仅凭 `--tls` 就假设获得了完整的服务器身份校验或中间人攻击防护。

默认命令不自动启用 PIN 或 TLS。请在可信局域网中使用，按需启用访问保护并核对目标设备。LANDrop 不应直接暴露到公网。

## 从源码构建

环境要求：Go 1.25、Node.js 18 或更高版本。

```bash
git clone https://github.com/rowanjove/LANDrop.git
cd LANDrop/web
npm ci
npm run typecheck
npm run build
cd ..
go test ./...
go build -o landrop .
```

Windows 可将最后一条替换为：

```powershell
go build -o landrop.exe .
```

前端构建产物位于 `web/dist`，发布二进制通过 `go:embed` 将其内嵌，不需要用户安装 Node.js。

## 文档

- [架构说明](docs/ARCHITECTURE.md)
- [实施记录](docs/IMPLEMENTATION.md)
- [v2.0.0 更新说明](docs/releases/v2.0.0.md)
- [v2.0.1 更新说明](docs/releases/v2.0.1.md)

## 贡献与许可

通过 [Issues](https://github.com/rowanjove/LANDrop/issues) 提交问题时，请说明操作系统、版本、发送与接收方式、网络环境和复现步骤，不要附私人文件或 PIN。代码改动提交前运行 `go test ./...`、`npm run typecheck` 和 `npm run build`。

本项目采用 [MIT License](LICENSE)。
