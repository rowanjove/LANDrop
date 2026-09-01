package main

import (
	"strings"
	"testing"
)

func TestTransferStoreOneTimeClaimsAreExclusive(t *testing.T) {
	ClearHistory()
	store := NewTransferStore()
	defer store.Cleanup()

	item, err := store.AddText("hello")
	if err != nil {
		t.Fatalf("AddText() error = %v", err)
	}
	item.OneTimeUse = true

	if _, found, unavailable := store.BeginDownload(item.Token); !found || unavailable {
		t.Fatalf("first BeginDownload() = found %v unavailable %v, want found=true unavailable=false", found, unavailable)
	}
	if _, found, unavailable := store.BeginDownload(item.Token); !found || !unavailable {
		t.Fatalf("second BeginDownload() = found %v unavailable %v, want found=true unavailable=true", found, unavailable)
	}

	store.FinishDownload(item.Token, DownloadFailed, "test_peer")

	if _, found, unavailable := store.BeginDownload(item.Token); !found || unavailable {
		t.Fatalf("BeginDownload() after failed transfer = found %v unavailable %v, want found=true unavailable=false", found, unavailable)
	}

	store.FinishDownload(item.Token, DownloadCompleted, "test_peer")

	if _, ok := store.Get(item.Token); ok {
		t.Fatal("expected one-time token to be removed after a successful transfer")
	}
}

func TestTransferStoreInterruptedOneTimeTransferReleasesClaim(t *testing.T) {
	ClearHistory()
	store := NewTransferStore()
	defer store.Cleanup()

	item, err := store.AddText("hello")
	if err != nil {
		t.Fatalf("AddText() error = %v", err)
	}
	item.OneTimeUse = true

	if _, found, unavailable := store.BeginDownload(item.Token); !found || unavailable {
		t.Fatalf("BeginDownload() = found %v unavailable %v, want found=true unavailable=false", found, unavailable)
	}

	store.FinishDownload(item.Token, DownloadInterrupted, "test_peer")

	if _, found, unavailable := store.BeginDownload(item.Token); !found || unavailable {
		t.Fatalf("BeginDownload() after interruption = found %v unavailable %v, want found=true unavailable=false", found, unavailable)
	}

	records := GetHistoryRecords()
	if len(records) == 0 {
		t.Fatal("expected interrupted transfer to be recorded in history")
	}
	if got := records[len(records)-1].Status; got != "interrupted" {
		t.Fatalf("latest history status = %q, want %q", got, "interrupted")
	}
}

func TestAddTextRejectsLargePayloadWithCurrentLimit(t *testing.T) {
	store := NewTransferStore()
	defer store.Cleanup()

	_, err := store.AddText(strings.Repeat("x", maxTextSize+1))
	if err == nil {
		t.Fatal("AddText() error = nil, want size validation failure")
	}
	if !strings.Contains(err.Error(), "10 MB") {
		t.Fatalf("AddText() error = %q, want updated 10 MB limit message", err.Error())
	}
}

func TestTransferStoreCleanupExpired(t *testing.T) {
	store := NewTransferStore()
	defer store.Cleanup()

	item, err := store.AddText("expired text")
	if err != nil {
		t.Fatalf("AddText() error = %v", err)
	}
	item.ExpiresAt = 1 // Already in the past

	cleaned := store.CleanupExpired()
	if cleaned != 1 {
		t.Fatalf("CleanupExpired() = %d, want 1", cleaned)
	}

	if _, ok := store.Get(item.Token); ok {
		t.Fatal("expected expired token to be removed from store")
	}
}

func TestTransferStoreCancel(t *testing.T) {
	store := NewTransferStore()
	defer store.Cleanup()

	item, err := store.AddText("to be cancelled")
	if err != nil {
		t.Fatalf("AddText() error = %v", err)
	}

	item.Status = TransferStatusTransferring
	ok := store.Cancel(item.Token)
	if !ok {
		t.Fatal("Cancel() = false, want true")
	}

	if _, ok := store.Get(item.Token); ok {
		t.Fatal("expected cancelled token to be removed from active items")
	}
	if !store.IsCancelled(item.Token) {
		t.Fatal("expected cancellation marker for an in-flight transfer")
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"normal.txt", "normal.txt"},
		{"../../evil.sh", "evil.sh"},
		{"..\\..\\evil.exe", "evil.exe"},
		{"/root/etc/passwd", "passwd"},
		{"CON.txt", "_CON.txt"},
		{"aux", "_aux"},
		{"NUL.tar.gz", "_NUL.tar.gz"},
		{"", "unnamed"},
		{".", "unnamed"},
		{"..", "unnamed"},
		{"hello\x00world.pdf", "helloworld.pdf"},
		{"  spaced.doc  ", "spaced.doc"},
	}

	for _, c := range cases {
		got := sanitizeFilename(c.input)
		if got != c.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
