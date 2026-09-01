package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	memoryThreshold = 1 * 1024 * 1024  // 1 MB: files <= 1MB in RAM, larger to disk
	maxTextSize     = 10 * 1024 * 1024 // 10 MB max text
	defaultTTL      = 3600             // 1 hour default TTL
)

type TransferStatus string

const (
	TransferStatusPending      TransferStatus = "pending"
	TransferStatusReady        TransferStatus = "ready"
	TransferStatusTransferring TransferStatus = "transferring"
	TransferStatusCompleted    TransferStatus = "completed"
	TransferStatusFailed       TransferStatus = "failed"
	TransferStatusInterrupted  TransferStatus = "interrupted"
	TransferStatusCancelled    TransferStatus = "cancelled"
	TransferStatusExpired      TransferStatus = "expired"
)

type TransferFile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type,omitempty"`
	FilePath string `json:"-"`
	Content  []byte `json:"-"`
}

type Transfer struct {
	ID               string         `json:"id"`
	Token            string         `json:"token"`
	Direction        string         `json:"direction"` // "send" or "recv"
	Type             string         `json:"type"`      // "file" or "text"
	Name             string         `json:"name"`
	Size             int64          `json:"size"`
	Items            []TransferFile `json:"items,omitempty"`
	CreatedAt        int64          `json:"created_at"`
	ExpiresAt        int64          `json:"expires_at"` // Unix timestamp or 0
	Status           TransferStatus `json:"status"`
	Peer             string         `json:"peer,omitempty"`
	BytesTransferred int64          `json:"bytes_transferred"`
	DownloadLimit    int            `json:"download_limit,omitempty"`
	DownloadCount    int            `json:"download_count,omitempty"`
	Content          []byte         `json:"-"`
	FilePath         string         `json:"-"`
	OneTimeUse       bool           `json:"one_time,omitempty"`
	Claimed          bool           `json:"-"`
	Downloaded       bool           `json:"downloaded,omitempty"`
}

// Backward compatibility alias for TransferItem
type TransferItem = Transfer

type TransferStore struct {
	mu             sync.RWMutex
	items          map[string]*Transfer
	history        []*Transfer
	usedTokens     map[string]time.Time
	cancelled      map[string]struct{}
	cancelledFiles map[string][]string
	tempDir        string
}

type DownloadOutcome int

const (
	DownloadReleased DownloadOutcome = iota
	DownloadFailed
	DownloadInterrupted
	DownloadCompleted
)

func NewTransferStore() *TransferStore {
	tmpDir, err := os.MkdirTemp("", "landrop-*")
	if err != nil {
		log.Fatalf("创建临时目录失败：%v", err)
	}
	return &TransferStore{
		items:          make(map[string]*Transfer),
		history:        make([]*Transfer, 0),
		usedTokens:     make(map[string]time.Time),
		cancelled:      make(map[string]struct{}),
		cancelledFiles: make(map[string][]string),
		tempDir:        tmpDir,
	}
}

// sanitizeFilename strips path separators, traversal sequences, and Windows reserved names
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")

	if name == "" {
		return "unnamed"
	}

	// Check for Windows reserved device names (e.g. NUL, NUL.tar.gz, COM1.txt)
	base := strings.ToUpper(name)
	if dotIdx := strings.Index(base, "."); dotIdx >= 0 {
		base = base[:dotIdx]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}

	return name
}

func (s *TransferStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.items {
		removeTransferFiles(item)
	}
	_ = os.RemoveAll(s.tempDir)
}

func (s *TransferStore) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	count := 0
	for token, item := range s.items {
		if item.ExpiresAt > 0 && now >= item.ExpiresAt {
			removeTransferFiles(item)
			item.Status = TransferStatusExpired
			delete(s.items, token)
			count++
		}
	}
	// Keep one-time tombstones long enough to return a useful TOKEN_USED response,
	// but do not let them grow without bound in a long-running process.
	for token, usedAt := range s.usedTokens {
		if now-usedAt.Unix() > 24*60*60 {
			delete(s.usedTokens, token)
		}
	}
	return count
}

func generateToken() string {
	id := uuid.New()
	hex := fmt.Sprintf("%x", id[:6])
	return hex
}

func (s *TransferStore) AddFile(name string, data []byte, size int64) (*Transfer, error) {
	name = sanitizeFilename(name)
	token := generateToken()
	now := time.Now().Unix()
	item := &Transfer{
		ID:        uuid.New().String(),
		Token:     token,
		Direction: "send",
		Type:      "file",
		Name:      name,
		Size:      size,
		CreatedAt: now,
		ExpiresAt: now + defaultTTL,
		Status:    TransferStatusReady,
	}

	if size <= memoryThreshold {
		item.Content = data
	} else {
		tmpFile := filepath.Join(s.tempDir, token+"_"+name)
		if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
			return nil, fmt.Errorf("failed to write temp file: %w", err)
		}
		item.FilePath = tmpFile
	}

	s.mu.Lock()
	s.items[token] = item
	s.mu.Unlock()
	return item, nil
}

func (s *TransferStore) AddFileFromReader(name string, src io.Reader, size int64) (*Transfer, error) {
	name = sanitizeFilename(name)
	token := generateToken()
	now := time.Now().Unix()
	item := &Transfer{
		ID:        uuid.New().String(),
		Token:     token,
		Direction: "send",
		Type:      "file",
		Name:      name,
		Size:      size,
		CreatedAt: now,
		ExpiresAt: now + defaultTTL,
		Status:    TransferStatusReady,
	}

	if size >= 0 && size <= memoryThreshold {
		data, err := io.ReadAll(src)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		item.Content = data
		item.Size = int64(len(data))
	} else {
		tmpFile := filepath.Join(s.tempDir, token+"_"+name)
		f, err := os.Create(tmpFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		written, copyErr := io.Copy(f, src)
		closeErr := f.Close()
		if copyErr != nil {
			_ = os.Remove(tmpFile)
			return nil, fmt.Errorf("failed to write temp file: %w", copyErr)
		}
		if closeErr != nil {
			_ = os.Remove(tmpFile)
			return nil, fmt.Errorf("failed to close temp file: %w", closeErr)
		}
		item.FilePath = tmpFile
		item.Size = written
	}

	s.mu.Lock()
	s.items[token] = item
	s.mu.Unlock()
	return item, nil
}

func (s *TransferStore) AddTempFile(name string, tempPath string, size int64) (*Transfer, error) {
	name = sanitizeFilename(name)
	token := generateToken()
	targetPath := filepath.Join(s.tempDir, token+"_"+name)
	if filepath.Clean(tempPath) != filepath.Clean(targetPath) {
		if err := os.Rename(tempPath, targetPath); err != nil {
			if err := copyFile(tempPath, targetPath); err != nil {
				return nil, fmt.Errorf("failed to move temp file: %w", err)
			}
			_ = os.Remove(tempPath)
		}
	} else {
		targetPath = tempPath
	}

	if size <= 0 {
		info, err := os.Stat(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat temp file: %w", err)
		}
		size = info.Size()
	}

	now := time.Now().Unix()
	item := &Transfer{
		ID:        uuid.New().String(),
		Token:     token,
		Direction: "send",
		Type:      "file",
		Name:      name,
		Size:      size,
		FilePath:  targetPath,
		CreatedAt: now,
		ExpiresAt: now + defaultTTL,
		Status:    TransferStatusReady,
	}

	s.mu.Lock()
	s.items[token] = item
	s.mu.Unlock()
	return item, nil
}

// AddMultiFiles stores a transfer whose individual files can be downloaded
// separately. The paths must already point inside the store's private temp dir.
func (s *TransferStore) AddMultiFiles(files []TransferFile) (*Transfer, error) {
	if len(files) < 2 {
		return nil, fmt.Errorf("multi-file transfer requires at least two files")
	}

	cleaned := make([]TransferFile, 0, len(files))
	var total int64
	for _, file := range files {
		name := sanitizeFilename(file.Name)
		if file.FilePath == "" && file.Content == nil {
			return nil, fmt.Errorf("file %q has no content", name)
		}
		if file.FilePath != "" {
			rel, err := filepath.Rel(s.tempDir, filepath.Clean(file.FilePath))
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("file %q is outside the transfer temp directory", name)
			}
		}
		size := file.Size
		if size < 0 {
			if file.FilePath == "" {
				size = int64(len(file.Content))
			} else {
				info, err := os.Stat(file.FilePath)
				if err != nil {
					return nil, fmt.Errorf("failed to stat %q: %w", name, err)
				}
				size = info.Size()
			}
		}
		cleaned = append(cleaned, TransferFile{
			ID:       file.ID,
			Name:     name,
			Size:     size,
			MimeType: file.MimeType,
			FilePath: file.FilePath,
			Content:  file.Content,
		})
		total += size
	}

	now := time.Now().Unix()
	item := &Transfer{
		ID:        uuid.New().String(),
		Token:     generateToken(),
		Direction: "send",
		Type:      "file",
		Name:      fmt.Sprintf("landrop_%s.zip", time.Now().Format("20060102_150405")),
		Size:      total,
		Items:     cleaned,
		CreatedAt: now,
		ExpiresAt: now + defaultTTL,
		Status:    TransferStatusReady,
	}
	for i := range item.Items {
		if item.Items[i].ID == "" {
			item.Items[i].ID = generateToken()
		}
	}

	s.mu.Lock()
	s.items[item.Token] = item
	s.mu.Unlock()
	return item, nil
}

func (s *TransferStore) AddFileFromPath(name string, srcPath string, size int64) (*Transfer, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return s.AddFileFromReader(name, f, size)
}

func (s *TransferStore) AddText(content string) (*Transfer, error) {
	if len(content) > maxTextSize {
		return nil, fmt.Errorf("text too large, max 10 MB")
	}
	token := generateToken()
	now := time.Now().Unix()
	item := &Transfer{
		ID:        uuid.New().String(),
		Token:     token,
		Direction: "send",
		Type:      "text",
		Size:      int64(len(content)),
		Content:   []byte(content),
		CreatedAt: now,
		ExpiresAt: now + defaultTTL,
		Status:    TransferStatusReady,
	}
	s.mu.Lock()
	s.items[token] = item
	s.mu.Unlock()
	return item, nil
}

func (s *TransferStore) Get(token string) (*Transfer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[token]
	if !ok {
		return nil, false
	}
	return cloneTransfer(item), true
}

func (s *TransferStore) GetByID(id string) (*Transfer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.ID == id || item.Token == id {
			return cloneTransfer(item), true
		}
	}
	return nil, false
}

func (s *TransferStore) Cancel(idOrToken string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, item := range s.items {
		if item.Token == idOrToken || item.ID == idOrToken {
			wasActive := item.Status == TransferStatusTransferring
			item.Status = TransferStatusCancelled
			if wasActive {
				s.cancelled[token] = struct{}{}
				s.cancelledFiles[token] = transferFilePaths(item)
			} else {
				removeTransferFiles(item)
			}
			delete(s.items, token)
			return true
		}
	}
	return false
}

func (s *TransferStore) IsCancelled(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.cancelled[token]
	return ok
}

func (s *TransferStore) BeginDownload(token string) (*Transfer, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[token]
	if !ok {
		if _, used := s.usedTokens[token]; used {
			return nil, true, true
		}
		return nil, false, false
	}
	if item.OneTimeUse && (item.Downloaded || item.Claimed) {
		return nil, true, true
	}
	if item.OneTimeUse {
		item.Claimed = true
	}
	item.Status = TransferStatusTransferring

	return cloneTransfer(item), true, false
}

func (s *TransferStore) IsUsed(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.usedTokens[token]
	return ok
}

// UpdateProgress updates the canonical transfer record. Callers must not mutate
// pointers returned by Get/List because those methods return snapshots.
func (s *TransferStore) UpdateProgress(token string, written int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.items[token]; ok {
		item.BytesTransferred = written
	}
}

func (s *TransferStore) FinishDownload(token string, outcome DownloadOutcome, peer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cancelled, token)
	if paths, ok := s.cancelledFiles[token]; ok {
		for _, path := range paths {
			_ = os.Remove(path)
		}
		delete(s.cancelledFiles, token)
	}
	if item, ok := s.items[token]; ok {
		if item.OneTimeUse {
			if !item.Claimed {
				return
			}
			switch outcome {
			case DownloadCompleted:
				item.Downloaded = true
				item.Claimed = false
				item.Status = TransferStatusCompleted
				item.DownloadCount++
				s.usedTokens[token] = time.Now()
				s.addToHistory(item, peer, "success")
				removeTransferFiles(item)
				delete(s.items, token)
				return
			case DownloadFailed:
				item.Status = TransferStatusFailed
				s.addToHistory(item, peer, "failed")
				item.Claimed = false
			case DownloadInterrupted:
				item.Status = TransferStatusInterrupted
				s.addToHistory(item, peer, "interrupted")
				item.Claimed = false
			case DownloadReleased:
				item.Status = TransferStatusReady
				item.Claimed = false
			}
			return
		}

		switch outcome {
		case DownloadCompleted:
			item.Downloaded = true
			item.Status = TransferStatusCompleted
			item.DownloadCount++
			s.addToHistory(item, peer, "success")
		case DownloadFailed:
			item.Status = TransferStatusFailed
			s.addToHistory(item, peer, "failed")
		case DownloadInterrupted:
			item.Status = TransferStatusInterrupted
			s.addToHistory(item, peer, "interrupted")
		case DownloadReleased:
			item.Status = TransferStatusReady
		}
	}
}

func copyFile(srcPath string, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Close()
}

func (s *TransferStore) addToHistory(item *Transfer, peer string, status string) {
	s.history = append(s.history, item)
	if len(s.history) > 20 {
		s.history = s.history[len(s.history)-20:]
	}

	AppendHistory(&HistoryRecord{
		Direction: "send",
		Name:      item.Name,
		Size:      item.Size,
		Type:      item.Type,
		Status:    status,
		Peer:      peer,
	})
}

func removeTransferFiles(item *Transfer) {
	if item == nil {
		return
	}
	if item.FilePath != "" {
		_ = os.Remove(item.FilePath)
	}
	for _, file := range item.Items {
		if file.FilePath != "" {
			_ = os.Remove(file.FilePath)
		}
	}
}

func transferFilePaths(item *Transfer) []string {
	if item == nil {
		return nil
	}
	paths := make([]string, 0, len(item.Items)+1)
	if item.FilePath != "" {
		paths = append(paths, item.FilePath)
	}
	for _, file := range item.Items {
		if file.FilePath != "" {
			paths = append(paths, file.FilePath)
		}
	}
	return paths
}

func (s *TransferStore) GetHistory() []*Transfer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Transfer, len(s.history))
	for i, item := range s.history {
		result[i] = cloneTransfer(item)
	}
	return result
}

func (s *TransferStore) List() []*Transfer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Transfer, 0, len(s.items))
	for _, item := range s.items {
		result = append(result, cloneTransfer(item))
	}
	return result
}

func cloneTransfer(item *Transfer) *Transfer {
	if item == nil {
		return nil
	}
	cp := *item
	if item.Items != nil {
		cp.Items = append([]TransferFile(nil), item.Items...)
	}
	return &cp
}
