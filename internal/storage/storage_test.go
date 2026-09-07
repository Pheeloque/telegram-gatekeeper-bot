package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
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

	groups := s.GroupsSnapshot()
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// snapshot is sorted by title
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

	groups := s.GroupsSnapshot()
	if groups[0].Title != "New" {
		t.Fatalf("expected updated title, got %q", groups[0].Title)
	}
}

func TestAddForbiddenChannel(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}

	ch := Channel{ID: 10, Username: "chan", Title: "Channel"}
	if err := s.AddForbiddenChannel(1, ch); err != nil {
		t.Fatalf("AddForbiddenChannel: %v", err)
	}

	if s.IsForbidden(1, 10) != true {
		t.Fatal("expected channel to be forbidden")
	}
	if s.IsForbidden(1, 99) != false {
		t.Fatal("expected non-listed channel not to be forbidden")
	}
}

func TestAddForbiddenChannelUnknownGroup(t *testing.T) {
	s := newTestStore(t)
	err := s.AddForbiddenChannel(42, Channel{ID: 1})
	if !errors.Is(err, ErrNoGroup) {
		t.Fatalf("expected ErrNoGroup, got %v", err)
	}
}

func TestRemoveForbiddenChannel(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddForbiddenChannel(1, Channel{ID: 10}); err != nil {
		t.Fatal(err)
	}

	if err := s.RemoveForbiddenChannel(1, 10); err != nil {
		t.Fatalf("RemoveForbiddenChannel: %v", err)
	}
	if s.IsForbidden(1, 10) {
		t.Fatal("channel should no longer be forbidden")
	}
}

func TestRemoveMissingChannel(t *testing.T) {
	s := newTestStore(t)
	if err := s.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveForbiddenChannel(1, 99); !errors.Is(err, ErrNoChannel) {
		t.Fatalf("expected ErrNoChannel, got %v", err)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddForbiddenChannel(1, Channel{ID: 10}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := New(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reloaded.IsForbidden(1, 10) {
		t.Fatal("channel should survive reload")
	}
}

func TestNewMissingFileCreatesEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(s.Groups) != 0 {
		t.Fatal("expected empty groups")
	}
	os.Remove(path)
}
