package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

type AppConfig struct {
	DeviceName string `json:"device_name,omitempty"`
}

type ConfigStore struct {
	mu   sync.RWMutex
	path string
	cfg  AppConfig
}

var appConfig = NewConfigStore()

func getStateDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	dir := filepath.Join(configDir, "LANDrop")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func getConfigPath() string {
	return filepath.Join(getStateDir(), "config.json")
}

func NewConfigStore() *ConfigStore {
	store := &ConfigStore{path: getConfigPath()}
	_ = store.Load()
	return store
}

func LoadConfig() {
	_ = appConfig.Load()
}

func normalizeDeviceName(name string) string {
	fields := strings.FieldsFunc(name, func(r rune) bool {
		return unicode.IsControl(r) || unicode.IsSpace(r)
	})
	name = strings.Join(fields, " ")
	if name == "" {
		return ""
	}
	runes := []rune(name)
	if len(runes) > 63 {
		name = string(runes[:63])
	}
	return name
}

func (s *ConfigStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.cfg = AppConfig{}
			return nil
		}
		return err
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	cfg.DeviceName = normalizeDeviceName(cfg.DeviceName)
	s.cfg = cfg
	return nil
}

func (s *ConfigStore) DeviceName(defaultName string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.DeviceName != "" {
		return s.cfg.DeviceName
	}
	return defaultName
}

func (s *ConfigStore) SetDeviceName(name string, defaultName string) (string, error) {
	name = normalizeDeviceName(name)
	defaultName = normalizeDeviceName(defaultName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if name == defaultName {
		name = ""
	}

	s.cfg.DeviceName = name
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return "", err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if s.cfg.DeviceName != "" {
		return s.cfg.DeviceName, nil
	}
	return defaultName, nil
}
