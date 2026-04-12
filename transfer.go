package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	maxFileSize     = 500 * 1024 * 1024 // 500 MB
	memoryThreshold = 100 * 1024 * 1024 // 100 MB
	maxTextSize     = 1 * 1024 * 1024   // 1 MB
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
}

type TransferStore struct {
	mu      sync.RWMutex
	items   map[string]*TransferItem
	history []*TransferItem
	tempDir string
}

func NewTransferStore() *TransferStore {
	tmpDir, err := os.MkdirTemp("", "landrop-*")
	if err != nil {
		tmpDir = os.TempDir()
	}
	return &TransferStore{
		items:   make(map[string]*TransferItem),
		history: make([]*TransferItem, 0),
		tempDir: tmpDir,
	}
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

func (s *TransferStore) AddFileFromPath(name string, srcPath string, size int64) (*TransferItem, error) {
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
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		item.Content = data
	} else {
		tmpFile := filepath.Join(s.tempDir, token+"_"+name)
		input, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(tmpFile, input, 0600); err != nil {
			return nil, err
		}
		item.FilePath = tmpFile
	}

	s.mu.Lock()
	s.items[token] = item
	s.mu.Unlock()
	return item, nil
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
	item, ok := s.items[token]
	s.mu.RUnlock()
	return item, ok
}

func (s *TransferStore) MarkDownloaded(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.items[token]; ok {
		item.Downloaded = true
		s.addToHistory(item)
		if item.OneTimeUse {
			if item.FilePath != "" {
				os.Remove(item.FilePath)
			}
			delete(s.items, token)
		}
	}
}

func (s *TransferStore) addToHistory(item *TransferItem) {
	s.history = append(s.history, item)
	if len(s.history) > 20 {
		s.history = s.history[len(s.history)-20:]
	}
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
