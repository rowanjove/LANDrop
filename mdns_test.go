package main

import "testing"

func TestParseTXTIncludesHTTPScheme(t *testing.T) {
	osType, serviceVersion, scheme := parseTXT([]string{"os=windows", "version=2.0.0", "scheme=https"})
	if osType != "windows" || serviceVersion != "2.0.0" || scheme != "https" {
		t.Fatalf("parseTXT() = %q, %q, %q", osType, serviceVersion, scheme)
	}
}

func TestWebClientRegistration(t *testing.T) {
	broker := NewSSEBroker()
	mgr := NewMDNSManager(53217, broker, "TestNode")

	remoteIP := "192.168.10.88"
	mgr.AddWebClient(remoteIP, "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)")

	devs := mgr.GetDevices()
	found := false
	for _, d := range devs {
		if d.Addr == remoteIP {
			found = true
			if d.Name != "iPhone (Web)" || d.OS != "ios" || !d.Online {
				t.Fatalf("unexpected web device: %+v", d)
			}
		}
	}
	if !found {
		t.Fatalf("expected web client %s in device list", remoteIP)
	}

	// Remove web client
	mgr.RemoveWebClient(remoteIP)
	devs = mgr.GetDevices()
	for _, d := range devs {
		if d.Addr == remoteIP && d.Online {
			t.Fatalf("expected web client to be offline, got: %+v", d)
		}
	}

	// Test IPv4-mapped IPv6 address normalization
	mappedIP := "::ffff:192.168.10.99"
	mgr.AddWebClient(mappedIP, "Mozilla/5.0 (Android 14; Mobile)")
	devs = mgr.GetDevices()
	foundMapped := false
	for _, d := range devs {
		if d.Addr == "192.168.10.99" {
			foundMapped = true
			if d.OS != "android" || !d.Online {
				t.Fatalf("unexpected mapped web device: %+v", d)
			}
		}
	}
	if !foundMapped {
		t.Fatalf("expected normalized IP 192.168.10.99 for %s", mappedIP)
	}
	mgr.RemoveWebClient(mappedIP)
}
