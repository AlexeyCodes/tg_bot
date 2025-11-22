package states

import (
	"sync"
	"tgbot/models"
)

type State string

const (
	StateIdle          State = "idle"
	WaitingName        State = "waiting_name"
	WaitingLastName    State = "waiting_lastname"
	WaitingClass       State = "waiting_class"
	ChoosingDiscipline State = "choosing_discipline"
	ReadingRules       State = "reading_rules"
	EnteringNick       State = "entering_nick"
	EnteringTag        State = "entering_tag"
	TriathlonSelect    State = "triathlon_select"
)

type Session struct {
	State       State
	Temp        *models.User
	CurrentGame string
	TriGames    map[string]bool
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[int64]*Session
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[int64]*Session)}
}

func (m *Manager) Get(userID int64) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[userID]
	if !ok {
		s = &Session{State: StateIdle, Temp: &models.User{Disciplines: make(map[string]models.GameData)}, TriGames: make(map[string]bool)}
		m.sessions[userID] = s
	}
	return s
}

func (m *Manager) SetState(userID int64, st State) {
	m.mu.Lock()
	if s, ok := m.sessions[userID]; ok {
		s.State = st
	} else {
		m.sessions[userID] = &Session{State: st, Temp: &models.User{Disciplines: make(map[string]models.GameData)}, TriGames: make(map[string]bool)}
	}
	m.mu.Unlock()
}

func (m *Manager) Reset(userID int64) {
	m.mu.Lock()
	delete(m.sessions, userID)
	m.mu.Unlock()
}
