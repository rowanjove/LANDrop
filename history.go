package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type HistoryRecord struct {
	Timestamp int64  `json:"timestamp"`
	Direction string `json:"direction"` // "send" or "recv"
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Type      string `json:"type"`   // "file" or "text"
	Status    string `json:"status"` // "success", "failed", "interrupted"
	Peer      string `json:"peer"`   // IP or Address
}

var (
	historyList []*HistoryRecord
	historyMu   sync.Mutex
)

func getHistoryPath() string {
	return filepath.Join(getStateDir(), "history.json")
}

func LoadHistory() {
	historyMu.Lock()
	defer historyMu.Unlock()
	data, err := os.ReadFile(getHistoryPath())
	if err == nil {
		json.Unmarshal(data, &historyList)
	}
}

func SaveHistory() {
	data, err := json.MarshalIndent(historyList, "", "  ")
	if err == nil {
		os.WriteFile(getHistoryPath(), data, 0644)
	}
}

func AppendHistory(record *HistoryRecord) {
	historyMu.Lock()
	defer historyMu.Unlock()
	if record.Timestamp == 0 {
		record.Timestamp = time.Now().Unix()
	}
	historyList = append(historyList, record)
	if len(historyList) > 500 { // limit to 500
		historyList = historyList[len(historyList)-500:]
	}
	SaveHistory()
}

func ClearHistory() int {
	historyMu.Lock()
	defer historyMu.Unlock()
	count := len(historyList)
	historyList = make([]*HistoryRecord, 0)
	SaveHistory()
	return count
}

func GetHistoryRecords() []*HistoryRecord {
	historyMu.Lock()
	defer historyMu.Unlock()
	res := make([]*HistoryRecord, len(historyList))
	copy(res, historyList)
	return res
}
