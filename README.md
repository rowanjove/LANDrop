# LAN Drop — 局域网文件与文本分享

[简体中文](README.md) | [English](README.en.md)

LAN Drop 是一个 Go 编写的局域网分享工具，用于在同一网络内传送文件、文本和剪贴板内容。它提供桌面与手机共用的 Web 界面，以及发送、接收和查看历史记录的命令行工具，不需要部署云端中转服务。

[下载 v1.2.2](https://github.com/rowanjove/LANDrop/releases/tag/v1.2.2) · [报告问题](https://github.com/rowanjove/LANDrop/issues)

## 安装与开始分享

Release 提供 Windows amd64、macOS amd64／arm64、Linux amd64／arm64 压缩包。下载对应平台版本，解压后在终端进入所在目录。

macOS／Linux：

```bash
chmod +x landrop
./landrop serve
```

Windows PowerShell：

```powershell
.\landrop.exe serve
```

打开程序打印的地址，或在同一局域网的手机上扫描二维码。请允许必要的局域网防火墙访问，不要把服务端口转发到公网。

下文以已将程序目录加入 PATH 后的 `landrop` 命令为例；未加入 PATH 时，使用上方的平台路径写法。

## 支持的操作

- 从浏览器上传文件，或用 CLI 发起分享。
- 生成文本分享链接，在 Web 界面预览和复制。
- 在 Web 界面推送与拉取纯文本剪贴板内容。
- 通过 mDNS 发现同一局域网内的设备。
- 可选 PIN、HTTPS/TLS 与一次性下载链接。
- 手动指定目标地址、继续下载及本地传输历史。

## 常用命令

```bash
landrop serve --port 53217
landrop send ./photo.jpg
landrop send --text "hello from LAN Drop"
landrop recv .
```

服务端启用 PIN 和 TLS 时，接收端应使用对应选项：

```bash
landrop serve --pin 5231 --tls
landrop recv . --pin 5231 --tls --target 192.168.1.10:53217
```

mDNS 不可用时，通过 `--target` 指定设备。继续写入同一路径的未完成下载：

```bash
landrop recv . --continue --target 192.168.1.10:53217
```

查看或清除本地历史：

```bash
landrop history --limit 20
landrop history --clear
```

`--clear` 会删除本地历史记录，执行前确认不再需要。

## 一次性链接与安全边界

```bash
landrop serve --one-time
```

一次性下载令牌在真实下载完成后消耗，打开预览页不会消耗令牌。

TLS 用于客户端与 LAN Drop 服务之间的传输加密。当前服务生成临时自签名证书，浏览器会提示证书不受信任；部分 CLI HTTPS 请求跳过证书验证，因此不能仅凭 `--tls` 就假设获得了完整的服务器身份校验或中间人攻击防护。

默认命令不自动启用 PIN 或 TLS。请在可信局域网中使用，按需启用访问保护，核对目标设备。当前命令没有提供直接加载自定义证书的选项，不应把生成本地证书理解为应用会自动使用它。

## 从源码构建

项目的 [go.mod](go.mod) 声明 Go **1.25.0**；使用兼容工具链。

```bash
git clone https://github.com/rowanjove/LANDrop.git
cd LANDrop
go test ./...
go build -o landrop .
```

Windows 可将最后一条替换为：

```powershell
go build -o landrop.exe .
```

## 贡献与许可

通过 [Issues](https://github.com/rowanjove/LANDrop/issues) 提交问题时，说明操作系统、版本、发送与接收命令、网络环境及复现步骤，不要附私人文件或 PIN。代码改动提交前运行 `go test ./...`。

本项目采用 [MIT License](LICENSE)。
