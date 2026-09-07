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

type Service struct {
	store *storage.Store
}

func New(store *storage.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Close() error {
	return s.store.Close()
}

func (s *Service) IsForbidden(groupID, channelID int64) bool {
	return s.store.IsForbidden(groupID, channelID)
}

func (s *Service) AddChannel(groupID int64, channel storage.Channel) error {
	if err := s.store.AddForbiddenChannel(groupID, channel); err != nil {
		if errors.Is(err, storage.ErrNoGroup) {
			return ErrNoGroup
		}
		return fmt.Errorf("%w: %v", ErrAddChannel, err)
	}
	return nil
}

func (s *Service) RemoveChannel(groupID, channelID int64) error {
	if err := s.store.RemoveForbiddenChannel(groupID, channelID); err != nil {
		if errors.Is(err, storage.ErrNoGroup) {
			return ErrNoGroup
		}
		return fmt.Errorf("%w: %v", ErrDelChannel, err)
	}
	return nil
}

func (s *Service) Channels(groupID int64) []storage.Channel {
	return s.store.GetForbiddenChannels(groupID)
}

func (s *Service) IsGroupRegistered(groupID int64) bool {
	groups := s.store.GroupsSnapshot()
	for _, g := range groups {
		if g.ID == groupID {
			return true
		}
	}
	return false
}
