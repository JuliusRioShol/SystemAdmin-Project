package data

import (
	"sync"
	"time"
)

// User represents a user
type User struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  []byte    `json:"-"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Message represents a discussion board message
type Message struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
}

// Token represents an authentication token
type Token struct {
	Plaintext string    `json:"plaintext"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"user_id"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"scope"`
}

// SessionManager handles user sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]int64 // token -> userID
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]int64),
	}
}

func (sm *SessionManager) Set(token string, userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[token] = userID
}

func (sm *SessionManager) Get(token string) (int64, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	userID, exists := sm.sessions[token]
	return userID, exists
}

func (sm *SessionManager) Delete(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, token)
}

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)
