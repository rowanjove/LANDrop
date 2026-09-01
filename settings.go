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
	DeviceName           string `json:"device_name,omitempty"`
	DefaultExpiry        int64  `json:"default_expiry,omitempty"`
	DefaultDownloadLimit int    `json:"default_download_limit,omitempty"`
	Theme                string `json:"theme,omitempty"`
	Language             string `json:"language,omitempty"`
}

type ConfigStore struct {
	mu   sync.RWMutex
	path string
	cfg  AppConfig
}

var appConfig = NewConfigStore()

func getStateDir() string {
	if override := strings.TrimSpace(os.Getenv("LANDROP_STATE_DIR")); override != "" {
		_ = os.MkdirAll(override, 0o755)
		return override
	}
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

func (s *ConfigStore) GetConfig(defaultName string) AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := s.cfg
	if cfg.DeviceName == "" {
		cfg.DeviceName = defaultName
	}
	return cfg
}

func (s *ConfigStore) UpdateConfig(newCfg AppConfig, defaultName string) (AppConfig, error) {
	defaultName = normalizeDeviceName(defaultName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if newCfg.DeviceName != "" {
		name := normalizeDeviceName(newCfg.DeviceName)
		if name == defaultName {
			name = ""
		}
		s.cfg.DeviceName = name
	}
	if newCfg.Theme != "" {
		s.cfg.Theme = newCfg.Theme
	}
	if newCfg.Language != "" {
		s.cfg.Language = newCfg.Language
	}
	if newCfg.DefaultExpiry > 0 {
		s.cfg.DefaultExpiry = newCfg.DefaultExpiry
	}
	if newCfg.DefaultDownloadLimit > 0 {
		s.cfg.DefaultDownloadLimit = newCfg.DefaultDownloadLimit
	}

	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return s.cfg, err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return s.cfg, err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return s.cfg, err
	}

	res := s.cfg
	if res.DeviceName == "" {
		res.DeviceName = defaultName
	}
	return res, nil
}

func (s *ConfigStore) DeviceName(defaultName string) string {
	return s.GetConfig(defaultName).DeviceName
}

func (s *ConfigStore) SetDeviceName(name string, defaultName string) (string, error) {
	cfg, err := s.UpdateConfig(AppConfig{DeviceName: name}, defaultName)
	if err != nil {
		return "", err
	}
	return cfg.DeviceName, nil
}
