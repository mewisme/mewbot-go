package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/command"
	"github.com/mewisme/mewbot-go/internal/logger"
	"github.com/mewisme/mewbot-go/internal/permissions"
)

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	cmd, ok := b.Registry.GetSlash(data.Name)
	if !ok {
		logger.Error("unknown slash command: %s", data.Name)
		return
	}

	userID := interactionUserID(i)
	path := resolveSlashPath(cmd, data.Options)

	if i.GuildID == "" {
		b.respondError(i, "This command can only be used in a server.")
		return
	}
	guildOwnerID, hasAdmin, ok := memberHasAdministrator(s, i.GuildID, userID, i.Member)
	if !ok {
		b.respondError(i, "Could not load server information.")
		return
	}
	userLevel := permissions.Resolve(guildOwnerID, userID, hasAdmin)
	if !b.gate(userLevel, path, userID) {
		b.respondError(i, "You don't have permission to use this command.")
		return
	}

	if remaining := b.Cooldown.Remaining(userID, cmd.Name(), cmd.Cooldown()); remaining > 0 {
		b.respondError(i, fmt.Sprintf("You are on cooldown for %s", formatDuration(remaining)))
		return
	}

	if err := cmd.RunSlash(s, i); err != nil {
		logger.Error("slash command %s failed: %v", cmd.Name(), err)
		b.respondError(i, fmt.Sprintf("Error: %v", err))
		return
	}
	b.Cooldown.Set(userID, cmd.Name())
}

// interactionUserID returns the invoking user's ID for guild or DM contexts.
func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

// resolveSlashPath walks slash options to build the invocation path, descending
// through subcommand groups and subcommands.
func resolveSlashPath(cmd command.Command, opts []*discordgo.ApplicationCommandInteractionDataOption) []string {
	path := []string{strings.ToLower(cmd.Name())}
	for len(opts) > 0 {
		opt := opts[0]
		if opt.Type != discordgo.ApplicationCommandOptionSubCommand &&
			opt.Type != discordgo.ApplicationCommandOptionSubCommandGroup {
			break
		}
		path = append(path, strings.ToLower(opt.Name))
		opts = opt.Options
	}
	return path
}

func (b *Bot) respondError(i *discordgo.InteractionCreate, text string) {
	err := b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{{
				Title:       "Error",
				Description: text,
				Color:       0xff0000,
			}},
		},
	})
	if err != nil {
		logger.Error("failed to send error response: %v", err)
	}
}

// formatDuration renders a cooldown remaining time like the Rust bot.
func formatDuration(d time.Duration) string {
	secs := int(d.Seconds())
	switch {
	case secs < 60:
		return pluralize(secs, "second")
	case secs < 3600:
		return pluralize(secs/60, "minute")
	default:
		return pluralize(secs/3600, "hour")
	}
}

func pluralize(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
