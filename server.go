package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	// Prefer private network IPs (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
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
		// Check if private network
		if ip4[0] == 192 && ip4[1] == 168 {
			return ip4.String() // Best: 192.168.x.x
		}
		if ip4[0] == 10 {
			return ip4.String() // Good: 10.x.x.x
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			if fallback == "" {
				fallback = ip4.String() // OK: 172.16-31.x.x (might be Docker/WSL)
			}
			continue
		}
		if fallback == "" {
			fallback = ip4.String()
		}
	}
	if fallback != "" {
		return fallback
	}
	return "127.0.0.1"
}

func findAvailablePort(startPort int) (int, error) {
	for port := startPort; port <= startPort+10; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("端口 %d-%d 均被占用", startPort, startPort+10)
}

func startServer(port int, pin string, useTLS bool, oneTimeUse bool) {
	actualPort, err := findAvailablePort(port)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	if actualPort != port {
		log.Printf("端口 %d 被占用，使用 %d", port, actualPort)
	}

	localIP := getLocalIP()
	addr := fmt.Sprintf("%s:%d", localIP, actualPort)
	listenAddr := fmt.Sprintf("0.0.0.0:%d", actualPort)

	app := NewApp(addr, pin)
	app.oneTimeUse = oneTimeUse
	app.mdns = NewMDNSManager(actualPort, app.broker)

	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	var handler http.Handler = mux
	if app.pin.IsEnabled() {
		handler = app.pin.Middleware(mux)
	}

	// Start mDNS
	if err := app.mdns.Start(); err != nil {
		log.Printf("mDNS 启动失败（设备发现不可用）: %v", err)
	}

	// Cleanup on exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("\n正在清理...")
		app.mdns.Stop()
		app.store.Cleanup()
		os.Exit(0)
	}()

	scheme := "http"
	server := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}

	if useTLS {
		scheme = "https"
		cert, err := generateSelfSignedCert()
		if err != nil {
			log.Fatalf("生成证书失败: %v", err)
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	url := fmt.Sprintf("%s://%s", scheme, addr)
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║           LAN Drop v1.1.0              ║")
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Printf("║  地址: %-31s ║\n", url)
	if app.pin.IsEnabled() {
		fmt.Printf("║  PIN:  %-31s ║\n", pin)
	}
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("二维码：")
	fmt.Println(generateQRASCII(url))
	fmt.Println("在浏览器中打开上面的地址，或用手机扫描二维码")
	fmt.Println("按 Ctrl+C 退出")

	if useTLS {
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
