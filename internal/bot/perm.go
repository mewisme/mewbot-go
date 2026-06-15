package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/permissions"
)

// memberHasAdministrator reports whether the guild member has the
// Administrator permission, accounting for the guild owner.
func memberHasAdministrator(s *discordgo.Session, guildID, userID string, member *discordgo.Member) (guildOwnerID string, hasAdmin bool, ok bool) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			return "", false, false
		}
	}
	guildOwnerID = guild.OwnerID

	if member == nil {
		return guildOwnerID, false, true
	}

	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			continue
		}
		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return guildOwnerID, true, true
		}
	}
	return guildOwnerID, false, true
}

// gate checks whether a user may run a command at the given invocation path
// using data/config.json perm rules. Bot owner always passes; otherwise access
// requires a matching grant (user ID, "member", or "admin") in the perm tree.
func (b *Bot) gate(userLevel permissions.Level, path []string, userID string) bool {
	if userLevel == permissions.LevelOwner {
		return true
	}
	return b.Perms.Allowed(path, userID, userLevel)
}
