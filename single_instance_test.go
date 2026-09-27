package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

func TestSingleInstancePIDLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "landrop-single-inst-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldEnv := os.Getenv("LANDROP_STATE_DIR")
	_ = os.Setenv("LANDROP_STATE_DIR", tempDir)
	defer func() {
		if oldEnv != "" {
			_ = os.Setenv("LANDROP_STATE_DIR", oldEnv)
		} else {
			_ = os.Unsetenv("LANDROP_STATE_DIR")
		}
	}()

	pidFile := getPidFilePath()

	// 1. Ensure initial state
	ReleaseSingleInstance()
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("expected pid file to not exist initially")
	}

	// 2. EnsureSingleInstance for an unused port
	err = EnsureSingleInstance(54321)
	if err != nil {
		t.Fatalf("EnsureSingleInstance returned error: %v", err)
	}

	// 3. Verify PID file contains current PID
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("failed to read pid file: %v", err)
	}
	if string(data) != strconv.Itoa(os.Getpid()) {
		t.Fatalf("expected pid %d, got %s", os.Getpid(), string(data))
	}

	// 4. ReleaseSingleInstance cleans up
	ReleaseSingleInstance()
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("expected pid file to be removed after release")
	}
}

func TestShutdownEndpointRejectsRemoteIP(t *testing.T) {
	app := NewApp("127.0.0.1:53217", "")
	mux := http.NewServeMux()
	app.SetupRoutes(mux)

	// Simulated remote client
	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for remote caller, got %d", rr.Code)
	}

	// Simulated loopback client
	reqLocal := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	reqLocal.RemoteAddr = "127.0.0.1:54321"
	rrLocal := httptest.NewRecorder()
	mux.ServeHTTP(rrLocal, reqLocal)

	if rrLocal.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for loopback caller, got %d", rrLocal.Code)
	}
}
