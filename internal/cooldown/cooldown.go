package cooldown

import (
	"sync"
	"time"
)

// Manager tracks per-user, per-command cooldowns.
type Manager struct {
	mu        sync.Mutex
	last      map[key]time.Time
	isOwnerFn func(userID string) bool
}

type key struct {
	userID  string
	command string
}

// New creates a cooldown manager. isOwner lets the bot owner bypass cooldowns.
func New(isOwner func(userID string) bool) *Manager {
	return &Manager{
		last:      make(map[key]time.Time),
		isOwnerFn: isOwner,
	}
}

// Remaining returns how much cooldown is left for (userID, command), or 0 if ready.
func (m *Manager) Remaining(userID, command string, d time.Duration) time.Duration {
	if m.isOwnerFn != nil && m.isOwnerFn(userID) {
		return 0
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	k := key{userID, command}
	last, ok := m.last[k]
	if !ok {
		return 0
	}

	elapsed := time.Since(last)
	// Drop entries older than an hour to keep the map small.
	if elapsed > time.Hour {
		delete(m.last, k)
		return 0
	}
	if elapsed < d {
		return d - elapsed
	}
	return 0
}

// Set records the current time as the last use of (userID, command).
func (m *Manager) Set(userID, command string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.last[key{userID, command}] = time.Now()
}
