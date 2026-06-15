package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/command"
	"github.com/mewisme/mewbot-go/internal/logger"
	"github.com/mewisme/mewbot-go/internal/permissions"
)

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot {
		return
	}

	prefix := b.Config.Prefix
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	content := strings.TrimPrefix(m.Content, prefix)
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return
	}

	name := parts[0]
	args := parts[1:]

	cmd, ok := b.Registry.GetPrefix(name)
	if !ok {
		return
	}

	// Resolve the invocation path (command + subcommand segments) for the
	// allowlist lookup.
	path := resolvePrefixPath(cmd, args)

	if m.GuildID == "" {
		b.replyError(m, "This command can only be used in a server.")
		return
	}
	guildOwnerID, hasAdmin, ok := memberHasAdministrator(s, m.GuildID, m.Author.ID, m.Member)
	if !ok {
		b.replyError(m, "Could not load server information.")
		return
	}
	userLevel := permissions.Resolve(guildOwnerID, m.Author.ID, hasAdmin)
	if !b.gate(userLevel, path, m.Author.ID) {
		b.replyError(m, "You don't have permission to use this command.")
		return
	}

	if remaining := b.Cooldown.Remaining(m.Author.ID, cmd.Name(), cmd.Cooldown()); remaining > 0 {
		b.replyError(m, fmt.Sprintf("You are on cooldown for %s", formatDuration(remaining)))
		return
	}

	if err := cmd.RunPrefix(s, m, args); err != nil {
		logger.Error("prefix command %s failed: %v", cmd.Name(), err)
		b.replyError(m, fmt.Sprintf("Error: %v", err))
		return
	}
	b.Cooldown.Set(m.Author.ID, cmd.Name())
}

// resolvePrefixPath walks args against the command's subcommand tree to build
// the invocation path used for allowlist lookups.
func resolvePrefixPath(cmd command.Command, args []string) []string {
	path := []string{strings.ToLower(cmd.Name())}
	subs := cmd.Subcommands()
	for _, arg := range args {
		canonical, ok := command.ResolveSubcommand(subs, arg)
		if !ok {
			break
		}
		path = append(path, strings.ToLower(canonical))
		// Descend into the matched subcommand's children.
		var next []command.SubCommandInfo
		for _, s := range subs {
			if s.Name == canonical {
				next = s.Children
				break
			}
		}
		subs = next
		if len(subs) == 0 {
			break
		}
	}
	return path
}

func (b *Bot) replyError(m *discordgo.MessageCreate, text string) {
	_, err := b.Session.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		Title:       "Error",
		Description: text,
		Color:       0xff0000,
	})
	if err != nil {
		logger.Error("failed to send error message: %v", err)
	}
}
