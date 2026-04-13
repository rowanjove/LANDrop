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
	for port := startPort; port <= startPort+10; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err == nil {
			_ = ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("ports %d-%d are all in use", startPort, startPort+10)
}

func startServer(port int, pin string, useTLS bool, oneTimeUse bool) {
	actualPort, err := findAvailablePort(port)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	if actualPort != port {
		log.Printf("Port %d is busy, using %d instead", port, actualPort)
	}

	localIP := getLocalIP()
	addr := fmt.Sprintf("%s:%d", localIP, actualPort)
	listenAddr := fmt.Sprintf("0.0.0.0:%d", actualPort)

	app := NewApp(addr, pin)
	app.oneTimeUse = oneTimeUse
	app.mdns = NewMDNSManager(actualPort, app.broker, app.hostname)

	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	var handler http.Handler = mux
	if app.pin.IsEnabled() {
		handler = app.pin.Middleware(mux)
	}

	if err := app.mdns.Start(); err != nil {
		log.Printf("mDNS unavailable: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("\nCleaning up...")
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
			log.Fatalf("failed to generate certificate: %v", err)
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	url := fmt.Sprintf("%s://%s", scheme, addr)
	fmt.Println("==========================================")
	fmt.Printf("LAN Drop v%s\n", version)
	fmt.Printf("Device: %s\n", app.hostname)
	fmt.Printf("Address: %s\n", url)
	if app.pin.IsEnabled() {
		fmt.Printf("PIN: %s\n", pin)
	}
	if oneTimeUse {
		fmt.Println("Mode: one-time downloads enabled")
	}
	fmt.Println()
	fmt.Println("QR:")
	fmt.Println(generateQRASCII(url))
	fmt.Println("Open the address above in a browser, or scan the QR code.")
	if useTLS {
		fmt.Println("TLS tip: browsers will warn about the temporary self-signed certificate.")
		fmt.Println("Use mkcert for a trusted local certificate, or continue past the warning.")
	}
	fmt.Println("Press Ctrl+C to stop.")

	if useTLS {
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
