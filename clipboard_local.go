package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runTextCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, stderr.String())
		}
		return "", err
	}
	return string(out), nil
}

func readClipboardText() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return runTextCommand("powershell", "-NoProfile", "-Command", "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; $text = Get-Clipboard -Raw; if ($null -ne $text) { [Console]::Out.Write($text) }")
	case "darwin":
		return runTextCommand("pbpaste")
	default:
		switch {
		case commandExists("wl-paste"):
			return runTextCommand("wl-paste", "-n")
		case commandExists("xclip"):
			return runTextCommand("xclip", "-selection", "clipboard", "-o")
		case commandExists("xsel"):
			return runTextCommand("xsel", "--clipboard", "--output")
		default:
			return "", fmt.Errorf("no clipboard reader available; install wl-paste, xclip, or xsel")
		}
	}
}
