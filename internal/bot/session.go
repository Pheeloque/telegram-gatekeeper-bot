package bot

import (
	"sync"
	"time"
)

type InputStep string

const (
	InputNone          InputStep = ""
	InputAddChannel    InputStep = "add_channel"
	InputRemoveChannel InputStep = "remove_channel"
)

// awaitingTTL determines how long an awaited input step stays valid.
const awaitingTTL = 30 * time.Minute

type Session struct {
	mu            sync.RWMutex
	selectedGroup map[int64]int64
	awaiting      map[int64]awaitingEntry
}

type awaitingEntry struct {
	step InputStep
	at   time.Time
}

func newSession() *Session {
	return &Session{
		selectedGroup: make(map[int64]int64),
		awaiting:      make(map[int64]awaitingEntry),
	}
}

func (s *Session) SetSelectedGroup(userID, groupID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectedGroup[userID] = groupID
}

func (s *Session) GetSelectedGroup(userID int64) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.selectedGroup[userID]
	return id, ok
}

func (s *Session) SetAwaiting(userID int64, step InputStep) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if step == InputNone {
		delete(s.awaiting, userID)
		return
	}
	s.awaiting[userID] = awaitingEntry{step: step, at: time.Now()}
}

func (s *Session) GetAwaiting(userID int64) InputStep {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.awaiting[userID]
	if !ok {
		return InputNone
	}
	if time.Since(entry.at) > awaitingTTL {
		delete(s.awaiting, userID)
		return InputNone
	}
	return entry.step
}
