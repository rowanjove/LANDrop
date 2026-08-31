# LAN Drop — Local network file and text sharing

[简体中文](README.md) | [English](README.en.md)

LAN Drop is a Go-based tool for sharing files, text, and clipboard content between devices on the same network. It provides a web interface for desktop and mobile browsers, plus CLI commands for sending, receiving, and viewing transfer history. No cloud relay service is required.

[Download v1.2.2](https://github.com/rowanjove/LANDrop/releases/tag/v1.2.2) · [Report an issue](https://github.com/rowanjove/LANDrop/issues)

## Install and start sharing

Release archives are available for Windows amd64, macOS amd64/arm64, and Linux amd64/arm64. Download the archive for your platform, extract it, and open a terminal in that directory.

macOS/Linux:

```bash
chmod +x landrop
./landrop serve
```

Windows PowerShell:

```powershell
.\landrop.exe serve
```

Open the printed address or scan the QR code with a phone on the same LAN. Allow necessary local-network firewall access; do not forward the service port to the public internet.

Examples below use `landrop` after its directory has been added to PATH. Otherwise, use the platform-specific executable paths shown above.

## Features

- Upload files in a browser or start a share from the CLI.
- Create text links with web preview and copy controls.
- Push and pull plain-text clipboard content through the web UI.
- Discover devices on the LAN using mDNS.
- Optionally enable PIN protection, HTTPS/TLS, and one-time download links.
- Set manual targets, resume downloads, and inspect local transfer history.

## Common commands

```bash
landrop serve --port 53217
landrop send ./photo.jpg
landrop send --text "hello from LAN Drop"
landrop recv .
```

When the server uses a PIN and TLS, supply the corresponding receiver options:

```bash
landrop serve --pin 5231 --tls
landrop recv . --pin 5231 --tls --target 192.168.1.10:53217
```

Use `--target` when mDNS discovery is unavailable. Resume an interrupted download into the same path:

```bash
landrop recv . --continue --target 192.168.1.10:53217
```

Inspect or clear local history:

```bash
landrop history --limit 20
landrop history --clear
```

`--clear` deletes local history records; confirm you no longer need them before running it.

## One-time links and security boundaries

```bash
landrop serve --one-time
```

A one-time download token is consumed after a real download completes, not when a preview page is opened.

TLS encrypts transport between a client and the LAN Drop server. The current server generates a temporary self-signed certificate, which triggers browser trust warnings. Some CLI HTTPS requests skip certificate verification, so `--tls` alone does not establish full server identity verification or protection against an active man-in-the-middle attack.

Default commands do not automatically enable a PIN or TLS. Use a trusted LAN, enable protection as needed, and verify the target device. The current CLI does not expose an option to load custom certificates; generating a local certificate does not make the application use it automatically.

## Build from source

[go.mod](go.mod) declares Go **1.25.0**. Use a compatible toolchain.

```bash
git clone https://github.com/rowanjove/LANDrop.git
cd LANDrop
go test ./...
go build -o landrop .
```

On Windows, replace the final command with:

```powershell
go build -o landrop.exe .
```

## Contributing and license

When reporting an [issue](https://github.com/rowanjove/LANDrop/issues), include your operating system, version, send/receive commands, network setup, and reproduction steps. Do not include private files or PINs. Run `go test ./...` before submitting code changes.

Licensed under the [MIT License](LICENSE).
