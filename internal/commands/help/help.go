package help

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/command"
	"github.com/mewisme/mewbot-go/internal/config"
	"github.com/mewisme/mewbot-go/internal/registry"
)

// Help lists commands and shows per-command / per-subcommand details.
type Help struct {
	command.Base
	cfg *config.Config
	reg *registry.Registry
}

// New creates the help command wired to the config and registry.
func New(cfg *config.Config, reg *registry.Registry) *Help {
	return &Help{cfg: cfg, reg: reg}
}

func (h *Help) Name() string            { return "help" }
func (h *Help) Description() string     { return "Show help information about commands" }
func (h *Help) Prefix() string          { return "help" }
func (h *Help) Aliases() []string       { return []string{"h", "commands"} }
func (h *Help) Cooldown() time.Duration { return 2 * time.Second }

func (h *Help) Slash() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Show help information about commands",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "command",
				Description: "The command (and optional subcommand) to get help for, e.g. wallet or wallet check",
				Required:    false,
			},
		},
	}
}

func (h *Help) RunSlash(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	var arg string
	data := i.ApplicationCommandData()
	for _, o := range data.Options {
		if o.Name == "command" {
			arg = o.StringValue()
		}
	}

	embed, ephemeral := h.render(arg)
	resp := &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}}
	if ephemeral {
		resp.Flags = discordgo.MessageFlagsEphemeral
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: resp,
	})
}

func (h *Help) RunPrefix(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	embed, _ := h.render(strings.Join(args, " "))
	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	return err
}

// render builds the help embed for the given argument string. The bool reports
// whether the response should be ephemeral (not-found cases).
func (h *Help) render(arg string) (*discordgo.MessageEmbed, bool) {
	commands := h.reg.List()
	prefix := h.cfg.Prefix

	cmdName, subName := parseTarget(arg)

	if cmdName == "" {
		var b strings.Builder
		for _, info := range commands {
			fmt.Fprintf(&b, "\u2022 **%s** - %s\n", info.Name, info.Description)
		}
		return &discordgo.MessageEmbed{
			Title:       "Available Commands",
			Description: b.String(),
			Color:       0x0099ff,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Use /help <command> or %shelp <command> for detailed info", prefix),
			},
		}, false
	}

	var info *command.Info
	for idx := range commands {
		if strings.EqualFold(commands[idx].Name, cmdName) {
			info = &commands[idx]
			break
		}
	}

	if info == nil {
		return &discordgo.MessageEmbed{
			Title:       "Command Not Found",
			Description: fmt.Sprintf("Command `%s` not found.", cmdName),
			Color:       0xff0000,
		}, true
	}

	if subName == "" {
		return buildCommandEmbed(prefix, info), false
	}

	var sub *command.SubCommandInfo
	for idx := range info.Subcommands {
		s := &info.Subcommands[idx]
		if strings.EqualFold(s.Name, subName) {
			sub = s
			break
		}
		for _, a := range s.Aliases {
			if strings.EqualFold(a, subName) {
				sub = s
				break
			}
		}
		if sub != nil {
			break
		}
	}
	if sub == nil {
		return &discordgo.MessageEmbed{
			Title:       "Subcommand Not Found",
			Description: fmt.Sprintf("Subcommand `%s` not found for command `%s`.", subName, info.Name),
			Color:       0xff0000,
		}, true
	}
	return buildSubcommandEmbed(prefix, info.Name, sub), false
}

func parseTarget(arg string) (string, string) {
	s := strings.TrimSpace(arg)
	if s == "" {
		return "", ""
	}
	parts := strings.Fields(s)
	cmd := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	return cmd, sub
}

func buildCommandEmbed(prefix string, info *command.Info) *discordgo.MessageEmbed {
	slashUsage := fmt.Sprintf("`/%s`", info.Name)
	prefixUsage := "\u2014"
	if info.Prefix != "" {
		prefixUsage = fmt.Sprintf("`%s%s`", prefix, info.Prefix)
	}
	aliasesStr := "\u2014"
	if len(info.Aliases) > 0 {
		parts := make([]string, len(info.Aliases))
		for idx, a := range info.Aliases {
			parts[idx] = fmt.Sprintf("`%s%s`", prefix, a)
		}
		aliasesStr = strings.Join(parts, ", ")
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Help: %s", info.Name),
		Description: info.Description,
		Color:       0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Slash", Value: slashUsage, Inline: true},
			{Name: "Prefix", Value: prefixUsage, Inline: true},
			{Name: "Aliases", Value: aliasesStr, Inline: false},
			{Name: "Cooldown", Value: fmt.Sprintf("%d seconds", info.CooldownSec), Inline: true},
			{Name: "Version", Value: info.Version, Inline: true},
		},
	}

	if len(info.Subcommands) > 0 {
		var b strings.Builder
		for _, sc := range info.Subcommands {
			aliasStr := ""
			if len(sc.Aliases) > 0 {
				aliasStr = fmt.Sprintf(" (%s)", strings.Join(sc.Aliases, ", "))
			}
			fmt.Fprintf(&b, "\u2022 **%s**%s \u2014 %s\n", sc.Name, aliasStr, sc.Description)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Subcommands",
			Value: b.String(),
		})
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Use /help %s <subcommand> or %shelp %s <subcommand> for details", info.Name, prefix, info.Name),
		}
	}
	return embed
}

func buildSubcommandEmbed(prefix, cmdName string, sub *command.SubCommandInfo) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Help: %s %s", cmdName, sub.Name),
		Description: sub.Description,
		Color:       0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Slash", Value: fmt.Sprintf("`/%s %s`", cmdName, sub.Name), Inline: true},
			{Name: "Prefix", Value: fmt.Sprintf("`%s%s %s`", prefix, cmdName, sub.Name), Inline: true},
		},
	}
	if len(sub.Aliases) > 0 {
		parts := make([]string, len(sub.Aliases))
		for idx, a := range sub.Aliases {
			parts[idx] = fmt.Sprintf("`%s%s %s`", prefix, cmdName, a)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Aliases",
			Value: strings.Join(parts, ", "),
		})
	}
	return embed
}
