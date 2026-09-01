package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Keep package tests from appending records to the developer's real history.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "landrop-test-state-")
	if err != nil {
		os.Exit(1)
	}
	_ = os.Setenv("LANDROP_STATE_DIR", filepath.Clean(dir))
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
