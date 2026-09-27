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

func TestPINManagerExponentialBackoff(t *testing.T) {
	manager := NewPINManager("8888")
	ip := "10.0.0.1"

	// 4 wrong attempts: should remain unlocked
	for i := 0; i < 4; i++ {
		ok, rem, locked := manager.Verify(ip, "0000")
		if ok || locked || rem != 4-i {
			t.Fatalf("attempt %d: got (ok=%v, rem=%d, locked=%v)", i+1, ok, rem, locked)
		}
	}

	// 5th attempt: triggers first lock
	ok, _, locked := manager.Verify(ip, "0000")
	if ok || !locked {
		t.Fatalf("expected lock after 5 wrong attempts")
	}

	// Check attemptInfo escalates consecutive locks
	manager.mu.Lock()
	info := manager.attempts[ip]
	consecutive := info.consecutiveLocks
	dur := info.currentLockDuration()
	manager.mu.Unlock()

	if consecutive != 1 {
		t.Fatalf("expected 1 consecutive lock, got %d", consecutive)
	}
	if dur != lockDuration {
		t.Fatalf("expected %v lock duration, got %v", lockDuration, dur)
	}
}

