package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const SessionTTL = 24 * time.Hour

// maxSessions is the upper bound on the number of concurrently live sessions.
// If this cap is reached the oldest session is evicted before creating a new
// one, preventing unbounded memory growth from repeated unauthenticated logins.
const maxSessions = 10000

// ErrSessionCapReached is returned when the session store is full and no
// expired session could be evicted to make room.
var ErrSessionCapReached = errors.New("session store is full")

type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
}

func NewSessionStore() *SessionStore {
	ss := &SessionStore{sessions: make(map[string]time.Time)}
	go ss.gc()
	return ss
}

func (ss *SessionStore) Create() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if len(ss.sessions) >= maxSessions {
		// Evict the oldest session to stay within the cap.
		oldest := ""
		var oldestExp time.Time
		for tok, exp := range ss.sessions {
			if oldest == "" || exp.Before(oldestExp) {
				oldest = tok
				oldestExp = exp
			}
		}
		if oldest != "" {
			delete(ss.sessions, oldest)
		} else {
			return "", ErrSessionCapReached
		}
	}

	ss.sessions[token] = time.Now().Add(SessionTTL)
	return token, nil
}

func (ss *SessionStore) IsValid(token string) bool {
	if token == "" {
		return false
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	exp, ok := ss.sessions[token]
	return ok && time.Now().Before(exp)
}

func (ss *SessionStore) Delete(token string) {
	ss.mu.Lock()
	delete(ss.sessions, token)
	ss.mu.Unlock()
}

// gc removes expired sessions every hour.
func (ss *SessionStore) gc() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		ss.mu.Lock()
		for tok, exp := range ss.sessions {
			if now.After(exp) {
				delete(ss.sessions, tok)
			}
		}
		ss.mu.Unlock()
	}
}
