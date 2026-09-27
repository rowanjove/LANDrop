package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func getPidFilePath() string {
	return filepath.Join(getStateDir(), "landrop.pid")
}

// EnsureSingleInstance ensures only one instance of LANDrop server runs at any time.
// If an existing instance is running, it will be gracefully or forcefully terminated
// to let the new instance take over.
func EnsureSingleInstance(port int) error {
	myPID := os.Getpid()
	pidFile := getPidFilePath()

	// 1. Check PID file and try to shut down previous instance
	if data, err := os.ReadFile(pidFile); err == nil {
		oldPIDStr := strings.TrimSpace(string(data))
		if oldPID, err := strconv.Atoi(oldPIDStr); err == nil && oldPID > 0 && oldPID != myPID {
			// Try graceful shutdown via HTTP first
			notifyShutdownHTTP(port)

			// Wait up to 500ms for old process to exit
			exited := waitForProcessExit(oldPID, 500*time.Millisecond)
			if !exited {
				_ = terminateProcessByPID(oldPID)
			}
		}
	}

	// 2. Platform-specific check for any other orphan instances of the same executable
	_ = terminateOtherInstances()

	// 3. Wait for the target port to be completely freed up
	waitForPortFree(port, 1500*time.Millisecond)

	// 4. Record current PID
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(myPID)), 0o644)
	return nil
}

// ReleaseSingleInstance removes the PID file if it matches the current process.
func ReleaseSingleInstance() {
	pidFile := getPidFilePath()
	if data, err := os.ReadFile(pidFile); err == nil {
		if strings.TrimSpace(string(data)) == strconv.Itoa(os.Getpid()) {
			_ = os.Remove(pidFile)
		}
	}
}

func notifyShutdownHTTP(port int) {
	client := &http.Client{Timeout: 400 * time.Millisecond}
	url := fmt.Sprintf("http://127.0.0.1:%d/api/shutdown", port)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err == nil {
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
}

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isProcessRunning(pid) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !isProcessRunning(pid)
}

func waitForPortFree(port int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
		if err != nil {
			// Port is free
			return
		}
		_ = conn.Close()
		time.Sleep(60 * time.Millisecond)
	}
}
