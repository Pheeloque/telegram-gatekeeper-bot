package moderation

import "telegram-gatekeeper-bot/internal/storage"

type Service struct {
	store *storage.Store
}

func New(store *storage.Store) *Service {
	return &Service{store: store}
}

func (s *Service) IsForbidden(groupID, channelID int64) bool {
	return s.store.IsForbidden(groupID, channelID)
}

func (s *Service) AddChannel(groupID int64, channel storage.Channel) error {
	return s.store.AddForbiddenChannel(groupID, channel)
}

func (s *Service) RemoveChannel(groupID, channelID int64) error {
	return s.store.RemoveForbiddenChannel(groupID, channelID)
}

func (s *Service) Channels(groupID int64) []storage.Channel {
	return s.store.GetForbiddenChannels(groupID)
}
