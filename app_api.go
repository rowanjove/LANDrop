package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parsePositiveInt(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}

func (a *App) setDeviceName(name string) (string, error) {
	deviceName, err := a.config.SetDeviceName(name, a.systemName)
	if err != nil {
		return "", err
	}

	a.hostname = deviceName
	if a.mdns != nil {
		if err := a.mdns.UpdateDeviceName(deviceName); err != nil {
			return "", err
		}
	}
	if a.broker != nil {
		a.broker.Broadcast("settings_updated", map[string]string{
			"device_name": deviceName,
		})
	}
	return deviceName, nil
}

func (a *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]string{
			"device_name": a.hostname,
		})
	case http.MethodPost:
		var req struct {
			DeviceName string `json:"device_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid request",
			})
			return
		}
		deviceName, err := a.setDeviceName(req.DeviceName)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to save settings",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"device_name": deviceName,
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func filterHistoryRecords(records []*HistoryRecord, peer string, day time.Time, filterByDay bool, limit int) []*HistoryRecord {
	result := make([]*HistoryRecord, 0, len(records))

	for _, record := range records {
		if peer != "" && !strings.Contains(strings.ToLower(record.Peer), peer) {
			continue
		}
		if filterByDay {
			ts := time.Unix(record.Timestamp, 0)
			if ts.Year() != day.Year() || ts.Month() != day.Month() || ts.Day() != day.Day() {
				continue
			}
		}
		result = append(result, record)
	}

	if limit > 0 && len(result) > limit {
		return result[len(result)-limit:]
	}
	return result
}

func (a *App) handleHistory(w http.ResponseWriter, r *http.Request) {
	peer := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("peer")))
	dateValue := strings.TrimSpace(r.URL.Query().Get("date"))
	limit := 0

	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := parsePositiveInt(rawLimit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid limit",
			})
			return
		}
		limit = parsed
	}

	var (
		day         time.Time
		filterByDay bool
	)
	if dateValue != "" {
		parsed, err := time.Parse("2006-01-02", dateValue)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid date",
			})
			return
		}
		day = parsed
		filterByDay = true
	}

	records := filterHistoryRecords(GetHistoryRecords(), peer, day, filterByDay, limit)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"history": records,
	})
}

func (a *App) handleClearHistory(w http.ResponseWriter, r *http.Request) {
	cleared := ClearHistory()
	writeJSON(w, http.StatusOK, map[string]int{
		"cleared": cleared,
	})
}
