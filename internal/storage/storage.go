package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

var (
	ErrNoGroup   = errors.New("group is not registered")
	ErrNoChannel = errors.New("channel is not in group")
)

type Channel struct {
	ID       int64  `json:"id"`
	Username string `json:"username,omitempty"`
	Title    string `json:"title"`
}

type Group struct {
	ID                int64             `json:"id"`
	Title             string            `json:"title"`
	ForbiddenChannels map[int64]Channel `json:"forbidden_channels"`
}

type Store struct {
	Groups map[int64]*Group `json:"groups"`

	mu   sync.RWMutex
	path string
}

func New(path string) (*Store, error) {
	s := &Store{Groups: make(map[int64]*Group), path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("decode storage: %w", err)
	}
	if s.Groups == nil {
		s.Groups = make(map[int64]*Group)
	}
	for _, group := range s.Groups {
		if group.ForbiddenChannels == nil {
			group.ForbiddenChannels = make(map[int64]Channel)
		}
	}
	return s, nil
}

func (s *Store) save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *Store) UpsertGroup(id int64, title string) error {
	s.mu.Lock()
	group, ok := s.Groups[id]
	changed := false
	if !ok {
		group = &Group{ID: id, Title: title, ForbiddenChannels: make(map[int64]Channel)}
		s.Groups[id] = group
		changed = true
	} else if title != "" && group.Title != title {
		group.Title = title
		changed = true
	}
	s.mu.Unlock()
	if !changed {
		return nil
	}
	return s.save()
}

func (s *Store) GroupsSnapshot() []Group {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Group, 0, len(s.Groups))
	for _, group := range s.Groups {
		copyGroup := Group{ID: group.ID, Title: group.Title, ForbiddenChannels: make(map[int64]Channel, len(group.ForbiddenChannels))}
		for id, channel := range group.ForbiddenChannels {
			copyGroup.ForbiddenChannels[id] = channel
		}
		result = append(result, copyGroup)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Title < result[j].Title })
	return result
}

func (s *Store) GetForbiddenChannels(groupID int64) []Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	group, ok := s.Groups[groupID]
	if !ok {
		return nil
	}
	channels := make([]Channel, 0, len(group.ForbiddenChannels))
	for _, channel := range group.ForbiddenChannels {
		channels = append(channels, channel)
	}
	sort.Slice(channels, func(i, j int) bool {
		return DisplayChannel(channels[i]) < DisplayChannel(channels[j])
	})
	return channels
}

func (s *Store) AddForbiddenChannel(groupID int64, channel Channel) error {
	s.mu.Lock()
	group, ok := s.Groups[groupID]
	if !ok {
		s.mu.Unlock()
		return ErrNoGroup
	}
	if group.ForbiddenChannels == nil {
		group.ForbiddenChannels = make(map[int64]Channel)
	}
	group.ForbiddenChannels[channel.ID] = channel
	s.mu.Unlock()
	return s.save()
}

func (s *Store) RemoveForbiddenChannel(groupID, channelID int64) error {
	s.mu.Lock()
	group, ok := s.Groups[groupID]
	if ok {
		if _, exists := group.ForbiddenChannels[channelID]; !exists {
			s.mu.Unlock()
			return ErrNoChannel
		}
		delete(group.ForbiddenChannels, channelID)
	}
	s.mu.Unlock()
	if !ok {
		return ErrNoGroup
	}
	return s.save()
}

func (s *Store) IsForbidden(groupID, channelID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	group, ok := s.Groups[groupID]
	if !ok {
		return false
	}
	_, ok = group.ForbiddenChannels[channelID]
	return ok
}

func DisplayChannel(channel Channel) string {
	if channel.Username != "" {
		return "@" + strings.TrimPrefix(channel.Username, "@")
	}
	if channel.Title != "" {
		return channel.Title
	}
	return fmt.Sprintf("%d", channel.ID)
}
