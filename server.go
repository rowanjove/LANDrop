package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func initLogging() {
	logDir := getStateDir()
	logPath := filepath.Join(logDir, "landrop.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		if info, statErr := logFile.Stat(); statErr == nil && info.Size() > 10*1024*1024 {
			_ = logFile.Close()
			_ = os.Remove(logPath + ".1")
			_ = os.Rename(logPath, logPath+".1")
			logFile, _ = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		}
		if logFile != nil {
			log.SetOutput(io.MultiWriter(os.Stderr, logFile))
		}
	}
}

func isVirtualInterface(name string) bool {
	lower := strings.ToLower(name)
	virtualKeywords := []string{
		"vethernet", "vmware", "vmnet", "virtualbox", "vbox",
		"docker", "container", "cni", "flannel", "calico",
		"tailscale", "zerotier", "tap", "tun", "wsl", "hyper-v", "npcap",
	}
	for _, kw := range virtualKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func GetAllLocalIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{"127.0.0.1"}
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 != nil {
				ips = append(ips, ip4.String())
			}
		}
	}
	if len(ips) == 0 {
		return []string{"127.0.0.1"}
	}
	return ips
}

func getLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	var physicalIPs []string
	var virtualIPs []string

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		isVirtual := isVirtualInterface(iface.Name)
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}
			if isVirtual {
				virtualIPs = append(virtualIPs, ip4.String())
			} else {
				physicalIPs = append(physicalIPs, ip4.String())
			}
		}
	}

	chooseBest := func(candidates []string) string {
		var fallback string
		for _, ipStr := range candidates {
			ip := net.ParseIP(ipStr).To4()
			if ip == nil {
				continue
			}
			switch {
			case ip[0] == 192 && ip[1] == 168:
				return ipStr
			case ip[0] == 10:
				return ipStr
			case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
				if fallback == "" {
					fallback = ipStr
				}
			case fallback == "":
				fallback = ipStr
			}
		}
		return fallback
	}

	if bestPhysical := chooseBest(physicalIPs); bestPhysical != "" {
		return bestPhysical
	}
	if bestVirtual := chooseBest(virtualIPs); bestVirtual != "" {
		return bestVirtual
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

func startServer(port int, pin string, useTLS bool, oneTimeUse bool, enableTray bool, hideConsole bool) {
	if err := EnsureSingleInstance(port); err != nil {
		log.Printf("单实例检查提示：%v", err)
	}
	defer ReleaseSingleInstance()

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

	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s", scheme, addr)

	var removeTray func()
	exitCleanup := func() {
		cleanupTicker.Stop()
		if removeTray != nil {
			removeTray()
		}
		ReleaseSingleInstance()
		app.mdns.Stop()
		app.store.Cleanup()
	}
	app.SetOnExit(exitCleanup)

	if enableTray {
		var trayErr error
		removeTray, trayErr = StartTray(url, app.hostname, func() {
			log.Println("\n正在通过系统托盘退出……")
			exitCleanup()
			os.Exit(0)
		})
		if trayErr == nil && hideConsole {
			HideConsole()
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("\n正在清理并退出……")
		exitCleanup()
		os.Exit(0)
	}()
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
	allIPs := GetAllLocalIPs()
	if len(allIPs) > 1 {
		fmt.Printf("备用局域网地址：\n")
		for _, ip := range allIPs {
			if ip != localIP {
				fmt.Printf("  - %s://%s:%d\n", scheme, ip, actualPort)
			}
		}
	}
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
	if enableTray && runtime.GOOS == "windows" {
		fmt.Println("提示：系统托盘已就绪，可在任务栏右下角托盘图标右键随时选择「显示/隐藏控制台」。")
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
