# LANDrop — Local Network File & Text Sharing

[简体中文](README.md) | [English](README.en.md)

LANDrop is a lightweight local-network transfer utility written in Go with an embedded responsive Web UI.

Connect PCs, Macs, Linux workstations, iPhones, Android devices, and tablets over the same Wi-Fi or LAN with zero app installations, no third-party cloud relays, and full wire-speed peer-to-peer performance.

[Download v2.0.4](https://github.com/rowanjove/LANDrop/releases/tag/v2.0.4) · [Release Notes](docs/releases/v2.0.4.md) · [Architecture Design](docs/ARCHITECTURE.md) · [Report an Issue](https://github.com/rowanjove/LANDrop/issues)

## Screenshots

Desktop interface (two-column responsive cards):

![LANDrop desktop home page](docs/screenshots/landrop-2.0-desktop.jpg)

Mobile interface (instant browser scan, fluid single column):

<img src="docs/screenshots/landrop-2.0-mobile.jpg" alt="LANDrop mobile home page" width="380">

## Key Features

- **Clientless Access**: Mobile devices connect via their native browser by scanning a QR code, complete with real-time peer discovery and online status sensing.
- **Windows System Tray Resident**: Automatically hides the command prompt window and stays resident in the Windows system tray. Right-click the tray icon to quickly open the Web UI, copy the local URL, toggle the console window, or exit cleanly.
- **Single-Instance Guardian**: Prevents port conflicts through PID-based mutex and local shutdown protocol, smoothly taking over existing instances.
- **Two-Column Card Layout**: Left column features file drag-and-drop queues and clipboard/text messaging; right column displays active local devices and ready-to-download items.
- **Smart Network Detection**: Filters out virtual network interfaces (Docker, WSL, VMware, Hyper-V, Tailscale) and prioritizes physical LAN IPv4 addresses while listing backup network endpoints.
- **Maximized Wire Speed**: Large files stream directly via HTTP; directory and batch transfers use uncompressed `zip.Store` streaming to minimize CPU usage and saturate local bandwidth.
- **Timing-Safe Protection**: 4-digit dynamic PIN with `crypto/subtle.ConstantTimeCompare`, exponential lockout backoff, self-signed TLS, and one-time links.
- **Complete CLI Mode**: Full headless support for headless servers and terminal environments with `send`, `recv`, resume, and device scanning.

## Quick Start

Download the release archive for your platform from [GitHub Releases](https://github.com/rowanjove/LANDrop/releases/tag/v2.0.4) and run:

### Windows
Double-click `landrop.exe` (starts silently in the tray by default), or run in terminal:
```powershell
.\landrop.exe
```

Optional flags:
- `.\landrop.exe --console`: Keep the command prompt window visible
- `.\landrop.exe --no-tray`: Disable the system tray icon
- `.\landrop.exe --port 53217`: Specify custom port

### macOS / Linux
```bash
chmod +x landrop
./landrop
```

The app will print local URLs and a terminal QR code. Other devices on the same Wi-Fi can scan or visit the address to start transferring immediately.

## CLI Usage

```bash
# Start background server on a custom port
landrop serve --port 53217

# Send files or directories to the LAN (or drag files onto the binary directly)
landrop send ./document.pdf ./photos/

# Send a text message
landrop send --text "Notification message"

# Receive files into a directory from a target node
landrop recv ./downloads --target 192.168.1.10:53217

# Resume an interrupted download
landrop recv ./downloads --continue --target 192.168.1.10:53217

# Enable PIN protection, TLS, and one-time links
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

# Run automated tests
go test ./...

# Build binary with embedded web assets
go build -ldflags "-s -w" -o landrop .
```

Windows executable (with embedded icon resources):
```powershell
go build -ldflags "-s -w" -o landrop.exe .
```

## Documentation

- [Architecture Design](docs/ARCHITECTURE.md)
- [v2.0.4 Release Notes](docs/releases/v2.0.4.md)
- [v2.0.3 Release Notes](docs/releases/v2.0.3.md)
- [v2.0.2 Release Notes](docs/releases/v2.0.2.md)
- [v2.0.1 Release Notes](docs/releases/v2.0.1.md)
- [v2.0.0 Release Notes](docs/releases/v2.0.0.md)

## License

This project is licensed under the [MIT License](LICENSE).
