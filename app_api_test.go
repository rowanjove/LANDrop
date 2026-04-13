package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigStoreSetDeviceNameFallsBackToDefault(t *testing.T) {
	store := &ConfigStore{path: filepath.Join(t.TempDir(), "config.json")}

	name, err := store.SetDeviceName("", "Host-PC")
	if err != nil {
		t.Fatalf("SetDeviceName() error = %v", err)
	}
	if name != "Host-PC" {
		t.Fatalf("SetDeviceName() name = %q, want %q", name, "Host-PC")
	}

	if got := store.DeviceName("Host-PC"); got != "Host-PC" {
		t.Fatalf("DeviceName() = %q, want %q", got, "Host-PC")
	}
}

func TestHandleSettingsPostUpdatesDeviceName(t *testing.T) {
	store := &ConfigStore{path: filepath.Join(t.TempDir(), "config.json")}
	app := NewApp("", "")
	app.config = store
	app.systemName = "Host-PC"
	app.hostname = "Host-PC"

	body := bytes.NewBufferString(`{"device_name":"Desk Node"}`)
	req := httptest.NewRequest(http.MethodPost, "/settings", body)
	rec := httptest.NewRecorder()

	app.handleSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("handleSettings() status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload struct {
		DeviceName string `json:"device_name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if payload.DeviceName != "Desk Node" {
		t.Fatalf("device_name = %q, want %q", payload.DeviceName, "Desk Node")
	}
	if got := store.DeviceName("Host-PC"); got != "Desk Node" {
		t.Fatalf("stored device name = %q, want %q", got, "Desk Node")
	}
}

func TestFilterHistoryRecordsByPeerAndDate(t *testing.T) {
	day := time.Date(2026, 4, 13, 10, 0, 0, 0, time.Local)
	records := []*HistoryRecord{
		{Timestamp: day.Add(-2 * time.Hour).Unix(), Peer: "192.168.1.2:53217", Name: "a"},
		{Timestamp: day.Add(-26 * time.Hour).Unix(), Peer: "192.168.1.2:53217", Name: "b"},
		{Timestamp: day.Add(-1 * time.Hour).Unix(), Peer: "192.168.1.9:53217", Name: "c"},
	}

	got := filterHistoryRecords(records, "1.2:53217", day, true, 0)
	if len(got) != 1 {
		t.Fatalf("filterHistoryRecords() len = %d, want %d", len(got), 1)
	}
	if got[0].Name != "a" {
		t.Fatalf("filterHistoryRecords() first name = %q, want %q", got[0].Name, "a")
	}
}
