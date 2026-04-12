package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

type Device struct {
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	OS       string `json:"os"`
	Version  string `json:"version"`
	LastSeen int64  `json:"last_seen"`
}

type MDNSManager struct {
	mu       sync.RWMutex
	devices  map[string]*Device
	server   *zeroconf.Server
	broker   *SSEBroker
	port     int
	hostname string
	cancel   context.CancelFunc
}

func NewMDNSManager(port int, broker *SSEBroker) *MDNSManager {
	hostname, _ := os.Hostname()
	return &MDNSManager{
		devices:  make(map[string]*Device),
		broker:   broker,
		port:     port,
		hostname: hostname,
	}
}

func (m *MDNSManager) Start() error {
	server, err := zeroconf.Register(
		m.hostname,
		"_landrop._tcp",
		"local.",
		m.port,
		[]string{"version=1.0.0", "os=" + getOS()},
		nil,
	)
	if err != nil {
		return fmt.Errorf("mDNS register failed: %w", err)
	}
	m.server = server

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	go m.discover(ctx)
	go m.cleanup(ctx)

	return nil
}

func (m *MDNSManager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.server != nil {
		m.server.Shutdown()
	}
}

func (m *MDNSManager) discover(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			log.Printf("mDNS resolver error: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		entries := make(chan *zeroconf.ServiceEntry)
		go func() {
			for entry := range entries {
				if len(entry.AddrIPv4) == 0 {
					continue
				}
				addr := fmt.Sprintf("%s:%d", entry.AddrIPv4[0], entry.Port)

				// Skip self
				if entry.Port == m.port && entry.Instance == m.hostname {
					continue
				}

				osType := ""
				version := ""
				for _, txt := range entry.Text {
					if len(txt) > 3 && txt[:3] == "os=" {
						osType = txt[3:]
					}
					if len(txt) > 8 && txt[:8] == "version=" {
						version = txt[8:]
					}
				}

				device := &Device{
					Name:     entry.Instance,
					Addr:     addr,
					OS:       osType,
					Version:  version,
					LastSeen: time.Now().Unix(),
				}

				m.mu.Lock()
				existing, found := m.devices[addr]
				m.devices[addr] = device
				m.mu.Unlock()

				if !found || existing.Name != device.Name {
					m.broker.Broadcast("device_found", map[string]string{
						"name": device.Name,
						"addr": device.Addr,
					})
				}
			}
		}()

		browseCtx, browseCancel := context.WithTimeout(ctx, 5*time.Second)
		err = resolver.Browse(browseCtx, "_landrop._tcp", "local.", entries)
		if err != nil {
			log.Printf("mDNS browse error: %v", err)
		}
		<-browseCtx.Done()
		browseCancel()

		time.Sleep(5 * time.Second)
	}
}

func (m *MDNSManager) cleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().Unix()
			m.mu.Lock()
			for addr, dev := range m.devices {
				if now-dev.LastSeen > 30 {
					delete(m.devices, addr)
					m.broker.Broadcast("device_lost", map[string]string{
						"addr": addr,
					})
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *MDNSManager) GetDevices() []*Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Device, 0, len(m.devices))
	for _, dev := range m.devices {
		result = append(result, dev)
	}
	return result
}
