package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
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

const version = "2.0.1"

type interruptedDownloadError struct {
	err error
}

func (e interruptedDownloadError) Error() string {
	return e.err.Error()
}

func (e interruptedDownloadError) Unwrap() error {
	return e.err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("未指定命令，正在启动 LANDrop 服务……")
		os.Args = append(os.Args, "serve")
	}

	LoadHistory()
	LoadConfig()

	port := flag.Int("port", 53217, "服务端口")
	pin := flag.String("pin", "", "4 位 PIN 保护")
	tlsFlag := flag.Bool("tls", false, "启用 HTTPS")

	switch os.Args[1] {
	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		setChineseFlagUsage(serveCmd, "landrop serve [选项]")
		sPort := serveCmd.Int("port", 53217, "服务端口")
		sPin := serveCmd.String("pin", "", "4 位 PIN 保护")
		sTLS := serveCmd.Bool("tls", false, "启用 HTTPS")
		sOneTime := serveCmd.Bool("one-time", false, "每个链接成功下载一次后失效")
		_ = serveCmd.Parse(os.Args[2:])
		startServer(*sPort, *sPin, *sTLS, *sOneTime)

	case "send":
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		setChineseFlagUsage(sendCmd, "landrop send [选项] <文件或目录...>")
		sPort := sendCmd.Int("port", 53217, "服务端口")
		sPin := sendCmd.String("pin", "", "4 位 PIN 保护")
		sTLS := sendCmd.Bool("tls", false, "启用 HTTPS")
		textMode := sendCmd.String("text", "", "发送纯文本")
		_ = sendCmd.Parse(os.Args[2:])

		if *textMode != "" {
			sendText(*sPort, *sPin, *sTLS, *textMode)
		} else if sendCmd.NArg() > 0 {
			sendFile(*sPort, *sPin, *sTLS, sendCmd.Args())
		} else {
			fmt.Println("用法：landrop send [--text '消息'] <文件或目录...>")
			os.Exit(1)
		}

	case "recv":
		recvCmd := flag.NewFlagSet("recv", flag.ExitOnError)
		setChineseFlagUsage(recvCmd, "landrop recv [保存目录] [选项]")
		rPin := recvCmd.String("pin", "", "4 位 PIN 保护")
		rTLS := recvCmd.Bool("tls", false, "启用 HTTPS")
		rCont := recvCmd.Bool("c", false, "断点续传")
		recvCmd.BoolVar(rCont, "continue", false, "断点续传")
		rTarget := recvCmd.String("target", "", "直接连接到指定设备")
		_ = recvCmd.Parse(os.Args[2:])

		saveDir := "."
		if recvCmd.NArg() > 0 {
			saveDir = recvCmd.Arg(0)
		}
		recvMode(saveDir, *rPin, *rTLS, *rCont, *rTarget)

	case "devices":
		listDevices()

	case "clipboard":
		runClipboardCommand(os.Args[2:])

	case "history":
		historyCmd := flag.NewFlagSet("history", flag.ExitOnError)
		setChineseFlagUsage(historyCmd, "landrop history [选项]")
		limit := historyCmd.Int("limit", 50, "显示最近 N 条记录")
		clear := historyCmd.Bool("clear", false, "清除本地传输历史")
		_ = historyCmd.Parse(os.Args[2:])

		if *clear {
			cleared := ClearHistory()
			fmt.Printf("已清除 %d 条传输记录\n", cleared)
			return
		}

		records := GetHistoryRecords()
		if len(records) == 0 {
			fmt.Println("暂无传输历史记录")
			return
		}

		showLimit := *limit
		if showLimit > len(records) {
			showLimit = len(records)
		}

		fmt.Printf("最近 %d 条传输记录：\n\n", showLimit)
		count := 0
		for i := len(records) - 1; i >= 0; i-- {
			record := records[i]
			ts := time.Unix(record.Timestamp, 0).Format("2006-01-02 15:04:05")
			statusIcon := "[OK]"
			switch record.Status {
			case "failed":
				statusIcon = "[ERR]"
			case "interrupted":
				statusIcon = "[INT]"
			}
			statusText := map[string]string{"success": "成功", "failed": "失败", "interrupted": "已中断"}[record.Status]
			if statusText == "" {
				statusText = record.Status
			}
			directionText := "接收"
			if record.Direction == "send" {
				directionText = "发送"
			}
			fmt.Printf("%s [%s] %s %s | %s (%s) | 对端：%s | 状态：%s\n",
				statusIcon, ts, directionText, record.Type, record.Name, formatSize(record.Size), record.Peer, statusText)
			count++
			if count >= showLimit {
				break
			}
		}

	case "version":
		fmt.Printf("LAN Drop 版本 %s\n", version)

	case "help", "--help", "-h":
		printUsage()

	default:
		if _, err := os.Stat(os.Args[1]); err == nil {
			flag.Parse()
			sendFile(*port, *pin, *tlsFlag, os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "未知命令：%s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`LAN Drop v%s

用法：
  landrop serve [选项]                         启动接收服务
  landrop send <文件或目录...>                 发送文件或目录
  landrop send --text "消息"                  发送纯文本
  landrop recv [保存目录] [选项]               接收传输内容
  landrop devices                              搜索局域网设备
  landrop clipboard watch [选项]              同步剪贴板
  landrop history [--limit N] [--clear]        查看或清除传输记录
  landrop version                              查看版本

常用选项：
  --port N                                     服务端口（默认 53217）
  --pin 1234                                   启用 4 位 PIN 保护
  --tls                                        启用 HTTPS
  --one-time                                   下载一次后链接失效

示例：
  landrop serve
  landrop serve --pin 1234 --tls --one-time
  landrop send ./report.pdf
  landrop send --text "来自 LAN Drop 的消息"
  landrop recv . --target 192.168.1.10:53217 --continue
  landrop clipboard watch --target 192.168.1.10:53217
`, version)
}

func setChineseFlagUsage(fs *flag.FlagSet, usage string) {
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "用法：%s\n\n选项：\n", usage)
		fs.PrintDefaults()
	}
}

func sendFile(port int, pin string, useTLS bool, files []string) {
	actualPort, err := findAvailablePort(port)
	if err != nil {
		log.Fatalf("启动失败：%v", err)
	}

	localIP := getLocalIP()
	addr := fmt.Sprintf("%s:%d", localIP, actualPort)

	app := NewApp(addr, pin)
	app.mdns = NewMDNSManager(actualPort, app.broker, app.hostname)

	scheme := "http"
	if useTLS {
		scheme = "https"
	}

	for _, filePath := range files {
		item, err := app.SendLocalFile(filePath)
		if err != nil {
			log.Printf("发送失败：%s - %v", filePath, err)
			continue
		}
		url := fmt.Sprintf("%s://%s/recv/%s", scheme, addr, item.Token)
		fmt.Printf("文件：%s（%s）\n", item.Name, formatSize(item.Size))
		fmt.Printf("下载地址：%s\n\n", url)
		fmt.Println(generateQRASCII(url))
	}

	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	var handler http.Handler = mux
	if app.pin.IsEnabled() {
		handler = app.pin.Middleware(mux)
	}

	server := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", actualPort),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	fmt.Println("等待对方下载……（按 Ctrl+C 停止）")
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
		log.Fatalf("启动失败：%v", err)
	}

	localIP := getLocalIP()
	addr := fmt.Sprintf("%s:%d", localIP, actualPort)

	app := NewApp(addr, pin)
	app.mdns = NewMDNSManager(actualPort, app.broker, app.hostname)

	item, err := app.store.AddText(text)
	if err != nil {
		log.Fatalf("准备文本失败：%v", err)
	}

	scheme := "http"
	if useTLS {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://%s/recv/%s", scheme, addr, item.Token)
	fmt.Printf("文本已就绪（%s）\n", formatSize(item.Size))
	fmt.Printf("获取地址：%s\n\n", url)
	fmt.Println(generateQRASCII(url))

	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	var handler http.Handler = mux
	if app.pin.IsEnabled() {
		handler = app.pin.Middleware(mux)
	}

	server := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", actualPort),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	fmt.Println("等待对方接收……（按 Ctrl+C 停止）")
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
	mdns := NewMDNSManager(0, broker, "")

	fmt.Println("正在搜索局域网中的 LAN Drop 设备……")
	if err := mdns.Start(); err != nil {
		log.Fatalf("mDNS 设备发现失败：%v", err)
	}

	time.Sleep(5 * time.Second)
	devices := mdns.GetDevices()
	mdns.Stop()

	if len(devices) == 0 {
		fmt.Println("没有发现 LAN Drop 设备")
		return
	}

	fmt.Printf("\n发现 %d 台设备：\n", len(devices))
	for _, d := range devices {
		status := "离线"
		if d.Online {
			status = "在线"
		}
		fmt.Printf("  %s（%s）- %s [%s]\n", d.Name, d.OS, d.Addr, status)
	}
}

func recvMode(saveDir string, pin string, useTLS bool, cont bool, target string) {
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		log.Fatalf("无法创建保存目录：%v", err)
	}
	absDir, _ := filepath.Abs(saveDir)
	fmt.Printf("接收内容将保存到：%s\n", absDir)

	addr := target
	if addr == "" {
		var err error
		addr, err = promptForTargetAddress()
		if err != nil {
			log.Fatalf("选择目标设备失败：%v", err)
		}
	}

	pollAndSave(addr, absDir, pin, useTLS, cont)
}

func historyStatusFromError(err error) string {
	if err == nil {
		return "success"
	}

	var interrupted interruptedDownloadError
	if errors.As(err, &interrupted) {
		return "interrupted"
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unexpected eof"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "closed network connection"),
		strings.Contains(msg, "context canceled"),
		strings.Contains(msg, "timeout"):
		return "interrupted"
	default:
		return "failed"
	}
}

func appendRecvHistory(name string, size int64, itemType string, status string, peer string) {
	AppendHistory(&HistoryRecord{
		Direction: "recv",
		Name:      name,
		Size:      size,
		Type:      itemType,
		Status:    status,
		Peer:      peer,
	})
}

func pollAndSave(addr string, saveDir string, pin string, useTLS bool, cont bool) {
	fmt.Printf("正在连接 %s，等待接收内容……\n", addr)

	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	baseURL := scheme + "://" + addr
	client := newHTTPClient(useTLS)

	req, _ := http.NewRequest(http.MethodGet, baseURL+"/events", nil)
	if pin != "" {
		req.Header.Set("X-LanDrop-PIN", pin)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("连接失败：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("连接失败：HTTP %d（可能需要 PIN）", resp.StatusCode)
	}

	fmt.Println("连接成功，等待传输……（按 Ctrl+C 停止）")

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	var eventType, eventData string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if eventType == "file_ready" && eventData != "" {
				var event struct {
					Token   string `json:"token"`
					Name    string `json:"name"`
					Size    int64  `json:"size"`
					Type    string `json:"type"`
					OneTime bool   `json:"one_time"`
				}
				if err := json.Unmarshal([]byte(eventData), &event); err == nil {
					switch event.Type {
					case "text":
						textReq, _ := http.NewRequest(http.MethodGet, baseURL+"/recv/"+event.Token, nil)
						if pin != "" {
							textReq.Header.Set("X-LanDrop-PIN", pin)
						}

						textResp, err := client.Do(textReq)
						if err != nil {
							status := historyStatusFromError(err)
							fmt.Printf("文本接收失败：%v\n", err)
							appendRecvHistory(event.Name, event.Size, "text", status, addr)
							goto resetEvent
						}

						if textResp.StatusCode != http.StatusOK {
							var apiErr struct {
								Error string `json:"error"`
							}
							_ = json.NewDecoder(textResp.Body).Decode(&apiErr)
							_ = textResp.Body.Close()
							msg := apiErr.Error
							if msg == "" {
								msg = fmt.Sprintf("HTTP %d", textResp.StatusCode)
							}
							fmt.Printf("文本接收失败：%s\n", msg)
							appendRecvHistory(event.Name, event.Size, "text", "failed", addr)
							goto resetEvent
						}

						var textData struct {
							Content string `json:"content"`
						}
						if err := json.NewDecoder(textResp.Body).Decode(&textData); err != nil {
							_ = textResp.Body.Close()
							status := historyStatusFromError(err)
							fmt.Printf("解析文本响应失败：%v\n", err)
							appendRecvHistory(event.Name, event.Size, "text", status, addr)
							goto resetEvent
						}
						_ = textResp.Body.Close()

						fmt.Printf("\n收到文本（%s）：\n%s\n", formatSize(event.Size), textData.Content)
						appendRecvHistory(event.Name, event.Size, "text", "success", addr)

					default:
						fmt.Printf("\n收到文件：%s（%s）\n", event.Name, formatSize(event.Size))
						savePath := filepath.Join(saveDir, sanitizeFilename(event.Name))
						dlReq, _ := http.NewRequest(http.MethodGet, baseURL+"/recv/"+event.Token, nil)
						if pin != "" {
							dlReq.Header.Set("X-LanDrop-PIN", pin)
						}

						var startByte int64
						if cont {
							if info, err := os.Stat(savePath); err == nil {
								switch {
								case info.Size() > event.Size:
									fmt.Printf("已拒绝断点续传：本地文件大于远端文件：%s\n", savePath)
									appendRecvHistory(event.Name, event.Size, "file", "failed", addr)
									goto resetEvent
								case info.Size() == event.Size:
									fmt.Printf("文件已完整存在，跳过下载：%s\n", savePath)
									appendRecvHistory(event.Name, event.Size, "file", "success", addr)
									goto resetEvent
								default:
									startByte = info.Size()
									dlReq.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
								}
							}
						}

						savePath, err = downloadFileWithClient(client, dlReq, savePath, startByte)
						if err != nil {
							status := historyStatusFromError(err)
							fmt.Printf("下载失败：%v\n", err)
							appendRecvHistory(event.Name, event.Size, "file", status, addr)
						} else {
							fmt.Printf("已保存到：%s\n", savePath)
							appendRecvHistory(event.Name, event.Size, "file", "success", addr)
						}
					}
				}
			}

		resetEvent:
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
		log.Printf("事件流已中断：%v", err)
	}
}

func downloadFile(url string, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	_, err = saveDownload(resp.Body, savePath, false)
	return err
}

func downloadFileWithClient(client *http.Client, req *http.Request, savePath string, resumeOffset int64) (string, error) {
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resumeOffset > 0 {
		if resp.StatusCode != http.StatusPartialContent {
			return "", fmt.Errorf("server did not honor resume request: HTTP %d", resp.StatusCode)
		}
		start, _, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if !ok {
			return "", fmt.Errorf("server returned invalid Content-Range")
		}
		if start != resumeOffset {
			return "", fmt.Errorf("resume offset mismatch: got %d want %d", start, resumeOffset)
		}
	} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return saveDownload(resp.Body, savePath, resumeOffset > 0)
}

func saveDownload(src io.Reader, savePath string, isResume bool) (string, error) {
	finalPath := savePath
	if !isResume {
		var err error
		finalPath, err = reserveDownloadPath(savePath)
		if err != nil {
			return "", err
		}
	}

	var (
		tempFile *os.File
		err      error
		tempPath string
	)

	if isResume {
		tempFile, err = os.OpenFile(finalPath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return "", err
		}
		tempPath = finalPath
	} else {
		tempFile, err = os.CreateTemp(filepath.Dir(finalPath), filepath.Base(finalPath)+".part-*")
		if err != nil {
			return "", err
		}
		tempPath = tempFile.Name()
	}

	defer func() {
		_ = tempFile.Close()
		if !isResume {
			_ = os.Remove(tempPath)
		}
	}()

	written, err := io.Copy(tempFile, src)
	if err != nil {
		if written > 0 {
			return "", interruptedDownloadError{err: err}
		}
		return "", err
	}
	if err := tempFile.Close(); err != nil {
		return "", err
	}

	if !isResume {
		if err := os.Rename(tempPath, finalPath); err != nil {
			return "", err
		}
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
