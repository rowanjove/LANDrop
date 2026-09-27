# LANDrop — 局域网文件与文本分享

[简体中文](README.md) | [English](README.en.md)

LANDrop 是一个专为局域网环境设计的轻量级文件与文本互传工具。采用 Go 编写，单文件无依赖，内嵌响应式 Web 操作界面。

在同一 Wi-Fi 或局域网下，任何设备（PC、Mac、Linux、iPhone、Android、iPad）只需通过原生浏览器即可高速收发文件与剪贴板内容，免装客户端，不经过任何第三方公网服务器。

[下载最新版本 (v2.0.4)](https://github.com/rowanjove/LANDrop/releases/tag/v2.0.4) · [更新说明](docs/releases/v2.0.4.md) · [系统架构设计](docs/ARCHITECTURE.md) · [提交反馈](https://github.com/rowanjove/LANDrop/issues)

## 界面预览

桌面端界面（双栏响应式卡片）：

![LANDrop 桌面端首页](docs/screenshots/landrop-2.0-desktop.jpg)

移动端界面（扫码即用，流式垂直单列）：

<img src="docs/screenshots/landrop-2.0-mobile.jpg" alt="LANDrop 移动端首页" width="380">

## 核心特性

- **零客户端接入**：手机、平板自带浏览器扫码即可收发文件，支持移动设备自动发现与在线状态感知。
- **Windows 系统托盘常驻**：默认启动后自动隐藏控制台黑框并常驻系统托盘，右键可快捷打开网页、复制局域网地址、切换控制台显示或退出程序。
- **单实例运行守护**：基于 PID 互斥与本地协议防止重复启动导致的端口冲突，平滑接管运行状态。
- **双栏卡片化工作区**：左侧聚合文件拖拽队列与文本/剪贴板发送，右侧实时显示局域网在线节点与本次传输待下载项目。
- **智能网卡与多 IP 识别**：自动过滤 Docker、WSL、虚拟机等虚拟网络接口，优先选择物理网卡真实局域网 IP，并展示备用地址。
- **跑满内网线速**：大文件流式透传，多文件打包采用 `zip.Store` 零压缩流式直出，极大降低主机 CPU 消耗，打满千兆/万兆局域网带宽。
- **安全与时序防护**：支持可选 4 位动态 PIN 码（防侧信道时序探测 + 错误指数退避锁定）、自签名 TLS 加密传输以及阅后即焚一次性链接。
- **纯终端命令行支持**：提供完整的无头模式，支持在无图形界面的服务器或命令行终端中完成收发、断点续传与局域网节点探测。

## 快速使用

在 [GitHub Releases](https://github.com/rowanjove/LANDrop/releases/tag/v2.0.4) 下载对应操作系统的发布包，解压后即可直接运行：

### Windows
双击 `landrop.exe` 直接启动（默认隐藏控制台并常驻任务栏右下角托盘），或在终端中运行：
```powershell
.\landrop.exe
```

常用可选参数：
- `.\landrop.exe --console`：保留控制台黑框窗口
- `.\landrop.exe --no-tray`：禁用系统托盘图标
- `.\landrop.exe --port 53217`：指定服务监听端口

### macOS / Linux
```bash
chmod +x landrop
./landrop
```

启动后程序将自动唤起默认浏览器，并在控制台输出本机局域网地址与二维码。局域网内其他设备扫码或访问该地址即可互通。

## 常用命令行示例

```bash
# 启动后台服务并自定义端口
landrop serve --port 53217

# 发送文件或目录至局域网（拖拽文件至可执行文件亦可触发）
landrop send ./document.pdf ./photos/

# 发送纯文本消息
landrop send --text "局域网通知内容"

# 接收文件并保存至指定目录（自动连接目标节点）
landrop recv ./downloads --target 192.168.1.10:53217

# 断点续传未下载完成的大文件
landrop recv ./downloads --continue --target 192.168.1.10:53217

# 启用 4 位数字 PIN 码、TLS 加密与一次性阅后即焚链接
landrop serve --pin 8848 --tls --one-time

# 扫描并查看局域网内当前在线设备
landrop devices

# 查看最近传输历史记录
landrop history --limit 20
```

## 从源码构建

环境要求：Go 1.22+，Node.js 18+。

```bash
# 克隆仓库
git clone https://github.com/rowanjove/LANDrop.git
cd LANDrop

# 构建前端产物
cd web
npm ci
npm run build
cd ..

# 运行自动化测试
go test ./...

# 编译各平台二进制（内嵌前端静态资源）
go build -ldflags "-s -w" -o landrop .
```

Windows 环境输出（含程序图标资源）：
```powershell
go build -ldflags "-s -w" -o landrop.exe .
```

## 相关文档

- [系统架构设计](docs/ARCHITECTURE.md)
- [v2.0.4 更新日志](docs/releases/v2.0.4.md)
- [v2.0.3 更新日志](docs/releases/v2.0.3.md)
- [v2.0.2 更新日志](docs/releases/v2.0.2.md)
- [v2.0.1 更新日志](docs/releases/v2.0.1.md)
- [v2.0.0 更新日志](docs/releases/v2.0.0.md)

## 开源协议

本项目基于 [MIT License](LICENSE) 许可协议开放。
