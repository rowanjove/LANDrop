//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	procCreateToolhelp32Snapshot = modKernel.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = modKernel.NewProc("Process32FirstW")
	procProcess32NextW           = modKernel.NewProc("Process32NextW")
	procOpenProcess              = modKernel.NewProc("OpenProcess")
	procTerminateProcess         = modKernel.NewProc("TerminateProcess")
	procCloseHandle              = modKernel.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = modKernel.NewProc("QueryFullProcessImageNameW")
)

const (
	th32csSnapProcess  = 0x00000002
	processTerminate   = 0x0001
	processQueryInfo   = 0x0400
	processQueryLimited = 0x1000
	invalidHandleValue = ^uintptr(0)
)

type processEntry32W struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

func isProcessRunning(pid int) bool {
	hProcess, _, _ := procOpenProcess.Call(uintptr(processQueryInfo), 0, uintptr(pid))
	if hProcess == 0 {
		return false
	}
	procCloseHandle.Call(hProcess)
	return true
}

func getProcessExePath(pid int) (string, error) {
	hProcess, _, _ := procOpenProcess.Call(uintptr(processQueryLimited), 0, uintptr(pid))
	if hProcess == 0 {
		return "", syscall.GetLastError()
	}
	defer procCloseHandle.Call(hProcess)

	var buf [1024]uint16
	size := uint32(len(buf))
	r, _, err := procQueryFullProcessImageNameW.Call(hProcess, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buf[:size]), nil
}

func terminateProcessByPID(pid int) error {
	hProcess, _, _ := procOpenProcess.Call(uintptr(processTerminate), 0, uintptr(pid))
	if hProcess == 0 {
		// Fallback to os.FindProcess
		if p, err := os.FindProcess(pid); err == nil {
			return p.Kill()
		}
		return nil
	}
	defer procCloseHandle.Call(hProcess)
	r, _, err := procTerminateProcess.Call(hProcess, 0)
	if r == 0 {
		return err
	}
	return nil
}

func terminateOtherInstances() error {
	myPID := uint32(os.Getpid())
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	myExeName := strings.ToLower(filepath.Base(exePath))

	hSnap, _, _ := procCreateToolhelp32Snapshot.Call(uintptr(th32csSnapProcess), 0)
	if hSnap == 0 || hSnap == invalidHandleValue {
		return nil
	}
	defer procCloseHandle.Call(hSnap)

	var entry processEntry32W
	entry.dwSize = uint32(unsafe.Sizeof(entry))

	r, _, _ := procProcess32FirstW.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
	if r == 0 {
		return nil
	}

	for {
		if entry.th32ProcessID != myPID && entry.th32ProcessID > 0 {
			name := strings.ToLower(syscall.UTF16ToString(entry.szExeFile[:]))
			if name == myExeName {
				// Don't kill arbitrary processes if running under generic names in dev/test
				if myExeName == "main.exe" || strings.HasSuffix(myExeName, ".test.exe") {
					if otherPath, err := getProcessExePath(int(entry.th32ProcessID)); err == nil {
						if strings.EqualFold(otherPath, exePath) {
							_ = terminateProcessByPID(int(entry.th32ProcessID))
						}
					}
				} else {
					if otherPath, err := getProcessExePath(int(entry.th32ProcessID)); err == nil {
						if strings.EqualFold(otherPath, exePath) {
							_ = terminateProcessByPID(int(entry.th32ProcessID))
						}
					} else {
						_ = terminateProcessByPID(int(entry.th32ProcessID))
					}
				}
			}
		}
		r, _, _ = procProcess32NextW.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
		if r == 0 {
			break
		}
	}
	return nil
}
