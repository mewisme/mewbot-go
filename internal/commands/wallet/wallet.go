package wallet

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/command"
	"github.com/mewisme/mewbot-go/internal/permissions"
)

// Wallet is the wallet command (check / credit / debit / init / reset / unit).
type Wallet struct {
	command.Base
}

// New creates the wallet command.
func New() *Wallet { return &Wallet{} }

func (w *Wallet) Name() string { return "wallet" }
func (w *Wallet) Description() string {
	return "Check or manage wallet balance (check / credit / debit / init / reset / unit)"
}
func (w *Wallet) Prefix() string          { return "wallet" }
func (w *Wallet) Aliases() []string       { return []string{"w", "bal", "balance"} }
func (w *Wallet) Cooldown() time.Duration { return 2 * time.Second }

func (w *Wallet) Subcommands() []command.SubCommandInfo {
	return []command.SubCommandInfo{
		{Name: "check", Description: "Check wallet balance (self or mentioned users)", Aliases: []string{"bal", "balance"}},
		{Name: "credit", Description: "Add money (bot owner / server admin only)", Aliases: []string{"add"}},
		{Name: "debit", Description: "Remove money (bot owner / server admin only)", Aliases: []string{"remove", "sub"}},
		{Name: "init", Description: "Initialize wallet(s) (bot owner / server admin only)"},
		{Name: "reset", Description: "Reset wallet(s) to balance (bot owner / server admin only)"},
		{Name: "unit", Description: "Set display unit for balance (e.g. xu) (bot owner / server admin only)"},
	}
}

func intOpt(min int) *float64 {
	v := float64(min)
	return &v
}

func (w *Wallet) Slash() *discordgo.ApplicationCommand {
	userOpt := func(desc string) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: desc,
			Required:    false,
		}
	}
	return &discordgo.ApplicationCommand{
		Name:        "wallet",
		Description: "Check or manage wallet balance",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "check",
				Description: "Check wallet balance (self or mentioned user)",
				Options:     []*discordgo.ApplicationCommandOption{userOpt("User to check (optional, default: self)")},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "credit",
				Description: "Add money (bot owner / server admin only)",
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount to add", Required: true, MinValue: intOpt(1)},
					userOpt("User to credit (optional, default: self)"),
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "debit",
				Description: "Remove money (bot owner / server admin only)",
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount to remove", Required: true, MinValue: intOpt(1)},
					userOpt("User to debit (optional, default: self)"),
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "init",
				Description: "Initialize wallet(s) (bot owner / server admin only)",
				Options: []*discordgo.ApplicationCommandOption{
					userOpt("User to init (optional; omit to init all server members)"),
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Initial balance (optional, default: 0)", Required: false, MinValue: intOpt(0)},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "reset",
				Description: "Reset wallet(s) to balance (bot owner / server admin only)",
				Options: []*discordgo.ApplicationCommandOption{
					userOpt("User to reset (optional; omit to reset all server members)"),
					{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Balance after reset (optional, default: 0)", Required: false, MinValue: intOpt(0)},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "unit",
				Description: "Set display unit for balance (e.g. xu) (bot owner / server admin only)",
				Options: []*discordgo.ApplicationCommandOption{
					{Type: discordgo.ApplicationCommandOptionString, Name: "unit", Description: "Unit name (e.g. xu, coins). Omit or empty to clear.", Required: false},
				},
			},
		},
	}
}

const (
	colorGreen = 0x00ff00
	colorAmber = 0xffaa00
)

func userMention(id string) string { return "<@" + id + ">" }

func noPing() *discordgo.MessageAllowedMentions {
	return &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}
}

// callerAdmin resolves whether the invoking user is at least Admin in the guild.
func callerAdmin(s *discordgo.Session, guildID, userID string, member *discordgo.Member) (bool, bool) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			return false, false
		}
	}
	level := permissions.Resolve(guild.OwnerID, userID, memberIsAdmin(s, guildID, member))
	return permissions.Has(level, permissions.LevelAdmin), true
}

func memberIsAdmin(s *discordgo.Session, guildID string, member *discordgo.Member) bool {
	if member == nil {
		return false
	}
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			continue
		}
		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return true
		}
	}
	return false
}

// guildHumanMembers returns non-bot member IDs from state, falling back to the API.
func guildHumanMembers(s *discordgo.Session, guildID string) []string {
	var ids []string
	if guild, err := s.State.Guild(guildID); err == nil {
		for _, m := range guild.Members {
			if m.User != nil && !m.User.Bot {
				ids = append(ids, m.User.ID)
			}
		}
	}
	if len(ids) > 0 {
		return ids
	}
	members, err := s.GuildMembers(guildID, "", 1000)
	if err != nil {
		return nil
	}
	for _, m := range members {
		if m.User != nil && !m.User.Bot {
			ids = append(ids, m.User.ID)
		}
	}
	return ids
}

func parsePositive(s string) (int64, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
