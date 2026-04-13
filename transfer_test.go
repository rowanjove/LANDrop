package main

import "testing"

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
