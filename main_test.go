package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReserveDownloadPathKeepsOriginalWhenFree(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")

	got, err := reserveDownloadPath(path)
	if err != nil {
		t.Fatalf("reserveDownloadPath() error = %v", err)
	}
	if got != path {
		t.Fatalf("reserveDownloadPath() = %q, want %q", got, path)
	}
}

func TestReserveDownloadPathAddsSuffixWhenOccupied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := reserveDownloadPath(path)
	if err != nil {
		t.Fatalf("reserveDownloadPath() error = %v", err)
	}

	want := filepath.Join(dir, "report (1).txt")
	if got != want {
		t.Fatalf("reserveDownloadPath() = %q, want %q", got, want)
	}
}
