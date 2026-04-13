package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type ClipboardManager struct {
	broker  *SSEBroker
	content string
	mu      sync.RWMutex
}

func NewClipboardManager(broker *SSEBroker) *ClipboardManager {
	return &ClipboardManager{broker: broker}
}

func (c *ClipboardManager) HandlePush(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB max
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

	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "content is empty",
			"code":  "BAD_REQUEST",
		})
		return
	}

	c.mu.Lock()
	c.content = req.Content
	c.mu.Unlock()

	count := c.broker.ClientCount()
	c.broker.Broadcast("clipboard", map[string]string{
		"content": req.Content,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pushed_to": count,
	})
}

func (c *ClipboardManager) HandleGet(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	content := c.content
	c.mu.RUnlock()

	if content == "" {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "clipboard is empty",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"content": content,
	})
}
