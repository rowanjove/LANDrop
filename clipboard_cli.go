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
	fmt.Println(`用法：
  landrop clipboard watch [--target IP:Port] [--pin 1234] [--tls] [--interval 1500ms]

监视本机剪贴板，将文本变化同步到远端 LAN Drop 服务。`)
}

func runClipboardCommand(args []string) {
	if len(args) == 0 {
		printClipboardUsage()
		return
	}

	switch args[0] {
	case "watch":
		watchCmd := flag.NewFlagSet("clipboard watch", flag.ExitOnError)
		setChineseFlagUsage(watchCmd, "landrop clipboard watch [选项]")
		wPin := watchCmd.String("pin", "", "PIN 保护码")
		wTLS := watchCmd.Bool("tls", false, "启用 HTTPS")
		wTarget := watchCmd.String("target", "", "直接连接到指定设备")
		wInterval := watchCmd.Duration("interval", 1500*time.Millisecond, "剪贴板轮询间隔")
		_ = watchCmd.Parse(args[1:])

		addr := *wTarget
		if addr == "" {
			var err error
			addr, err = promptForTargetAddress()
			if err != nil {
				log.Fatalf("选择目标设备失败：%v", err)
			}
		}
		watchClipboard(addr, *wPin, *wTLS, *wInterval)
	default:
		fmt.Fprintf(os.Stderr, "未知的剪贴板子命令：%s\n", args[0])
		printClipboardUsage()
		os.Exit(1)
	}
}

func promptForTargetAddress() (string, error) {
	fmt.Println("正在搜索用于剪贴板同步的 LAN Drop 设备……")

	broker := NewSSEBroker()
	mdns := NewMDNSManager(0, broker, "")
	if err := mdns.Start(); err != nil {
		log.Printf("mDNS 设备发现不可用：%v", err)
	}

	time.Sleep(3 * time.Second)
	devices := mdns.GetDevices()
	mdns.Stop()

	if len(devices) == 0 {
		fmt.Println("没有发现设备，请手动输入 IP:端口：")
		var addr string
		fmt.Scanln(&addr)
		if addr == "" {
			return "", fmt.Errorf("目标地址为空")
		}
		return addr, nil
	}

	fmt.Printf("发现 %d 台设备：\n", len(devices))
	for i, d := range devices {
		fmt.Printf("  [%d] %s (%s)\n", i+1, d.Name, d.Addr)
	}
	fmt.Print("请选择设备编号：")

	var choice int
	fmt.Scanln(&choice)
	if choice < 1 || choice > len(devices) {
		return "", fmt.Errorf("设备编号无效")
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

	fmt.Printf("正在监视本机剪贴板，并同步到 %s\n", baseURL)
	fmt.Println("按 Ctrl+C 停止。")

	lastSent := ""
	for {
		content, err := readClipboardText()
		if err != nil {
			log.Fatalf("读取本机剪贴板失败：%v", err)
		}

		if content != "" && content != lastSent {
			if len(content) > maxTextSize {
				fmt.Printf("[%s] 已跳过超过 %s 的剪贴板内容\n", time.Now().Format("15:04:05"), formatSize(maxTextSize))
				lastSent = content
			} else if err := pushClipboardContent(client, baseURL, pin, content); err != nil {
				fmt.Printf("[%s] 剪贴板同步失败：%v\n", time.Now().Format("15:04:05"), err)
			} else {
				lastSent = content
				fmt.Printf("[%s] 剪贴板已同步（%s）\n", time.Now().Format("15:04:05"), formatSize(int64(len(content))))
			}
		}

		time.Sleep(interval)
	}
}
