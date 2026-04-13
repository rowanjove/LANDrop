package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestDownloadFileWithClientRejectsUnexpectedFullResponseDuringResume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("abcdef"))
	}))
	defer server.Close()

	dir := t.TempDir()
	savePath := filepath.Join(dir, "demo.txt")
	if err := os.WriteFile(savePath, []byte("abc"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Range", "bytes=3-")

	if _, err := downloadFileWithClient(server.Client(), req, savePath, 3); err == nil {
		t.Fatal("downloadFileWithClient() error = nil, want resume validation failure")
	}

	data, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := string(data); got != "abc" {
		t.Fatalf("file content after rejected resume = %q, want %q", got, "abc")
	}
}

func TestDownloadFileWithClientAppendsOnValidResumeResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Range"); got != "bytes=3-" {
			t.Fatalf("Range header = %q, want %q", got, "bytes=3-")
		}
		w.Header().Set("Content-Range", "bytes 3-5/6")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("def"))
	}))
	defer server.Close()

	dir := t.TempDir()
	savePath := filepath.Join(dir, "demo.txt")
	if err := os.WriteFile(savePath, []byte("abc"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Range", "bytes=3-")

	gotPath, err := downloadFileWithClient(server.Client(), req, savePath, 3)
	if err != nil {
		t.Fatalf("downloadFileWithClient() error = %v", err)
	}
	if gotPath != savePath {
		t.Fatalf("downloadFileWithClient() path = %q, want %q", gotPath, savePath)
	}

	data, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := string(data); got != "abcdef" {
		t.Fatalf("resumed file content = %q, want %q", got, "abcdef")
	}
}

func TestParseContentRange(t *testing.T) {
	start, end, ok := parseContentRange("bytes 10-19/20")
	if !ok {
		t.Fatal("parseContentRange() = not ok, want ok")
	}
	if start != 10 || end != 19 {
		t.Fatalf("parseContentRange() = (%d, %d), want (10, 19)", start, end)
	}

	if _, _, ok := parseContentRange(strings.TrimSpace("")); ok {
		t.Fatal("parseContentRange(\"\") = ok, want false")
	}
}
