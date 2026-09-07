package moderation

import (
	"errors"
	"path/filepath"
	"testing"

	"telegram-gatekeeper-bot/internal/storage"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.db")
	store, err := storage.New(path)
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	if err := store.UpsertGroup(1, "G"); err != nil {
		t.Fatal(err)
	}
	s := New(store)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestAddAndIsForbidden(t *testing.T) {
	s := newTestService(t)

	if err := s.AddChannel(1, storage.Channel{ID: 10}); err != nil {
		t.Fatalf("AddChannel: %v", err)
	}
	if !s.IsForbidden(1, 10) {
		t.Fatal("expected channel forbidden")
	}
	if s.IsForbidden(1, 99) {
		t.Fatal("unexpected channel forbidden")
	}
}

func TestAddChannelUnknownGroup(t *testing.T) {
	s := newTestService(t)
	err := s.AddChannel(42, storage.Channel{ID: 10})
	if !errors.Is(err, ErrNoGroup) {
		t.Fatalf("expected ErrNoGroup, got %v", err)
	}
}

func TestRemoveChannel(t *testing.T) {
	s := newTestService(t)
	if err := s.AddChannel(1, storage.Channel{ID: 10}); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveChannel(1, 10); err != nil {
		t.Fatalf("RemoveChannel: %v", err)
	}
	if s.IsForbidden(1, 10) {
		t.Fatal("channel should be removed")
	}
}

func TestChannels(t *testing.T) {
	s := newTestService(t)
	if err := s.AddChannel(1, storage.Channel{ID: 2, Username: "b"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddChannel(1, storage.Channel{ID: 1, Username: "a"}); err != nil {
		t.Fatal(err)
	}

	channels := s.Channels(1)
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
	if channels[0].Username != "a" || channels[1].Username != "b" {
		t.Fatalf("expected sorted by username, got %+v", channels)
	}
}

func TestEnrichChannel(t *testing.T) {
	s := newTestService(t)
	if err := s.AddChannel(1, storage.Channel{ID: 10}); err != nil {
		t.Fatal(err)
	}
	if err := s.EnrichChannel(1, storage.Channel{ID: 10, Username: "chan", Title: "Channel"}); err != nil {
		t.Fatalf("EnrichChannel: %v", err)
	}

	channels := s.Channels(1)
	if len(channels) != 1 || channels[0].Username != "chan" || channels[0].Title != "Channel" {
		t.Fatalf("expected enriched channel, got %+v", channels)
	}
}
