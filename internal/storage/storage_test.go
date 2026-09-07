package storage

import (
	"errors"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.db")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestUpsertAndSnapshotGroup(t *testing.T) {
	s := newTestStore(t)

	if err := s.UpsertGroup(1, "First"); err != nil {
		t.Fatalf("UpsertGroup: %v", err)
	}
	if err := s.UpsertGroup(2, "Second"); err != nil {
		t.Fatalf("UpsertGroup: %v", err)
	}

	groups := s.Groups()
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Title != "First" || groups[1].Title != "Second" {
		t.Fatalf("unexpected order: %+v", groups)
	}
}

func TestUpsertGroupUpdatesTitle(t *testing.T) {
	s := newTestStore(t)

	if err := s.UpsertGroup(1, "Old"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertGroup(1, "New"); err != nil {
		t.Fatal(err)
	}

	groups := s.Groups()
	if groups[0].Title != "New" {
		t.Fatalf("expected updated title, got %q", groups[0].Title)
	}
}

func TestAddChannel(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}

	ch := Channel{ID: 10, Username: "chan", Title: "Channel"}
	if err := s.AddChannel(1, ch); err != nil {
		t.Fatalf("AddChannel: %v", err)
	}

	if s.IsChannelForbidden(1, 10) != true {
		t.Fatal("expected channel to be forbidden")
	}
	if s.IsChannelForbidden(1, 99) != false {
		t.Fatal("expected non-listed channel not to be forbidden")
	}
}

func TestAddChannelUnknownGroup(t *testing.T) {
	s := newTestStore(t)
	err := s.AddChannel(42, Channel{ID: 1})
	if !errors.Is(err, ErrNoGroup) {
		t.Fatalf("expected ErrNoGroup, got %v", err)
	}
}

func TestRemoveChannel(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddChannel(1, Channel{ID: 10}); err != nil {
		t.Fatal(err)
	}

	if err := s.RemoveChannel(1, 10); err != nil {
		t.Fatalf("RemoveChannel: %v", err)
	}
	if s.IsChannelForbidden(1, 10) {
		t.Fatal("channel should no longer be forbidden")
	}
}

func TestRemoveMissingChannel(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveChannel(1, 99); !errors.Is(err, ErrNoChannel) {
		t.Fatalf("expected ErrNoChannel, got %v", err)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddChannel(1, Channel{ID: 10}); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	reloaded, err := New(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	defer reloaded.Close()

	if !reloaded.IsChannelForbidden(1, 10) {
		t.Fatal("channel should survive reload")
	}
	groups := reloaded.Groups()
	if len(groups) != 1 || groups[0].ForbiddenChannels[10].ID != 10 {
		t.Fatalf("expected group with channel to survive reload, got %+v", groups)
	}
}

func TestNewCreatesEmptyStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.db")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	if len(s.Groups()) != 0 {
		t.Fatal("expected empty groups")
	}
}
