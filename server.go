package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	var fallback string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil {
			continue
		}

		switch {
		case ip4[0] == 192 && ip4[1] == 168:
			return ip4.String()
		case ip4[0] == 10:
			return ip4.String()
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			if fallback == "" {
				fallback = ip4.String()
			}
		case fallback == "":
			fallback = ip4.String()
		}
	}

	if fallback != "" {
		return fallback
	}
	return "127.0.0.1"
}

func findAvailablePort(startPort int) (int, error) {
	ln, port, err := listenAvailablePort(startPort)
	if err != nil {
		return 0, err
	}
	_ = ln.Close()
	return port, nil
}

// listenAvailablePort reserves the first free port in the configured range.
// Keeping the listener open until the HTTP server starts avoids a probe/listen
// race where another process can claim the port between the two operations.
func listenAvailablePort(startPort int) (net.Listener, int, error) {
	for port := startPort; port <= startPort+10; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err == nil {
			return ln, port, nil
		}
	}
	return nil, 0, fmt.Errorf("端口 %d-%d 均已被占用", startPort, startPort+10)
}

func browserCommand(rawURL string) (string, []string, error) {
	if rawURL == "" {
		return "", nil, fmt.Errorf("浏览器地址为空")
	}

	switch runtime.GOOS {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", rawURL}, nil
	case "darwin":
		return "open", []string{rawURL}, nil
	case "linux", "freebsd", "openbsd", "netbsd":
		return "xdg-open", []string{rawURL}, nil
	default:
		return "", nil, fmt.Errorf("当前系统不支持自动打开浏览器")
	}
}

func openBrowser(rawURL string) error {
	command, args, err := browserCommand(rawURL)
	if err != nil {
		return err
	}
	return exec.Command(command, args...).Start()
}

func startServer(port int, pin string, useTLS bool, oneTimeUse bool) {
	listener, actualPort, err := listenAvailablePort(port)
	if err != nil {
		log.Fatalf("启动失败：%v", err)
	}
	if actualPort != port {
		log.Printf("端口 %d 已被占用，改用端口 %d", port, actualPort)
	}

	localIP := getLocalIP()
	addr := fmt.Sprintf("%s:%d", localIP, actualPort)
	listenAddr := fmt.Sprintf("0.0.0.0:%d", actualPort)

	app := NewApp(addr, pin)
	app.oneTimeUse = oneTimeUse
	app.mdns = NewMDNSManager(actualPort, app.broker, app.hostname)
	mdnsScheme := "http"
	if useTLS {
		mdnsScheme = "https"
	}
	app.mdns.SetScheme(mdnsScheme)

	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	var handler http.Handler = mux
	if app.pin.IsEnabled() {
		handler = app.pin.Middleware(mux)
	}

	if err := app.mdns.Start(); err != nil {
		log.Printf("mDNS 服务不可用：%v", err)
	}

	// Background TTL cleanup worker (Phase 19)
	cleanupTicker := time.NewTicker(30 * time.Second)
	go func() {
		for range cleanupTicker.C {
			cleaned := app.store.CleanupExpired()
			if cleaned > 0 {
				log.Printf("已清理 %d 个过期传输", cleaned)
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cleanupTicker.Stop()
		log.Println("\n正在清理并退出……")
		app.mdns.Stop()
		app.store.Cleanup()
		os.Exit(0)
	}()

	scheme := "http"
	server := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	if useTLS {
		scheme = "https"
		cert, err := generateSelfSignedCert()
		if err != nil {
			log.Fatalf("生成 HTTPS 证书失败：%v", err)
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	url := fmt.Sprintf("%s://%s", scheme, addr)
	serveErr := make(chan error, 1)
	if useTLS {
		go func() { serveErr <- server.Serve(tls.NewListener(listener, server.TLSConfig)) }()
	} else {
		go func() { serveErr <- server.Serve(listener) }()
	}

	fmt.Println("==========================================")
	fmt.Printf("LAN Drop v%s\n", version)
	fmt.Printf("设备：%s\n", app.hostname)
	fmt.Printf("访问地址：%s\n", url)
	if app.pin.IsEnabled() {
		fmt.Printf("PIN：%s\n", pin)
	}
	if oneTimeUse {
		fmt.Println("模式：下载链接仅可使用一次")
	}
	fmt.Println()
	fmt.Println("二维码：")
	fmt.Println(generateQRASCII(url))
	fmt.Println("正在打开浏览器；也可以手动访问上方地址或扫描二维码。")
	if err := openBrowser(url); err != nil {
		log.Printf("自动打开浏览器失败：%v；请手动访问：%s", err, url)
	}
	if useTLS {
		fmt.Println("HTTPS 提示：临时自签名证书可能触发浏览器警告。")
		fmt.Println("可使用 mkcert 生成受信任的本地证书，或在浏览器中继续访问。")
	}
	fmt.Println("按 Ctrl+C 停止服务。")

	if err := <-serveErr; err != nil {
		log.Fatal(err)
	}
}
