package bot

import "sync"

type InputStep string

const (
	InputNone          InputStep = ""
	InputAddChannel    InputStep = "add_channel"
	InputRemoveChannel InputStep = "remove_channel"
)

type Session struct {
	mu            sync.RWMutex
	selectedGroup map[int64]int64
	awaiting      map[int64]InputStep
}

func newSession() *Session {
	return &Session{selectedGroup: make(map[int64]int64), awaiting: make(map[int64]InputStep)}
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
	s.awaiting[userID] = step
}

func (s *Session) GetAwaiting(userID int64) InputStep {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.awaiting[userID]
}
