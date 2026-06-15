package permissions

import "sync"

// Level is a permission tier. Higher values outrank lower ones.
type Level int

const (
	// LevelMember is the lowest tier: any guild member.
	LevelMember Level = iota
	// LevelAdmin is a guild admin or guild owner.
	LevelAdmin
	// LevelOwner is the configured bot owner (highest).
	LevelOwner
)

var (
	ownerMu sync.RWMutex
	ownerID string
)

// SetOwnerID stores the bot owner's user ID for later resolution.
func SetOwnerID(id string) {
	ownerMu.Lock()
	defer ownerMu.Unlock()
	ownerID = id
}

func botOwnerID() string {
	ownerMu.RLock()
	defer ownerMu.RUnlock()
	return ownerID
}

// Resolve determines a user's level within a guild.
func Resolve(guildOwnerID, userID string, hasAdministrator bool) Level {
	if owner := botOwnerID(); owner != "" && owner == userID {
		return LevelOwner
	}
	if guildOwnerID == userID || hasAdministrator {
		return LevelAdmin
	}
	return LevelMember
}

// Has reports whether userLevel satisfies the required level.
func Has(userLevel, required Level) bool {
	return userLevel >= required
}

// Message returns a human label for a required level.
func Message(required Level) string {
	switch required {
	case LevelOwner:
		return "bot owner"
	case LevelAdmin:
		return "server admin"
	default:
		return "member"
	}
}
