package main

import "testing"

func TestPINManagerSessionsAreRandomAndIPBound(t *testing.T) {
	manager := NewPINManager("1234")

	first, err := manager.createSession("192.168.0.10")
	if err != nil {
		t.Fatalf("createSession() error = %v", err)
	}
	second, err := manager.createSession("192.168.0.10")
	if err != nil {
		t.Fatalf("createSession() error = %v", err)
	}

	if first == second {
		t.Fatal("expected unique session tokens")
	}
	if !manager.validSession("192.168.0.10", first) {
		t.Fatal("expected token to validate for the issuing IP")
	}
	if manager.validSession("192.168.0.11", first) {
		t.Fatal("expected token reuse from a different IP to fail")
	}
}
