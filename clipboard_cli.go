package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func printClipboardUsage() {
	fmt.Println(`Usage:
  landrop clipboard watch [--target IP:Port] [--pin 1234] [--tls] [--interval 1500ms]

Watch the local system clipboard and push text changes to a remote LAN Drop server.`)
}

func runClipboardCommand(args []string) {
	if len(args) == 0 {
		printClipboardUsage()
		return
	}

	switch args[0] {
	case "watch":
		watchCmd := flag.NewFlagSet("clipboard watch", flag.ExitOnError)
		wPin := watchCmd.String("pin", "", "PIN protection code")
		wTLS := watchCmd.Bool("tls", false, "Use HTTPS")
		wTarget := watchCmd.String("target", "", "Directly connect to a specific device")
		wInterval := watchCmd.Duration("interval", 1500*time.Millisecond, "Clipboard polling interval")
		_ = watchCmd.Parse(args[1:])

		addr := *wTarget
		if addr == "" {
			var err error
			addr, err = promptForTargetAddress()
			if err != nil {
				log.Fatalf("failed to choose target device: %v", err)
			}
		}
		watchClipboard(addr, *wPin, *wTLS, *wInterval)
	default:
		fmt.Fprintf(os.Stderr, "unknown clipboard subcommand: %s\n", args[0])
		printClipboardUsage()
		os.Exit(1)
	}
}

func promptForTargetAddress() (string, error) {
	fmt.Println("Scanning LAN Drop devices for clipboard sync...")

	broker := NewSSEBroker()
	mdns := NewMDNSManager(0, broker, "")
	if err := mdns.Start(); err != nil {
		log.Printf("mDNS discovery unavailable: %v", err)
	}

	time.Sleep(3 * time.Second)
	devices := mdns.GetDevices()
	mdns.Stop()

	if len(devices) == 0 {
		fmt.Println("No devices discovered. Enter IP:Port manually:")
		var addr string
		fmt.Scanln(&addr)
		if addr == "" {
			return "", fmt.Errorf("empty target address")
		}
		return addr, nil
	}

	fmt.Printf("Found %d device(s):\n", len(devices))
	for i, d := range devices {
		fmt.Printf("  [%d] %s (%s)\n", i+1, d.Name, d.Addr)
	}
	fmt.Print("Choose device number: ")

	var choice int
	fmt.Scanln(&choice)
	if choice < 1 || choice > len(devices) {
		return "", fmt.Errorf("invalid choice")
	}
	return devices[choice-1].Addr, nil
}

func newHTTPClient(useTLS bool) *http.Client {
	client := &http.Client{}
	if useTLS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return client
}

func pushClipboardContent(client *http.Client, baseURL string, pin string, content string) error {
	body, err := json.Marshal(map[string]string{
		"content": content,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/clipboard/push", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if pin != "" {
		req.Header.Set("X-LanDrop-PIN", pin)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Error != "" {
			return fmt.Errorf("%s", apiErr.Error)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func watchClipboard(addr string, pin string, useTLS bool, interval time.Duration) {
	if interval <= 0 {
		interval = 1500 * time.Millisecond
	}

	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	baseURL := scheme + "://" + addr
	client := newHTTPClient(useTLS)

	fmt.Printf("Watching local clipboard and pushing changes to %s\n", baseURL)
	fmt.Println("Press Ctrl+C to stop.")

	lastSent := ""
	for {
		content, err := readClipboardText()
		if err != nil {
			log.Fatalf("failed to read local clipboard: %v", err)
		}

		if content != "" && content != lastSent {
			if len(content) > maxTextSize {
				fmt.Printf("[%s] skipped clipboard update larger than %s\n", time.Now().Format("15:04:05"), formatSize(maxTextSize))
				lastSent = content
			} else if err := pushClipboardContent(client, baseURL, pin, content); err != nil {
				fmt.Printf("[%s] clipboard push failed: %v\n", time.Now().Format("15:04:05"), err)
			} else {
				lastSent = content
				fmt.Printf("[%s] clipboard pushed (%s)\n", time.Now().Format("15:04:05"), formatSize(int64(len(content))))
			}
		}

		time.Sleep(interval)
	}
}
