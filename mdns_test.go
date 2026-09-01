package main

import "testing"

func TestParseTXTIncludesHTTPScheme(t *testing.T) {
	osType, serviceVersion, scheme := parseTXT([]string{"os=windows", "version=2.0.0", "scheme=https"})
	if osType != "windows" || serviceVersion != "2.0.0" || scheme != "https" {
		t.Fatalf("parseTXT() = %q, %q, %q", osType, serviceVersion, scheme)
	}
}
