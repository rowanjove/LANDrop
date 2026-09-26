# LANDrop — Local Network File & Text Sharing

[简体中文](README.md) | [English](README.en.md)

LANDrop is a lightweight local-network transfer utility written in Go with an embedded responsive Web UI. It enables instant file, text, and clipboard sharing between PCs, phones, and tablets over the same Wi-Fi or LAN without requiring app installation or cloud relay services.

[Download v2.0.3](https://github.com/rowanjove/LANDrop/releases/tag/v2.0.3) · [Release Notes](docs/releases/v2.0.3.md) · [Report an Issue](https://github.com/rowanjove/LANDrop/issues)

## Screenshots

Desktop interface:

![LANDrop desktop home page](docs/screenshots/landrop-2.0-desktop.jpg)

Mobile interface:

<img src="docs/screenshots/landrop-2.0-mobile.jpg" alt="LANDrop mobile home page" width="380">

## Key Features

- **Zero-Setup Startup**: Launches directly, opens your default browser automatically, and prints the local IP and QR code in the terminal.
- **Clientless Experience**: Mobile phones and tablets connect via their built-in browser with automatic device discovery and online status detection.
- **Send Queue & Batch Transfers**: Supports drag-and-drop file queues, individual file downloads, and on-demand automatic ZIP bundling.
- **Text & Clipboard Sync**: Share text messages and sync plain-text clipboard contents while preserving formatting and line breaks.
- **Access Control & Encryption**: Optional 4-digit dynamic PIN verification, self-signed HTTPS/TLS transport, and one-time download links.
- **Complete CLI Support**: Full command-line interface for headless environments, supporting sending, receiving, resuming downloads, and device scanning.

## Quick Start

Download the release archive for your platform from [GitHub Releases](https://github.com/rowanjove/LANDrop/releases/tag/v2.0.3) and run:

### Windows
Double-click `landrop.exe`, or run in terminal:
```powershell
.\landrop.exe
```

### macOS / Linux
```bash
chmod +x landrop
./landrop
```

The app will print the local URL and QR code in the console. Other devices on the same Wi-Fi can scan or visit the address to start transferring.

## CLI Usage

```bash
# Start background server on a specific port
landrop serve --port 53217

# Send files or directories to the LAN
landrop send ./photo.jpg ./documents/

# Send a text message
landrop send --text "Notification message"

# Receive files into a directory from a target node
landrop recv ./downloads --target 192.168.1.10:53217

# Resume an interrupted download
landrop recv ./downloads --continue --target 192.168.1.10:53217

# Enable PIN protection, TLS, and one-time downloads
landrop serve --pin 8848 --tls --one-time

# Scan for active devices on the local network
landrop devices

# View recent transfer history
landrop history --limit 20
```

## Building from Source

Requirements: Go 1.22+ and Node.js 18+.

```bash
# Clone the repository
git clone https://github.com/rowanjove/LANDrop.git
cd LANDrop

# Build web frontend assets
cd web
npm ci
npm run build
cd ..

# Build binary with embedded web assets
go test ./...
go build -ldflags "-s -w" -o landrop .
```

Windows executable:
```powershell
go build -ldflags "-s -w" -o landrop.exe .
```

## Documentation

- [Architecture Design](docs/ARCHITECTURE.md)
- [v2.0.3 Release Notes](docs/releases/v2.0.3.md)
- [v2.0.2 Release Notes](docs/releases/v2.0.2.md)
- [v2.0.1 Release Notes](docs/releases/v2.0.1.md)
- [v2.0.0 Release Notes](docs/releases/v2.0.0.md)

## License

This project is licensed under the [MIT License](LICENSE).
