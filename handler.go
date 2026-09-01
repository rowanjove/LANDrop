package main

import (
	"archive/zip"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Keep the no-2GB behavior while still bounding a single hostile request.
// The limit is intentionally well above the supported 5GB acceptance case.
const maxUploadSize int64 = 8 * 1024 * 1024 * 1024

//go:embed web/dist/*
var webDist embed.FS
var distFS, _ = fs.Sub(webDist, "web/dist")

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
	// SPA & Static Routes
	mux.HandleFunc("GET /", a.handleIndex)
	mux.HandleFunc("GET /activity", a.handleIndex)
	mux.HandleFunc("GET /settings", a.handleSettingsRoute)
	mux.HandleFunc("GET /r/{token}", a.handleIndex)
	mux.HandleFunc("POST /settings", a.handleSettings)

	// Legacy V1 API
	mux.HandleFunc("GET /info", a.handleInfo)
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

	// API V2 (Phase 24 & 25)
	mux.HandleFunc("GET /api/v2/info", a.handleInfo)
	mux.HandleFunc("GET /api/v2/devices", a.handleDevices)
	mux.HandleFunc("GET /api/v2/transfers", a.handleListTransfers)
	mux.HandleFunc("GET /api/v2/transfers/{id}", a.handleGetTransfer)
	mux.HandleFunc("DELETE /api/v2/transfers/{id}", a.handleCancelTransfer)
	mux.HandleFunc("POST /api/v2/transfers/files", a.handleSendFile)
	mux.HandleFunc("POST /api/v2/transfers/text", a.handleSendText)
	mux.HandleFunc("GET /api/v2/history", a.handleHistory)
	mux.HandleFunc("DELETE /api/v2/history", a.handleClearHistory)
	mux.HandleFunc("GET /api/v2/settings", a.handleSettings)
	mux.HandleFunc("PUT /api/v2/settings", a.handleSettings)
	mux.HandleFunc("GET /api/v2/events", a.handleEvents)
	mux.HandleFunc("GET /api/v2/share/{token}", a.handleShareInfo)
	mux.HandleFunc("GET /api/v2/share/{token}/items/{itemID}", a.handleRecv)
	mux.HandleFunc("GET /api/v2/share/{token}/zip", a.handleRecv)
}

func (a *App) handleSettingsRoute(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		a.handleSettings(w, r)
		return
	}
	a.handleIndex(w, r)
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if cleanPath == "" || cleanPath == "." || cleanPath == "activity" || cleanPath == "settings" ||
		(strings.HasPrefix(cleanPath, "r/") && strings.Count(cleanPath, "/") == 1) {
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			http.Error(w, "Web UI not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
		return
	}

	// Try serving static asset from distFS
	data, err := fs.ReadFile(distFS, cleanPath)
	if err == nil {
		if strings.HasPrefix(cleanPath, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		switch {
		case strings.HasSuffix(cleanPath, ".js"):
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case strings.HasSuffix(cleanPath, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(cleanPath, ".png"):
			w.Header().Set("Content-Type", "image/png")
		case strings.HasSuffix(cleanPath, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		case strings.HasSuffix(cleanPath, ".ico"):
			w.Header().Set("Content-Type", "image/x-icon")
		}
		w.Write(data)
		return
	}

	http.NotFound(w, r)
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

func (a *App) handleListTransfers(w http.ResponseWriter, r *http.Request) {
	items := a.store.List()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"transfers": items,
	})
}

func (a *App) handleGetTransfer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, ok := a.store.GetByID(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "transfer not found",
		})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *App) handleCancelTransfer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ok := a.store.Cancel(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "transfer not found",
		})
		return
	}
	a.broker.Broadcast("transfer_cancelled", map[string]string{"id": id})
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": true})
}

func (a *App) handleSendFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	// Always remove multipart temp files, including when parsing fails halfway.
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	// Parse multipart form, keep up to 32 MB in memory, rest goes to temp disk files.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		status := http.StatusBadRequest
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			status = http.StatusRequestEntityTooLarge
		}
		writeJSON(w, status, map[string]interface{}{
			"error": "Failed to parse upload form",
			"code":  "UPLOAD_FAILED",
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
			"id":         item.ID,
			"token":      item.Token,
			"name":       item.Name,
			"size":       item.Size,
			"expires_at": item.ExpiresAt,
		})
		return
	}

	// Multi-file transfers retain each file and generate a ZIP only when the
	// receiver chooses "download all". This keeps individual item downloads
	// available without paying the upload-time compression cost.
	storedFiles := make([]TransferFile, 0, len(files))
	cleanupStored := true
	defer func() {
		if cleanupStored {
			for _, file := range storedFiles {
				if file.FilePath != "" {
					_ = os.Remove(file.FilePath)
				}
			}
		}
	}()
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store upload.",
				"code":  "UPLOAD_FAILED",
			})
			return
		}
		tmpFile, err := os.CreateTemp(a.store.tempDir, "item-*")
		if err != nil {
			_ = f.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store upload.",
				"code":  "UPLOAD_FAILED",
			})
			return
		}
		written, copyErr := io.Copy(tmpFile, f)
		closeErr := tmpFile.Close()
		_ = f.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(tmpFile.Name())
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to store upload.",
				"code":  "UPLOAD_FAILED",
			})
			return
		}
		storedFiles = append(storedFiles, TransferFile{
			ID:       generateToken(),
			Name:     sanitizeFilename(fh.Filename),
			Size:     written,
			FilePath: tmpFile.Name(),
		})
	}
	item, err := a.store.AddMultiFiles(storedFiles)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to store upload.",
			"code":  "UPLOAD_FAILED",
		})
		return
	}
	cleanupStored = false
	item.OneTimeUse = a.oneTimeUse

	a.broker.Broadcast("file_ready", map[string]interface{}{
		"token":    item.Token,
		"name":     item.Name,
		"size":     item.Size,
		"items":    item.Items,
		"type":     "file",
		"sender":   r.Header.Get("X-Client-ID"),
		"one_time": item.OneTimeUse,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         item.ID,
		"token":      item.Token,
		"name":       item.Name,
		"size":       item.Size,
		"expires_at": item.ExpiresAt,
		"items":      item.Items,
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

// handleShareInfo is a non-consuming share-page API. Opening a preview must
// not claim a one-time token; only the actual /recv download does that.
func (a *App) handleShareInfo(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	item, ok := a.store.Get(token)
	if !ok {
		if a.store.IsUsed(token) {
			writeJSON(w, http.StatusGone, map[string]string{"error": "This item has already been used.", "code": "TOKEN_USED"})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "token not found", "code": "TOKEN_NOT_FOUND"})
		return
	}
	data := map[string]interface{}{
		"type":       item.Type,
		"name":       item.Name,
		"size":       item.Size,
		"expires_at": item.ExpiresAt,
		"one_time":   item.OneTimeUse,
		"status":     item.Status,
	}
	if len(item.Items) > 0 {
		data["items"] = item.Items
	}
	if item.Type == "text" {
		data["content"] = string(item.Content)
	}
	writeJSON(w, http.StatusOK, data)
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
	if itemID := r.PathValue("itemID"); itemID != "" {
		a.handleRecvItem(w, r, token, itemID)
		return
	}

	if r.Method == http.MethodHead {
		item, ok := a.store.Get(token)
		if !ok {
			if a.store.IsUsed(token) {
				writeJSON(w, http.StatusGone, map[string]interface{}{
					"error": "This item has already been claimed.",
					"code":  "TOKEN_USED",
				})
				return
			}
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
		if len(item.Items) > 0 {
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", contentDisposition(item.Name))
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
	if len(item.Items) > 0 {
		fileOutcome, err := a.serveZip(w, r, item)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "file read error"})
			return
		}
		outcome = fileOutcome
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

func (a *App) handleRecvItem(w http.ResponseWriter, r *http.Request, token string, itemID string) {
	item, ok := a.store.Get(token)
	if !ok {
		status := http.StatusNotFound
		code := "TOKEN_NOT_FOUND"
		message := "token not found"
		if a.store.IsUsed(token) {
			status = http.StatusGone
			code = "TOKEN_USED"
			message = "This item has already been claimed."
		}
		writeJSON(w, status, map[string]string{"error": message, "code": code})
		return
	}
	if item.OneTimeUse {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "One-time transfers must be downloaded as a single ZIP.",
			"code":  "USE_ZIP_FOR_ONE_TIME",
		})
		return
	}
	if len(item.Items) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "item not found", "code": "ITEM_NOT_FOUND"})
		return
	}
	var selected *TransferFile
	for i := range item.Items {
		if item.Items[i].ID == itemID {
			selected = &item.Items[i]
			break
		}
	}
	if selected == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "item not found", "code": "ITEM_NOT_FOUND"})
		return
	}
	if _, found, unavailable := a.store.BeginDownload(token); !found || unavailable {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "transfer not found", "code": "TOKEN_NOT_FOUND"})
		return
	}
	child := &Transfer{
		Token:     token,
		Type:      "file",
		Name:      selected.Name,
		Size:      selected.Size,
		CreatedAt: item.CreatedAt,
		Content:   selected.Content,
		FilePath:  selected.FilePath,
	}
	outcome := DownloadReleased
	defer func() { a.store.FinishDownload(token, outcome, r.RemoteAddr) }()
	var err error
	_, err = a.serveFile(w, r, child)
	// Item downloads are independent in a multi-file transfer. Keep the
	// aggregate transfer available for subsequent item downloads and ZIP export.
	outcome = DownloadReleased
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "file read error"})
	}
}

type trackingResponseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
	writeErr     error
	onProgress   func(written int64)
	allowWrite   func() bool
}

func (w *trackingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *trackingResponseWriter) Write(p []byte) (int, error) {
	if w.allowWrite != nil && !w.allowWrite() {
		w.writeErr = context.Canceled
		return 0, context.Canceled
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += int64(n)
	if err != nil {
		w.writeErr = err
	}
	if w.onProgress != nil && n > 0 {
		w.onProgress(w.bytesWritten)
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

	var lastProgressTime time.Time
	onProgress := func(written int64) {
		item.BytesTransferred = written
		a.store.UpdateProgress(item.Token, written)
		now := time.Now()
		if now.Sub(lastProgressTime) >= 250*time.Millisecond {
			lastProgressTime = now
			a.broker.Broadcast("transfer.progress", map[string]interface{}{
				"token":  item.Token,
				"name":   item.Name,
				"loaded": written,
				"total":  item.Size,
				"status": "transferring",
			})
		}
	}

	tw := &trackingResponseWriter{ResponseWriter: w, onProgress: onProgress, allowWrite: func() bool {
		return !a.store.IsCancelled(item.Token)
	}}
	tw.Header().Set("Content-Type", "application/octet-stream")
	tw.Header().Set("Content-Disposition", contentDisposition(item.Name))
	http.ServeContent(tw, r, item.Name, modTime, reader)
	return classifyDownloadOutcome(item.Size, tw), nil
}

func (a *App) serveZip(w http.ResponseWriter, r *http.Request, item *TransferItem) (DownloadOutcome, error) {
	tw := &trackingResponseWriter{
		ResponseWriter: w,
		allowWrite: func() bool {
			return !a.store.IsCancelled(item.Token)
		},
	}
	tw.Header().Set("Content-Type", "application/zip")
	tw.Header().Set("Content-Disposition", contentDisposition(item.Name))
	tw.WriteHeader(http.StatusOK)
	zipWriter := zip.NewWriter(tw)
	var transferred int64
	for _, file := range item.Items {
		header := &zip.FileHeader{Name: sanitizeFilename(file.Name), Method: zip.Deflate}
		header.UncompressedSize64 = uint64(maxInt64(file.Size, 0))
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return zipDownloadOutcome(tw), err
		}
		var reader io.Reader
		var closeFn func() error
		if file.Content != nil {
			reader = bytes.NewReader(file.Content)
		} else {
			f, openErr := os.Open(file.FilePath)
			if openErr != nil {
				return zipDownloadOutcome(tw), openErr
			}
			reader = f
			closeFn = f.Close
		}
		_, copyErr := io.Copy(writer, reader)
		if closeFn != nil {
			_ = closeFn()
		}
		if copyErr != nil {
			return zipDownloadOutcome(tw), copyErr
		}
		transferred += file.Size
		a.store.UpdateProgress(item.Token, transferred)
		a.broker.Broadcast("transfer.progress", map[string]interface{}{
			"token": item.Token, "name": item.Name, "loaded": transferred, "total": item.Size, "status": "transferring",
		})
	}
	if err := zipWriter.Close(); err != nil {
		return zipDownloadOutcome(tw), err
	}
	if tw.writeErr != nil {
		return zipDownloadOutcome(tw), tw.writeErr
	}
	return DownloadCompleted, nil
}

func zipDownloadOutcome(w *trackingResponseWriter) DownloadOutcome {
	if w.writeErr != nil && w.bytesWritten > 0 {
		return DownloadInterrupted
	}
	return DownloadFailed
}

func maxInt64(value, fallback int64) int64 {
	if value < fallback {
		return fallback
	}
	return value
}

func contentDisposition(name string) string {
	name = sanitizeFilename(name)
	// Keep an ASCII-safe fallback for older clients and provide the UTF-8 form
	// for browsers that support RFC 5987.
	fallback := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' || r == '\\' {
			return -1
		}
		if r > 0x7f {
			return '_'
		}
		return r
	}, name)
	if fallback == "" {
		fallback = "download"
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fallback, url.PathEscape(name))
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
