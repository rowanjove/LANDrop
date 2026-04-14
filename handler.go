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

//go:embed web/*
var webUI embed.FS

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
	store      *TransferStore
	broker     *SSEBroker
	mdns       *MDNSManager
	pin        *PINManager
	clipboard  *ClipboardManager
	addr       string
	hostname   string
	systemName string
	config     *ConfigStore
	oneTimeUse bool // Token one-time-use mode (SEC-02)
}

func NewApp(addr string, pin string) *App {
	broker := NewSSEBroker()
	hostname, _ := os.Hostname()
	hostname = normalizeDeviceName(hostname)
	if hostname == "" {
		hostname = "LAN Drop"
	}
	config := appConfig
	deviceName := config.DeviceName(hostname)
	return &App{
		store:      NewTransferStore(),
		broker:     broker,
		pin:        NewPINManager(pin),
		clipboard:  NewClipboardManager(broker),
		addr:       addr,
		hostname:   deviceName,
		systemName: hostname,
		config:     config,
	}
}

func (a *App) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", a.handleIndex)
	mux.HandleFunc("GET /info", a.handleInfo)
	mux.HandleFunc("GET /settings", a.handleSettings)
	mux.HandleFunc("POST /settings", a.handleSettings)
	mux.HandleFunc("GET /qr", a.handleQR)
	mux.HandleFunc("POST /send/file", a.handleSendFile)
	mux.HandleFunc("POST /send/text", a.handleSendText)
	mux.HandleFunc("GET /preview/{token}", a.handlePreview)
	mux.HandleFunc("GET /recv/{token}", a.handleRecv)
	mux.HandleFunc("GET /devices", a.handleDevices)
	mux.HandleFunc("GET /events", a.handleEvents)
	mux.HandleFunc("POST /clipboard/push", a.clipboard.HandlePush)
	mux.HandleFunc("GET /clipboard", a.clipboard.HandleGet)
	mux.HandleFunc("GET /history", a.handleHistory)
	mux.HandleFunc("DELETE /history", a.handleClearHistory)
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	page := "web/desktop.html"
	data, err := webUI.ReadFile(page)
	if err != nil {
		http.Error(w, "Web UI not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (a *App) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":     a.hostname,
		"version":  version,
		"os":       getOS(),
		"addr":     a.addr,
		"one_time": a.oneTimeUse,
	})
}

func (a *App) handleQR(w http.ResponseWriter, r *http.Request) {
	handleQR(w, r, a.addr)
}

func (a *App) handleSendFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize+(64<<20))
	// Parse multipart form, keep up to 32 MB in memory, rest goes to temp files.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
			"error": "File too large. Max 2 GB per transfer.",
			"code":  "FILE_TOO_LARGE",
		})
		return
	}
	defer r.MultipartForm.RemoveAll()

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
				"error": "File too large. Max 2 GB per transfer.",
				"code":  "FILE_TOO_LARGE",
			})
			return
		}
		f, err := fh.Open()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store upload.",
				"code":  "UPLOAD_FAILED",
			})
			return
		}
		defer f.Close()

		item, err := a.store.AddFileFromReader(fh.Filename, f, fh.Size)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store upload.",
				"code":  "UPLOAD_FAILED",
			})
			return
		}

		item.OneTimeUse = a.oneTimeUse
		a.broker.Broadcast("file_ready", map[string]interface{}{
			"token":    item.Token,
			"name":     item.Name,
			"size":     item.Size,
			"type":     "file",
			"sender":   r.Header.Get("X-Client-ID"),
			"one_time": item.OneTimeUse,
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
	tmpFile, err := os.CreateTemp(a.store.tempDir, "upload-*.zip")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "upload failed",
			"code":  "UPLOAD_FAILED",
		})
		return
	}
	tmpPath := tmpFile.Name()
	cleanupTemp := true
	defer func() {
		_ = tmpFile.Close()
		if cleanupTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	zipWriter := zip.NewWriter(tmpFile)
	var totalSize int64

	for _, fh := range files {
		totalSize += fh.Size
		if totalSize > maxFileSize {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
				"error": "Total file size exceeds 2 GB.",
				"code":  "FILE_TOO_LARGE",
			})
			return
		}

		f, err := fh.Open()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store upload.",
				"code":  "UPLOAD_FAILED",
			})
			return
		}

		zw, err := zipWriter.Create(sanitizeFilename(fh.Filename))
		if err != nil {
			f.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store upload.",
				"code":  "UPLOAD_FAILED",
			})
			return
		}
		if _, err := io.Copy(zw, f); err != nil {
			f.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store upload.",
				"code":  "UPLOAD_FAILED",
			})
			return
		}
		f.Close()
	}
	if err := zipWriter.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to store upload.",
			"code":  "UPLOAD_FAILED",
		})
		return
	}

	if err := tmpFile.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "upload failed",
			"code":  "UPLOAD_FAILED",
		})
		return
	}

	zipName := fmt.Sprintf("landrop_%s.zip", time.Now().Format("20060102_150405"))
	info, err := os.Stat(tmpPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "upload failed",
			"code":  "UPLOAD_FAILED",
		})
		return
	}
	item, err := a.store.AddTempFile(zipName, tmpPath, info.Size())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to store upload.",
			"code":  "UPLOAD_FAILED",
		})
		return
	}
	cleanupTemp = false
	item.OneTimeUse = a.oneTimeUse

	a.broker.Broadcast("file_ready", map[string]interface{}{
		"token":    item.Token,
		"name":     item.Name,
		"size":     item.Size,
		"type":     "file",
		"sender":   r.Header.Get("X-Client-ID"),
		"one_time": item.OneTimeUse,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      item.Token,
		"name":       item.Name,
		"size":       item.Size,
		"expires_at": item.ExpiresAt,
	})
}

func (a *App) handleSendText(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxTextSize+1024) // 10MB + overhead
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
			"error": "Text too large. Use file transfer instead.",
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
	item.OneTimeUse = a.oneTimeUse

	a.broker.Broadcast("file_ready", map[string]interface{}{
		"token":    item.Token,
		"name":     "",
		"size":     item.Size,
		"type":     "text",
		"sender":   r.Header.Get("X-Client-ID"),
		"one_time": item.OneTimeUse,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":  item.Token,
		"length": item.Size,
	})
}

func (a *App) handlePreview(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	item, ok := a.store.Get(token)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "not found"})
		return
	}
	if item.Type != "text" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"type": "file", "name": item.Name, "size": item.Size, "one_time": item.OneTimeUse,
		})
		return
	}
	content := string(item.Content)
	if len(content) > 500 {
		content = content[:500]
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"type": "text", "preview": content, "size": item.Size, "one_time": item.OneTimeUse,
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

	if r.Method == http.MethodHead {
		item, ok := a.store.Get(token)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]interface{}{
				"error": "token not found",
				"code":  "TOKEN_NOT_FOUND",
			})
			return
		}
		if item.Type == "text" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return
		}
		if _, err := a.serveFile(w, r, item); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "file read error",
			})
		}
		return
	}

	item, found, unavailable := a.store.BeginDownload(token)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "token not found",
			"code":  "TOKEN_NOT_FOUND",
		})
		return
	}

	if unavailable {
		writeJSON(w, http.StatusGone, map[string]interface{}{
			"error": "This item has already been claimed.",
			"code":  "TOKEN_USED",
		})
		return
	}

	outcome := DownloadFailed
	defer func() {
		a.store.FinishDownload(token, outcome, r.RemoteAddr)
		if outcome == DownloadCompleted {
			a.broker.Broadcast("done", map[string]string{"token": token})
		}
	}()

	if item.Type == "text" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"content": string(item.Content),
			"type":    "text",
		})
		if err == nil {
			outcome = DownloadCompleted
		}
		return
	}

	fileOutcome, err := a.serveFile(w, r, item)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "file read error",
		})
		return
	}
	outcome = fileOutcome
}

type trackingResponseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
	writeErr     error
}

func (w *trackingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *trackingResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += int64(n)
	if err != nil {
		w.writeErr = err
	}
	return n, err
}

func (w *trackingResponseWriter) Flush() {
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (w *trackingResponseWriter) StatusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (a *App) serveFile(w http.ResponseWriter, r *http.Request, item *TransferItem) (DownloadOutcome, error) {
	var (
		reader  io.ReadSeeker
		modTime = time.Unix(item.CreatedAt, 0)
		closeFn func() error
	)

	if item.Content != nil {
		reader = bytes.NewReader(item.Content)
	} else if item.FilePath != "" {
		f, err := os.Open(item.FilePath)
		if err != nil {
			return DownloadFailed, err
		}
		info, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return DownloadFailed, err
		}
		reader = f
		modTime = info.ModTime()
		closeFn = f.Close
	} else {
		return DownloadFailed, fmt.Errorf("empty file source")
	}

	if closeFn != nil {
		defer closeFn()
	}

	tw := &trackingResponseWriter{ResponseWriter: w}
	tw.Header().Set("Content-Type", "application/octet-stream")
	tw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, item.Name))
	http.ServeContent(tw, r, item.Name, modTime, reader)
	return classifyDownloadOutcome(item.Size, tw), nil
}

func classifyDownloadOutcome(totalSize int64, w *trackingResponseWriter) DownloadOutcome {
	if w.writeErr != nil {
		if w.bytesWritten > 0 {
			return DownloadInterrupted
		}
		return DownloadFailed
	}

	switch status := w.StatusCode(); status {
	case http.StatusOK:
		if totalSize == 0 || w.bytesWritten == totalSize {
			return DownloadCompleted
		}
		if w.bytesWritten > 0 {
			return DownloadInterrupted
		}
		return DownloadFailed
	case http.StatusPartialContent:
		start, end, ok := parseContentRange(w.Header().Get("Content-Range"))
		if !ok {
			return DownloadFailed
		}
		if start >= 0 && end == totalSize-1 {
			return DownloadCompleted
		}
		return DownloadReleased
	case http.StatusNotModified, http.StatusRequestedRangeNotSatisfiable:
		return DownloadReleased
	default:
		if status >= 400 {
			return DownloadFailed
		}
	}
	return DownloadReleased
}

func parseContentRange(header string) (int64, int64, bool) {
	if header == "" {
		return 0, 0, false
	}

	var (
		unit       string
		start, end int64
		total      int64
	)
	n, err := fmt.Sscanf(header, "%s %d-%d/%d", &unit, &start, &end, &total)
	if err != nil || n != 4 || unit != "bytes" {
		return 0, 0, false
	}
	return start, end, true
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
		return nil, fmt.Errorf("file too large: max 2 GB per transfer")
	}
	return a.store.AddFileFromPath(info.Name(), absPath, info.Size())
}

func (a *App) sendDirectory(dirPath string, dirName string) (*TransferItem, error) {
	tmpFile, err := os.CreateTemp(a.store.tempDir, "dir-*.zip")
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	cleanupTemp := true
	defer func() {
		_ = tmpFile.Close()
		if cleanupTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	zipWriter := zip.NewWriter(tmpFile)

	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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
	if err := zipWriter.Close(); err != nil {
		_ = tmpFile.Close()
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}

	zipName := fmt.Sprintf("%s.zip", dirName)
	info, err := os.Stat(tmpPath)
	if err != nil {
		return nil, err
	}
	item, err := a.store.AddTempFile(zipName, tmpPath, info.Size())
	if err != nil {
		return nil, err
	}
	cleanupTemp = false
	return item, nil
}
