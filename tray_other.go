//go:build !windows

package main

func HideConsole()   {}
func ShowConsole()   {}
func ToggleConsole() {}

func StartTray(appURL string, deviceName string, onExit func()) (func(), error) {
	return func() {}, nil
}

func RemoveTray() {}
