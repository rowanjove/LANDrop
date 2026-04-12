package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//go:embed web/index.html
var webUI embed.FS

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "darwin"
	default:
		return "linux"
	}
}

type App struct {
	store     *TransferStore
	broker    *SSEBroker
	mdns      *MDNSManager
	pin       *PINManager
	clipboard *ClipboardManager
	addr      string
	hostname  string
}

func NewApp(addr string, pin string) *App {
	broker := NewSSEBroker()
	hostname, _ := os.Hostname()
	return &App{
		store:     NewTransferStore(),
		broker:    broker,
		pin:       NewPINManager(pin),
		clipboard: NewClipboardManager(broker),
		addr:      addr,
		hostname:  hostname,
	}
}

func (a *App) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", a.handleIndex)
	mux.HandleFunc("GET /info", a.handleInfo)
	mux.HandleFunc("GET /qr", a.handleQR)
	mux.HandleFunc("POST /send/file", a.handleSendFile)
	mux.HandleFunc("POST /send/text", a.handleSendText)
	mux.HandleFunc("GET /recv/{token}", a.handleRecv)
	mux.HandleFunc("GET /devices", a.handleDevices)
	mux.HandleFunc("GET /events", a.handleEvents)
	mux.HandleFunc("POST /clipboard/push", a.clipboard.HandlePush)
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := webUI.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "Web UI not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (a *App) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"name":    a.hostname,
		"version": "1.0.0",
		"os":      getOS(),
		"addr":    a.addr,
	})
}

func (a *App) handleQR(w http.ResponseWriter, r *http.Request) {
	handleQR(w, r, a.addr)
}

func (a *App) handleSendFile(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form, max 500MB
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
			"error": "文件过大，单次限制 500 MB",
			"code":  "FILE_TOO_LARGE",
		})
		return
	}

	// Check for multiple files
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "no file provided",
			"code":  "BAD_REQUEST",
		})
		return
	}

	if len(files) == 1 {
		// Single file
		fh := files[0]
		if fh.Size > maxFileSize {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
				"error": "文件过大，单次限制 500 MB",
				"code":  "FILE_TOO_LARGE",
			})
			return
		}
		f, err := fh.Open()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "保存失败",
				"code":  "UPLOAD_FAILED",
			})
			return
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "保存失败",
				"code":  "UPLOAD_FAILED",
			})
			return
		}

		item, err := a.store.AddFile(fh.Filename, data, fh.Size)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "保存失败，请检查磁盘空间",
				"code":  "UPLOAD_FAILED",
			})
			return
		}

		a.broker.Broadcast("file_ready", map[string]interface{}{
			"token": item.Token,
			"name":  item.Name,
			"size":  item.Size,
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"token":      item.Token,
			"name":       item.Name,
			"size":       item.Size,
			"expires_at": item.ExpiresAt,
		})
		return
	}

	// Multiple files: zip them
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	var totalSize int64

	for _, fh := range files {
		totalSize += fh.Size
		if totalSize > maxFileSize {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
				"error": "文件总大小超过 500 MB",
				"code":  "FILE_TOO_LARGE",
			})
			return
		}

		f, err := fh.Open()
		if err != nil {
			continue
		}

		w, err := zipWriter.Create(fh.Filename)
		if err != nil {
			f.Close()
			continue
		}
		io.Copy(w, f)
		f.Close()
	}
	zipWriter.Close()

	zipName := fmt.Sprintf("landrop_%s.zip", time.Now().Format("20060102_150405"))
	data := buf.Bytes()
	item, err := a.store.AddFile(zipName, data, int64(len(data)))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "保存失败",
			"code":  "UPLOAD_FAILED",
		})
		return
	}

	a.broker.Broadcast("file_ready", map[string]interface{}{
		"token": item.Token,
		"name":  item.Name,
		"size":  item.Size,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      item.Token,
		"name":       item.Name,
		"size":       item.Size,
		"expires_at": item.ExpiresAt,
	})
}

func (a *App) handleSendText(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "invalid request",
			"code":  "BAD_REQUEST",
		})
		return
	}

	if len(req.Content) > maxTextSize {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
			"error": "文本过长，建议改用文件传输",
			"code":  "TEXT_TOO_LARGE",
		})
		return
	}

	item, err := a.store.AddText(req.Content)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
			"code":  "UPLOAD_FAILED",
		})
		return
	}

	a.broker.Broadcast("file_ready", map[string]interface{}{
		"token": item.Token,
		"name":  "",
		"size":  item.Size,
		"type":  "text",
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":  item.Token,
		"length": item.Size,
	})
}

func (a *App) handleRecv(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "missing token",
			"code":  "BAD_REQUEST",
		})
		return
	}

	item, ok := a.store.Get(token)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "token not found",
			"code":  "TOKEN_NOT_FOUND",
		})
		return
	}

	if item.OneTimeUse && item.Downloaded {
		writeJSON(w, http.StatusGone, map[string]interface{}{
			"error": "该内容已被取走",
			"code":  "TOKEN_USED",
		})
		return
	}

	if item.Type == "text" {
		a.store.MarkDownloaded(token)
		a.broker.Broadcast("done", map[string]string{"token": token})
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"content": string(item.Content),
			"type":    "text",
		})
		return
	}

	// File download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, item.Name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", item.Size))

	if item.Content != nil {
		// Stream from memory with progress reporting
		reader := bytes.NewReader(item.Content)
		written := int64(0)
		buf := make([]byte, 32*1024)
		lastReport := time.Now()

		for {
			n, err := reader.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				written += int64(n)
				if time.Since(lastReport) > 200*time.Millisecond {
					a.broker.Broadcast("progress", map[string]interface{}{
						"token": token,
						"bytes": written,
						"total": item.Size,
						"speed": float64(written) / time.Since(lastReport).Seconds(),
					})
					lastReport = time.Now()
				}
			}
			if err != nil {
				break
			}
		}
	} else if item.FilePath != "" {
		f, err := os.Open(item.FilePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "file read error",
			})
			return
		}
		defer f.Close()
		io.Copy(w, f)
	}

	a.store.MarkDownloaded(token)
	a.broker.Broadcast("done", map[string]string{"token": token})
}

func (a *App) handleDevices(w http.ResponseWriter, r *http.Request) {
	devices := a.mdns.GetDevices()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"devices": devices,
	})
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	a.broker.ServeHTTP(w, r)
}

// CLI helper: send file from local path
func (a *App) SendLocalFile(filePath string) (*TransferItem, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return a.sendDirectory(absPath, info.Name())
	}
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("文件过大，单次限制 500 MB")
	}
	return a.store.AddFileFromPath(info.Name(), absPath, info.Size())
}

func (a *App) sendDirectory(dirPath string, dirName string) (*TransferItem, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(dirPath, path)
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		w, err := zipWriter.Create(filepath.Join(dirName, relPath))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		return nil, err
	}
	zipWriter.Close()

	zipName := fmt.Sprintf("%s.zip", dirName)
	data := buf.Bytes()
	return a.store.AddFile(zipName, data, int64(len(data)))
}
