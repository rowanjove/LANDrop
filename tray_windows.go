//go:build windows

package main

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	modShell32 = syscall.NewLazyDLL("shell32.dll")
	modUser32  = syscall.NewLazyDLL("user32.dll")
	modKernel  = syscall.NewLazyDLL("kernel32.dll")

	procShellNotifyIconW   = modShell32.NewProc("Shell_NotifyIconW")
	procRegisterClassExW   = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW     = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW     = modUser32.NewProc("DefWindowProcW")
	procDestroyWindow      = modUser32.NewProc("DestroyWindow")
	procPostQuitMessage    = modUser32.NewProc("PostQuitMessage")
	procGetMessageW        = modUser32.NewProc("GetMessageW")
	procTranslateMessage   = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW   = modUser32.NewProc("DispatchMessageW")
	procCreatePopupMenu    = modUser32.NewProc("CreatePopupMenu")
	procAppendMenuW        = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenu     = modUser32.NewProc("TrackPopupMenu")
	procDestroyMenu        = modUser32.NewProc("DestroyMenu")
	procGetCursorPos       = modUser32.NewProc("GetCursorPos")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
	procLoadIconW          = modUser32.NewProc("LoadIconW")
	procShowWindow         = modUser32.NewProc("ShowWindow")
	procEmptyClipboard     = modUser32.NewProc("EmptyClipboard")
	procSetClipboardData   = modUser32.NewProc("SetClipboardData")

	procGetModuleHandleW = modKernel.NewProc("GetModuleHandleW")
	procGetConsoleWindow = modKernel.NewProc("GetConsoleWindow")
	procGlobalAlloc      = modKernel.NewProc("GlobalAlloc")
	procGlobalFree       = modKernel.NewProc("GlobalFree")
)

const (
	wmUser      = 0x0400
	wmTrayIcon  = wmUser + 1
	wmLBtnDbl   = 0x0203
	wmLBtnUp    = 0x0202
	wmRBtnUp    = 0x0205

	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	mfString    = 0x00000000
	mfSeparator = 0x00000800

	tpmRightButton = 0x0002

	swHide = 0
	swShow = 5

	idmOpenWeb       = 1001
	idmCopyUrl       = 1002
	idmToggleConsole = 1003
	idmExit          = 1004

	idiApplication = 32512
)

type point struct {
	x, y int32
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type notifyIconDataW struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uTimeoutOrVersion uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

var (
	trayHwnd      uintptr
	trayNID       notifyIconDataW
	trayURL       string
	trayOnExit    func()
	consoleHidden bool
)

func HideConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, swHide)
		consoleHidden = true
	}
}

func ShowConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, swShow)
		consoleHidden = false
	}
}

func ToggleConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		if consoleHidden {
			procShowWindow.Call(hwnd, swShow)
			consoleHidden = false
		} else {
			procShowWindow.Call(hwnd, swHide)
			consoleHidden = true
		}
	}
}

func trayWndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	switch msg {
	case wmTrayIcon:
		switch lParam {
		case wmLBtnDbl, wmLBtnUp:
			if trayURL != "" {
				_ = openBrowser(trayURL)
			}
		case wmRBtnUp:
			var pt point
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			procSetForegroundWindow.Call(hwnd)

			hMenu, _, _ := procCreatePopupMenu.Call()
			if hMenu != 0 {
				title, _ := syscall.UTF16PtrFromString("LANDrop - 局域网分享")
				openWeb, _ := syscall.UTF16PtrFromString("打开网页界面")
				copyUrl, _ := syscall.UTF16PtrFromString("复制访问地址")
				toggleCon, _ := syscall.UTF16PtrFromString("显示/隐藏控制台")
				exitText, _ := syscall.UTF16PtrFromString("退出 LANDrop")

				procAppendMenuW.Call(hMenu, mfString, uintptr(idmOpenWeb), uintptr(unsafe.Pointer(title)))
				procAppendMenuW.Call(hMenu, mfSeparator, 0, 0)
				procAppendMenuW.Call(hMenu, mfString, uintptr(idmOpenWeb), uintptr(unsafe.Pointer(openWeb)))
				procAppendMenuW.Call(hMenu, mfString, uintptr(idmCopyUrl), uintptr(unsafe.Pointer(copyUrl)))
				procAppendMenuW.Call(hMenu, mfString, uintptr(idmToggleConsole), uintptr(unsafe.Pointer(toggleCon)))
				procAppendMenuW.Call(hMenu, mfSeparator, 0, 0)
				procAppendMenuW.Call(hMenu, mfString, uintptr(idmExit), uintptr(unsafe.Pointer(exitText)))

				cmd, _, _ := procTrackPopupMenu.Call(hMenu, tpmRightButton, uintptr(pt.x), uintptr(pt.y), 0, hwnd, 0)
				_ = cmd
				procDestroyMenu.Call(hMenu)
			}
		}
		return 0

	case 0x0111: // WM_COMMAND
		cmdID := int(wParam & 0xffff)
		switch cmdID {
		case idmOpenWeb:
			if trayURL != "" {
				_ = openBrowser(trayURL)
			}
		case idmCopyUrl:
			if trayURL != "" {
				_ = copyClipboardTextWindows(trayURL)
			}
		case idmToggleConsole:
			ToggleConsole()
		case idmExit:
			RemoveTray()
			if trayOnExit != nil {
				go trayOnExit()
			}
		}
		return 0

	case 0x0010: // WM_CLOSE
		procDestroyWindow.Call(hwnd)
		return 0

	case 0x0002: // WM_DESTROY
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

// StartTray initializes and displays the system tray icon in a dedicated thread.
func StartTray(appURL string, deviceName string, onExit func()) (func(), error) {
	trayURL = appURL
	trayOnExit = onExit

	started := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		hInst, _, _ := procGetModuleHandleW.Call(0)
		className, _ := syscall.UTF16PtrFromString("LANDropTrayWindowClass")

		// Register hidden window class
		wc := wndClassExW{
			cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
			lpfnWndProc:   syscall.NewCallback(trayWndProc),
			hInstance:     hInst,
			lpszClassName: className,
		}

		// Try loading embedded icon from resources (id 1 from syso), fallback to system icon
		hIcon, _, _ := procLoadIconW.Call(hInst, uintptr(1))
		if hIcon == 0 {
			hIcon, _, _ = procLoadIconW.Call(0, uintptr(idiApplication))
		}
		wc.hIcon = hIcon

		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

		// Create message-only window
		hwnd, _, _ := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(className)),
			0,
			0, 0, 0, 0,
			0, 0, hInst, 0,
		)

		if hwnd == 0 {
			started <- fmt.Errorf("failed to create tray message window")
			return
		}
		trayHwnd = hwnd

		// Initialize NOTIFYICONDATAW
		trayNID = notifyIconDataW{
			cbSize:           uint32(unsafe.Sizeof(notifyIconDataW{})),
			hWnd:             hwnd,
			uID:              1,
			uFlags:           nifMessage | nifIcon | nifTip,
			uCallbackMessage: wmTrayIcon,
			hIcon:            hIcon,
		}

		tipText := fmt.Sprintf("LANDrop - %s (%s)", deviceName, appURL)
		if len(tipText) > 120 {
			tipText = tipText[:120]
		}
		tipUTF16, _ := syscall.UTF16FromString(tipText)
		copy(trayNID.szTip[:], tipUTF16)

		procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&trayNID)))

		started <- nil

		// Standard Win32 Message Loop
		var m msg
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(r) <= 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}()

	err := <-started
	if err != nil {
		return nil, err
	}
	return RemoveTray, nil
}

// RemoveTray removes the tray icon from the notification area.
func RemoveTray() {
	if trayHwnd != 0 {
		procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&trayNID)))
		trayHwnd = 0
	}
}

// copyClipboardTextWindows is a Win32 clipboard writer helper for the tray menu
func copyClipboardTextWindows(text string) error {
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
		return fmt.Errorf("unable to open clipboard")
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}

	size := len(utf16) * 2
	hMem, _, _ := procGlobalAlloc.Call(0x0002 /* GMEM_MOVEABLE */, uintptr(size))
	if hMem == 0 {
		return fmt.Errorf("GlobalAlloc failed")
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("GlobalLock failed")
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(utf16))
	copy(dst, utf16)
	procGlobalUnlock.Call(hMem)

	r, _, _ := procSetClipboardData.Call(cfUnicodeText, hMem)
	if r == 0 {
		procGlobalFree.Call(hMem)
		return fmt.Errorf("SetClipboardData failed")
	}
	return nil
}
