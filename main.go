package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Args = append(os.Args, "serve")
	}

	// Global flags
	port := flag.Int("port", 53217, "服务端口")
	pin := flag.String("pin", "", "设置 4 位 PIN 保护")
	tlsFlag := flag.Bool("tls", false, "启用 HTTPS（自签名证书）")

	switch os.Args[1] {
	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		sPort := serveCmd.Int("port", 53217, "服务端口")
		sPin := serveCmd.String("pin", "", "设置 4 位 PIN 保护")
		sTLS := serveCmd.Bool("tls", false, "启用 HTTPS")
		serveCmd.Parse(os.Args[2:])
		startServer(*sPort, *sPin, *sTLS)

	case "send":
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		sPort := sendCmd.Int("port", 53217, "服务端口")
		sPin := sendCmd.String("pin", "", "设置 4 位 PIN 保护")
		sTLS := sendCmd.Bool("tls", false, "启用 HTTPS")
		textMode := sendCmd.String("text", "", "发送文本内容")
		sendCmd.Parse(os.Args[2:])

		if *textMode != "" {
			sendText(*sPort, *sPin, *sTLS, *textMode)
		} else if sendCmd.NArg() > 0 {
			sendFile(*sPort, *sPin, *sTLS, sendCmd.Args())
		} else {
			fmt.Println("用法: landrop send [--text '内容'] <文件路径...>")
			os.Exit(1)
		}

	case "devices":
		devCmd := flag.NewFlagSet("devices", flag.ExitOnError)
		devCmd.Parse(os.Args[2:])
		listDevices()

	case "version":
		fmt.Printf("LAN Drop v%s\n", version)

	case "help", "--help", "-h":
		printUsage()

	default:
		// If first arg looks like a file, treat as "send"
		if _, err := os.Stat(os.Args[1]); err == nil {
			flag.Parse()
			sendFile(*port, *pin, *tlsFlag, os.Args[1:])
		} else {
			fmt.Printf("未知命令: %s\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Printf(`LAN Drop v%s — 局域网轻量文件与文本传输工具

用法:
  landrop serve [options]         启动服务（默认命令）
  landrop send <文件路径...>       发送文件
  landrop send --text '内容'      发送文本
  landrop devices                 列出局域网内设备
  landrop version                 显示版本

选项:
  --port <端口>    自定义端口（默认 53217）
  --pin <PIN>     设置 4 位 PIN 保护
  --tls           启用 HTTPS（自签名证书）

示例:
  landrop                         启动服务，打开浏览器使用
  landrop send report.pdf         发送文件
  landrop send --text 'hello'     发送文本
  landrop serve --port 8888       使用自定义端口
  landrop serve --pin 1234        启用 PIN 保护
`, version)
}

func sendFile(port int, pin string, useTLS bool, files []string) {
	actualPort, err := findAvailablePort(port)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	localIP := getLocalIP()
	addr := fmt.Sprintf("%s:%d", localIP, actualPort)

	app := NewApp(addr, pin)
	app.mdns = NewMDNSManager(actualPort, app.broker)

	scheme := "http"
	if useTLS {
		scheme = "https"
	}

	// Add files
	for _, filePath := range files {
		item, err := app.SendLocalFile(filePath)
		if err != nil {
			log.Printf("Error: %s - %v", filePath, err)
			continue
		}
		url := fmt.Sprintf("%s://%s/recv/%s", scheme, addr, item.Token)
		fmt.Printf("文件: %s (%s)\n", item.Name, formatSize(item.Size))
		fmt.Printf("下载链接: %s\n", url)
		fmt.Println()
		fmt.Println(generateQRASCII(url))
	}

	// Start server to serve the files
	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	var handler http.Handler = mux
	if app.pin.IsEnabled() {
		handler = app.pin.Middleware(mux)
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%d", actualPort)
	fmt.Printf("等待接收方下载... (Ctrl+C 退出)\n")

	server := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}
	if useTLS {
		cert, _ := generateSelfSignedCert()
		server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}

func sendText(port int, pin string, useTLS bool, text string) {
	actualPort, err := findAvailablePort(port)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	localIP := getLocalIP()
	addr := fmt.Sprintf("%s:%d", localIP, actualPort)

	app := NewApp(addr, pin)
	app.mdns = NewMDNSManager(actualPort, app.broker)

	item, err := app.store.AddText(text)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	scheme := "http"
	if useTLS {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://%s/recv/%s", scheme, addr, item.Token)
	fmt.Printf("文本已就绪 (%d 字节)\n", item.Size)
	fmt.Printf("获取链接: %s\n", url)
	fmt.Println()
	fmt.Println(generateQRASCII(url))

	// Start server
	mux := http.NewServeMux()
	app.SetupRoutes(mux)
	listenAddr := fmt.Sprintf("0.0.0.0:%d", actualPort)
	fmt.Printf("等待接收方获取... (Ctrl+C 退出)\n")
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

func listDevices() {
	broker := NewSSEBroker()
	mdns := NewMDNSManager(0, broker)

	fmt.Println("正在扫描局域网内的 LAN Drop 设备...")
	if err := mdns.Start(); err != nil {
		log.Fatalf("mDNS 启动失败: %v", err)
	}

	// Wait for discovery
	time.Sleep(5 * time.Second)
	devices := mdns.GetDevices()
	mdns.Stop()

	if len(devices) == 0 {
		fmt.Println("未发现其他 LAN Drop 设备")
		return
	}

	fmt.Printf("\n发现 %d 个设备:\n", len(devices))
	for _, d := range devices {
		fmt.Printf("  %s (%s) - %s\n", d.Name, d.OS, d.Addr)
	}
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
