package main

import "testing"

func TestTransferStoreOneTimeClaimsAreExclusive(t *testing.T) {
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

	store.FinishDownload(item.Token, false)

	if _, found, unavailable := store.BeginDownload(item.Token); !found || unavailable {
		t.Fatalf("BeginDownload() after failed transfer = found %v unavailable %v, want found=true unavailable=false", found, unavailable)
	}

	store.FinishDownload(item.Token, true)

	if _, ok := store.Get(item.Token); ok {
		t.Fatal("expected one-time token to be removed after a successful transfer")
	}
}
