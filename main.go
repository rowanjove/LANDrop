package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		sOneTime := serveCmd.Bool("one-time", false, "Token 一次有效")
		serveCmd.Parse(os.Args[2:])
		startServer(*sPort, *sPin, *sTLS, *sOneTime)

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

	case "recv":
		recvCmd := flag.NewFlagSet("recv", flag.ExitOnError)
		rPin := recvCmd.String("pin", "", "璁剧疆 4 浣?PIN 淇濇姢")
		rTLS := recvCmd.Bool("tls", false, "鍚敤 HTTPS")
		recvCmd.Parse(os.Args[2:])
		saveDir := "."
		if recvCmd.NArg() > 0 {
			saveDir = recvCmd.Arg(0)
		}
		recvMode(saveDir, *rPin, *rTLS)

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
  landrop recv [save-dir]         接收模式，自动保存到指定目录
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

	// Start server with PIN and TLS support
	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	var handler http.Handler = mux
	if app.pin.IsEnabled() {
		handler = app.pin.Middleware(mux)
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%d", actualPort)
	fmt.Printf("等待接收方获取... (Ctrl+C 退出)\n")

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

func recvMode(saveDir string, pin string, useTLS bool) {
	// Ensure save directory exists
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		log.Fatalf("无法创建保存目录: %v", err)
	}
	absDir, _ := filepath.Abs(saveDir)
	fmt.Printf("接收模式已启动，文件将保存到: %s\n", absDir)
	fmt.Println("正在扫描局域网内的 LAN Drop 设备...")

	broker := NewSSEBroker()
	mdns := NewMDNSManager(0, broker)
	if err := mdns.Start(); err != nil {
		log.Printf("mDNS 启动失败: %v", err)
	}

	// Wait for discovery then poll
	time.Sleep(3 * time.Second)
	devices := mdns.GetDevices()

	if len(devices) == 0 {
		fmt.Println("未发现设备，请手动输入设备地址（如 192.168.1.10:53217）:")
		var addr string
		fmt.Scanln(&addr)
		if addr != "" {
			pollAndSave(addr, absDir, pin, useTLS)
		}
		return
	}

	fmt.Printf("发现 %d 个设备:\n", len(devices))
	for i, d := range devices {
		fmt.Printf("  [%d] %s (%s)\n", i+1, d.Name, d.Addr)
	}
	fmt.Print("选择设备编号: ")
	var choice int
	fmt.Scanln(&choice)
	if choice < 1 || choice > len(devices) {
		fmt.Println("无效选择")
		return
	}

	pollAndSave(devices[choice-1].Addr, absDir, pin, useTLS)
}

func pollAndSave(addr string, saveDir string, pin string, useTLS bool) {
	fmt.Printf("连接到 %s，等待文件...\n", addr)
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	baseURL := scheme + "://" + addr

	client := &http.Client{}
	if useTLS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// Connect to SSE for real-time notifications
	req, _ := http.NewRequest("GET", baseURL+"/events", nil)
	if pin != "" {
		req.Header.Set("X-LanDrop-PIN", pin)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("连接失败: HTTP %d (可能需要 PIN)", resp.StatusCode)
	}

	fmt.Println("已连接，等待接收... (Ctrl+C 退出)")

	// Proper SSE parsing: read line-by-line, accumulate event type + data
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB max line
	var eventType, eventData string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = end of event, dispatch
			if eventType == "file_ready" && eventData != "" {
				var event struct {
					Token string `json:"token"`
					Name  string `json:"name"`
					Size  int64  `json:"size"`
					Type  string `json:"type"`
				}
				if err := json.Unmarshal([]byte(eventData), &event); err == nil {
					if event.Type == "text" {
						textReq, _ := http.NewRequest("GET", baseURL+"/recv/"+event.Token, nil)
						if pin != "" {
							textReq.Header.Set("X-LanDrop-PIN", pin)
						}
						textResp, err := client.Do(textReq)
						if err != nil {
							fmt.Printf("获取文本失败: %v\n", err)
						} else {
							var textData struct {
								Content string `json:"content"`
							}
							json.NewDecoder(textResp.Body).Decode(&textData)
							textResp.Body.Close()
							fmt.Printf("\n收到文本 (%s):\n%s\n", formatSize(event.Size), textData.Content)
						}
					} else {
						fmt.Printf("\n收到文件: %s (%s)\n", event.Name, formatSize(event.Size))
						savePath := filepath.Join(saveDir, sanitizeFilename(event.Name))
						dlReq, _ := http.NewRequest("GET", baseURL+"/recv/"+event.Token, nil)
						if pin != "" {
							dlReq.Header.Set("X-LanDrop-PIN", pin)
						}
						savePath, err = downloadFileWithClient(client, dlReq, savePath)
						if err != nil {
							fmt.Printf("下载失败: %v\n", err)
						} else {
							fmt.Printf("已保存到: %s\n", savePath)
						}
					}
				}
			}
			eventType = ""
			eventData = ""
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			if eventData != "" {
				eventData += "\n"
			}
			eventData += strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("连接中断: %v", err)
	}
}

func downloadFile(url string, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	_, err = saveDownload(resp.Body, savePath)
	return err
}

func downloadFileWithClient(client *http.Client, req *http.Request, savePath string) (string, error) {
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return saveDownload(resp.Body, savePath)
}

func saveDownload(src io.Reader, savePath string) (string, error) {
	finalPath, err := reserveDownloadPath(savePath)
	if err != nil {
		return "", err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(finalPath), filepath.Base(finalPath)+".part-*")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(tempFile, src); err != nil {
		return "", err
	}
	if err := tempFile.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", err
	}
	return finalPath, nil
}

func reserveDownloadPath(savePath string) (string, error) {
	if _, err := os.Stat(savePath); err != nil {
		if os.IsNotExist(err) {
			return savePath, nil
		}
		return "", err
	}

	dir := filepath.Dir(savePath)
	base := filepath.Base(savePath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

func contains(s, sub string) bool {
	return indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func lastIndexOf(s, sub string) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
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
