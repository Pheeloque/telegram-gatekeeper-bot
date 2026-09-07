package bot

import (
	"testing"
	"time"
)

func TestSessionSelectedGroup(t *testing.T) {
	s := newSession()

	if _, ok := s.GetSelectedGroup(1); ok {
		t.Fatal("expected no selected group initially")
	}

	s.SetSelectedGroup(1, 555)
	id, ok := s.GetSelectedGroup(1)
	if !ok || id != 555 {
		t.Fatalf("expected selected group 555, got %d (ok=%v)", id, ok)
	}
}

func TestSessionAwaiting(t *testing.T) {
	s := newSession()

	if got := s.GetAwaiting(1); got != InputNone {
		t.Fatalf("expected InputNone, got %q", got)
	}

	s.SetAwaiting(1, InputAddChannel)
	if got := s.GetAwaiting(1); got != InputAddChannel {
		t.Fatalf("expected InputAddChannel, got %q", got)
	}

	s.SetAwaiting(1, InputNone)
	if got := s.GetAwaiting(1); got != InputNone {
		t.Fatalf("expected InputNone after reset, got %q", got)
	}
}

func TestSessionIsolation(t *testing.T) {
	s := newSession()

	s.SetSelectedGroup(1, 100)
	s.SetAwaiting(1, InputAddChannel)

	if _, ok := s.GetSelectedGroup(2); ok {
		t.Fatal("user 2 should not share a session")
	}
	if got := s.GetAwaiting(2); got != InputNone {
		t.Fatalf("user 2 awaiting should be InputNone, got %q", got)
	}
}

func TestSessionAwaitingExpires(t *testing.T) {
	s := newSession()

	s.SetAwaiting(1, InputAddChannel)

	s.mu.Lock()
	entry := s.awaiting[1]
	entry.at = time.Now().Add(-(awaitingTTL + time.Minute))
	s.awaiting[1] = entry
	s.mu.Unlock()

	if got := s.GetAwaiting(1); got != InputNone {
		t.Fatalf("expected InputNone after TTL, got %q", got)
	}
	if _, ok := s.awaiting[1]; ok {
		t.Fatal("expired entry should be removed")
	}
}
