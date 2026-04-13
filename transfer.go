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
	maxFileSize     = 2 * 1024 * 1024 * 1024 // 2 GB
	memoryThreshold = 100 * 1024 * 1024      // 100 MB
	maxTextSize     = 10 * 1024 * 1024       // 10 MB
)

type TransferItem struct {
	Token      string `json:"token"`
	Type       string `json:"type"` // "file" or "text"
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Content    []byte `json:"-"`
	FilePath   string `json:"-"`
	CreatedAt  int64  `json:"created_at"`
	ExpiresAt  int64  `json:"expires_at"`
	Downloaded bool   `json:"downloaded"`
	OneTimeUse bool   `json:"-"`
	Claimed    bool   `json:"-"`
}

type TransferStore struct {
	mu      sync.RWMutex
	items   map[string]*TransferItem
	history []*TransferItem
	tempDir string
}

type DownloadOutcome int

const (
	DownloadReleased DownloadOutcome = iota
	DownloadFailed
	DownloadCompleted
)

func NewTransferStore() *TransferStore {
	tmpDir, err := os.MkdirTemp("", "landrop-*")
	if err != nil {
		log.Fatalf("failed to create temp directory: %v", err)
	}
	return &TransferStore{
		items:   make(map[string]*TransferItem),
		history: make([]*TransferItem, 0),
		tempDir: tmpDir,
	}
}

// sanitizeFilename strips path separators and traversal sequences from filenames
func sanitizeFilename(name string) string {
	// Take only the base name, stripping any directory components
	name = filepath.Base(name)
	// On Windows, also handle forward slashes
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	// Reject empty or dot-only names
	if name == "" || name == "." || name == ".." {
		name = "unnamed"
	}
	return name
}

func (s *TransferStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.items {
		if item.FilePath != "" {
			os.Remove(item.FilePath)
		}
	}
	os.RemoveAll(s.tempDir)
}

func generateToken() string {
	id := uuid.New()
	hex := fmt.Sprintf("%x", id[:6])
	return hex
}

func (s *TransferStore) AddFile(name string, data []byte, size int64) (*TransferItem, error) {
	name = sanitizeFilename(name)
	token := generateToken()
	item := &TransferItem{
		Token:     token,
		Type:      "file",
		Name:      name,
		Size:      size,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: 0,
	}

	if size <= memoryThreshold {
		item.Content = data
	} else {
		tmpFile := filepath.Join(s.tempDir, token+"_"+name)
		if err := os.WriteFile(tmpFile, data, 0600); err != nil {
			return nil, fmt.Errorf("failed to write temp file: %w", err)
		}
		item.FilePath = tmpFile
	}

	s.mu.Lock()
	s.items[token] = item
	s.mu.Unlock()
	return item, nil
}

func (s *TransferStore) AddFileFromReader(name string, src io.Reader, size int64) (*TransferItem, error) {
	name = sanitizeFilename(name)
	if size > maxFileSize {
		return nil, fmt.Errorf("file too large")
	}

	token := generateToken()
	item := &TransferItem{
		Token:     token,
		Type:      "file",
		Name:      name,
		Size:      size,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: 0,
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

func (s *TransferStore) AddTempFile(name string, tempPath string, size int64) (*TransferItem, error) {
	name = sanitizeFilename(name)
	if size > maxFileSize {
		return nil, fmt.Errorf("file too large")
	}

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

	item := &TransferItem{
		Token:     token,
		Type:      "file",
		Name:      name,
		Size:      size,
		FilePath:  targetPath,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: 0,
	}

	s.mu.Lock()
	s.items[token] = item
	s.mu.Unlock()
	return item, nil
}

func (s *TransferStore) AddFileFromPath(name string, srcPath string, size int64) (*TransferItem, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return s.AddFileFromReader(name, f, size)
}

func (s *TransferStore) AddText(content string) (*TransferItem, error) {
	if len(content) > maxTextSize {
		return nil, fmt.Errorf("text too large, max 1 MB")
	}
	token := generateToken()
	item := &TransferItem{
		Token:     token,
		Type:      "text",
		Size:      int64(len(content)),
		Content:   []byte(content),
		CreatedAt: time.Now().Unix(),
		ExpiresAt: 0,
	}
	s.mu.Lock()
	s.items[token] = item
	s.mu.Unlock()
	return item, nil
}

func (s *TransferStore) Get(token string) (*TransferItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[token]
	if !ok {
		return nil, false
	}
	// Return a shallow copy to avoid races on shared fields
	cp := *item
	return &cp, true
}

func (s *TransferStore) BeginDownload(token string) (*TransferItem, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[token]
	if !ok {
		return nil, false, false
	}
	if item.OneTimeUse && (item.Downloaded || item.Claimed) {
		return nil, true, true
	}
	if item.OneTimeUse {
		item.Claimed = true
	}

	cp := *item
	return &cp, true, false
}

func (s *TransferStore) FinishDownload(token string, outcome DownloadOutcome, peer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.items[token]; ok {
		if item.OneTimeUse {
			if !item.Claimed {
				return
			}
			switch outcome {
			case DownloadCompleted:
				item.Downloaded = true
				item.Claimed = false
				s.addToHistory(item, peer, true)
				if item.FilePath != "" {
					_ = os.Remove(item.FilePath)
				}
				delete(s.items, token)
				return
			case DownloadFailed:
				s.addToHistory(item, peer, false)
				item.Claimed = false
			case DownloadReleased:
				item.Claimed = false
			}
			return
		}
		switch outcome {
		case DownloadCompleted:
			if item.Downloaded {
				return
			}
			item.Downloaded = true
			s.addToHistory(item, peer, true)
		case DownloadFailed:
			s.addToHistory(item, peer, false)
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

func (s *TransferStore) addToHistory(item *TransferItem, peer string, success bool) {
	s.history = append(s.history, item)
	if len(s.history) > 20 {
		s.history = s.history[len(s.history)-20:]
	}

	status := "success"
	if !success {
		status = "failed"
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

func (s *TransferStore) GetHistory() []*TransferItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*TransferItem, len(s.history))
	copy(result, s.history)
	return result
}

func (s *TransferStore) List() []*TransferItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*TransferItem, 0, len(s.items))
	for _, item := range s.items {
		result = append(result, item)
	}
	return result
}
