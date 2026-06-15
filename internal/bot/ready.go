package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/logger"
)

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	cmds := b.Registry.All()
	defs := make([]*discordgo.ApplicationCommand, 0, len(cmds))
	for _, c := range cmds {
		if def := c.Slash(); def != nil {
			defs = append(defs, def)
		}
	}

	if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", defs); err != nil {
		logger.Error("Failed to register slash commands: %v", err)
	}

	logger.Done("%s is connected", r.User.Username)
}
