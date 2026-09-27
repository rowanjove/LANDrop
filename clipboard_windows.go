//go:build windows

package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard          = user32.NewProc("OpenClipboard")
	procCloseClipboard         = user32.NewProc("CloseClipboard")
	procGetClipboardData       = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAvail = user32.NewProc("IsClipboardFormatAvailable")
	procGlobalLock             = kernel32.NewProc("GlobalLock")
	procGlobalUnlock           = kernel32.NewProc("GlobalUnlock")
	procGlobalSize             = kernel32.NewProc("GlobalSize")
)

const cfUnicodeText = 13 // CF_UNICODETEXT

func readClipboardText() (string, error) {
	// Try opening clipboard with retries in case another process currently holds it
	var opened bool
	for i := 0; i < 5; i++ {
		r, _, _ := procOpenClipboard.Call(0)
		if r != 0 {
			opened = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !opened {
		return "", fmt.Errorf("unable to open clipboard")
	}
	defer procCloseClipboard.Call()

	avail, _, _ := procIsClipboardFormatAvail.Call(cfUnicodeText)
	if avail == 0 {
		return "", nil // Clipboard does not contain unicode text
	}

	hMem, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if hMem == 0 {
		return "", nil
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return "", nil
	}
	defer procGlobalUnlock.Call(hMem)

	size, _, _ := procGlobalSize.Call(hMem)
	if size == 0 {
		return "", nil
	}

	// Read UTF-16 units up to null terminator
	maxUnits := size / 2
	slice := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), maxUnits)
	return syscall.UTF16ToString(slice), nil
}
