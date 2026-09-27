//go:build !windows

package main

import (
	"os"
	"syscall"
)

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, signal 0 checks if process exists without sending a signal
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func terminateProcessByPID(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

func terminateOtherInstances() error {
	return nil
}
