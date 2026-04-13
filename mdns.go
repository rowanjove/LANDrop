package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	deviceOfflineAfter = 30 * time.Second
	devicePurgeAfter   = 24 * time.Hour
)

type Device struct {
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	OS       string `json:"os"`
	Version  string `json:"version"`
	LastSeen int64  `json:"last_seen"`
	Online   bool   `json:"online"`
}

type MDNSManager struct {
	mu         sync.RWMutex
	devices    map[string]*Device
	server     *zeroconf.Server
	broker     *SSEBroker
	port       int
	deviceName string
	localIPs   map[string]struct{}
	cancel     context.CancelFunc
}

func NewMDNSManager(port int, broker *SSEBroker, deviceName string) *MDNSManager {
	if deviceName == "" {
		deviceName, _ = os.Hostname()
	}
	return &MDNSManager{
		devices:    make(map[string]*Device),
		broker:     broker,
		port:       port,
		deviceName: deviceName,
		localIPs:   getLocalIPv4s(),
	}
}

func getLocalIPv4s() map[string]struct{} {
	result := make(map[string]struct{})
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return result
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil {
			continue
		}
		result[ip4.String()] = struct{}{}
	}
	return result
}

func (m *MDNSManager) txtRecords(name string) []string {
	return []string{
		"version=" + version,
		"os=" + getOS(),
		"name=" + name,
	}
}

func (m *MDNSManager) registerServer(name string) error {
	if m.port <= 0 {
		return nil
	}

	server, err := zeroconf.Register(
		name,
		"_landrop._tcp",
		"local.",
		m.port,
		m.txtRecords(name),
		nil,
	)
	if err != nil {
		return fmt.Errorf("mDNS register failed: %w", err)
	}

	m.mu.Lock()
	old := m.server
	m.server = server
	m.deviceName = name
	m.mu.Unlock()

	if old != nil {
		old.Shutdown()
	}
	return nil
}

func (m *MDNSManager) Start() error {
	m.mu.Lock()
	if m.cancel != nil {
		m.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	name := m.deviceName
	m.mu.Unlock()

	if err := m.registerServer(name); err != nil {
		cancel()
		m.mu.Lock()
		m.cancel = nil
		m.mu.Unlock()
		return err
	}

	go m.discover(ctx)
	go m.cleanup(ctx)
	return nil
}

func (m *MDNSManager) Stop() {
	m.mu.Lock()
	cancel := m.cancel
	server := m.server
	m.cancel = nil
	m.server = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if server != nil {
		server.Shutdown()
	}
}

func (m *MDNSManager) UpdateDeviceName(name string) error {
	if name == "" {
		name, _ = os.Hostname()
	}
	name = normalizeDeviceName(name)
	if name == "" {
		name = "LAN Drop"
	}

	m.mu.Lock()
	running := m.cancel != nil
	current := m.deviceName
	server := m.server
	m.server = nil
	m.deviceName = name
	m.mu.Unlock()

	if current == name {
		if server != nil {
			server.SetText(m.txtRecords(name))
			m.mu.Lock()
			m.server = server
			m.mu.Unlock()
		}
		return nil
	}

	if server != nil {
		server.Shutdown()
	}
	if running {
		return m.registerServer(name)
	}
	return nil
}

func (m *MDNSManager) isSelf(ip string, port int) bool {
	if port != m.port || port == 0 {
		return false
	}
	_, ok := m.localIPs[ip]
	return ok
}

func parseTXT(entries []string) (string, string) {
	osType := ""
	serviceVersion := ""
	for _, txt := range entries {
		switch {
		case len(txt) > 3 && txt[:3] == "os=":
			osType = txt[3:]
		case len(txt) > 8 && txt[:8] == "version=":
			serviceVersion = txt[8:]
		}
	}
	return osType, serviceVersion
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

				ip := entry.AddrIPv4[0].String()
				if m.isSelf(ip, entry.Port) {
					continue
				}

				addr := fmt.Sprintf("%s:%d", ip, entry.Port)
				osType, serviceVersion := parseTXT(entry.Text)
				now := time.Now().Unix()
				device := &Device{
					Name:     entry.Instance,
					Addr:     addr,
					OS:       osType,
					Version:  serviceVersion,
					LastSeen: now,
					Online:   true,
				}

				m.mu.Lock()
				existing, found := m.devices[addr]
				shouldBroadcast := !found ||
					!existing.Online ||
					existing.Name != device.Name ||
					existing.OS != device.OS ||
					existing.Version != device.Version
				m.devices[addr] = device
				m.mu.Unlock()

				if shouldBroadcast && m.broker != nil {
					m.broker.Broadcast("device_found", device)
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
			now := time.Now()
			var lost []*Device

			m.mu.Lock()
			for addr, dev := range m.devices {
				age := now.Sub(time.Unix(dev.LastSeen, 0))
				if dev.Online && age > deviceOfflineAfter {
					dev.Online = false
					cp := *dev
					lost = append(lost, &cp)
				}
				if age > devicePurgeAfter {
					delete(m.devices, addr)
				}
			}
			m.mu.Unlock()

			if m.broker != nil {
				for _, dev := range lost {
					m.broker.Broadcast("device_lost", dev)
				}
			}
		}
	}
}

func (m *MDNSManager) GetDevices() []*Device {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Device, 0, len(m.devices))
	for _, dev := range m.devices {
		cp := *dev
		result = append(result, &cp)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Online != result[j].Online {
			return result[i].Online
		}
		return result[i].LastSeen > result[j].LastSeen
	})
	return result
}
