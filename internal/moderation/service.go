package moderation

import (
	"errors"
	"fmt"

	"telegram-gatekeeper-bot/internal/storage"
)

var (
	ErrNoGroup    = errors.New("group is not registered")
	ErrNoChannel  = errors.New("channel is not in group")
	ErrAddChannel = errors.New("failed to add channel")
	ErrDelChannel = errors.New("failed to remove channel")
)

// Storage describes the persistence operations the Service needs.
type Storage interface {
	UpsertGroup(id int64, title string) error
	GroupTitle(id int64) (string, bool)
	Groups() []storage.Group
	Channels(groupID int64) []storage.Channel
	AddChannel(groupID int64, channel storage.Channel) error
	EnrichChannel(groupID int64, channel storage.Channel) error
	RemoveChannel(groupID, channelID int64) error
	IsChannelForbidden(groupID, channelID int64) bool
}

type Service struct {
	store Storage
}

func New(store Storage) *Service {
	return &Service{store: store}
}

func (s *Service) Close() error {
	if c, ok := s.store.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}

func (s *Service) UpsertGroup(id int64, title string) error {
	return s.store.UpsertGroup(id, title)
}

func (s *Service) Groups() []storage.Group {
	return s.store.Groups()
}

func (s *Service) GroupTitle(id int64) (string, bool) {
	return s.store.GroupTitle(id)
}

func (s *Service) IsForbidden(groupID, channelID int64) bool {
	return s.store.IsChannelForbidden(groupID, channelID)
}

func (s *Service) AddChannel(groupID int64, channel storage.Channel) error {
	if err := s.store.AddChannel(groupID, channel); err != nil {
		if errors.Is(err, storage.ErrNoGroup) {
			return ErrNoGroup
		}
		return fmt.Errorf("%w: %v", ErrAddChannel, err)
	}
	return nil
}

func (s *Service) RemoveChannel(groupID, channelID int64) error {
	if err := s.store.RemoveChannel(groupID, channelID); err != nil {
		if errors.Is(err, storage.ErrNoGroup) {
			return ErrNoGroup
		}
		return fmt.Errorf("%w: %v", ErrDelChannel, err)
	}
	return nil
}

func (s *Service) Channels(groupID int64) []storage.Channel {
	return s.store.Channels(groupID)
}

func (s *Service) EnrichChannel(groupID int64, channel storage.Channel) error {
	return s.store.EnrichChannel(groupID, channel)
}
