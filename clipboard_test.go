package main

import (
	"testing"
)

func TestReadClipboardText(t *testing.T) {
	// Should not crash or panic when querying system clipboard
	_, err := readClipboardText()
	if err != nil {
		t.Logf("readClipboardText returned error (normal if clipboard locked): %v", err)
	}
}
