package perm

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/appconfig"
	"github.com/mewisme/mewbot-go/internal/command"
	"github.com/mewisme/mewbot-go/internal/permissions"
	"github.com/mewisme/mewbot-go/internal/registry"
)

// validatePath resolves a dotted command path against the registry. The first
// segment must be a registered command; remaining segments must resolve to
// declared (canonical or alias) subcommands. A trailing "all" segment selects
// the inheriting "all" bucket of the node before it (e.g. wallet.all or
// wallet.credit.all). It returns the canonical path and the target bucket.
func validatePath(reg *registry.Registry, dotted string) ([]string, appconfig.Bucket, error) {
	segs := appconfig.SplitPath(dotted)

	bucket := appconfig.BucketUsers
	if len(segs) > 1 && segs[len(segs)-1] == appconfig.AllSegment {
		bucket = appconfig.BucketAll
		segs = segs[:len(segs)-1]
	}

	if err := appconfig.ValidatePath(segs); err != nil {
		return nil, bucket, err
	}

	cmd, ok := reg.GetSlash(segs[0])
	if !ok {
		return nil, bucket, fmt.Errorf("unknown command `%s`", segs[0])
	}

	path := []string{strings.ToLower(cmd.Name())}
	subs := cmd.Subcommands()
	for _, seg := range segs[1:] {
		canonical, ok := command.ResolveSubcommand(subs, seg)
		if !ok {
			return nil, bucket, fmt.Errorf("`%s` is not a subcommand of `%s`", seg, strings.Join(path, "."))
		}
		path = append(path, strings.ToLower(canonical))
		var next []command.SubCommandInfo
		for _, sc := range subs {
			if sc.Name == canonical {
				next = sc.Children
				break
			}
		}
		subs = next
	}
	return path, bucket, nil
}

// pathLabel renders a path + bucket back into the dotted form for display.
func pathLabel(path []string, bucket appconfig.Bucket) string {
	dotted := strings.Join(path, ".")
	if bucket == appconfig.BucketAll {
		return dotted + "." + appconfig.AllSegment
	}
	return dotted
}

// mentionIDs returns the non-bot mentioned user IDs from a message.
func mentionIDs(m *discordgo.MessageCreate) []string {
	var ids []string
	for _, u := range m.Mentions {
		if u != nil && !u.Bot {
			ids = append(ids, u.ID)
		}
	}
	return ids
}

// slashUserIDs collects user IDs from the slash "user1".."userN" options.
func slashUserIDs(opts []*discordgo.ApplicationCommandInteractionDataOption) []string {
	var ids []string
	for _, o := range opts {
		if o.Type == discordgo.ApplicationCommandOptionUser {
			if u := o.UserValue(nil); u != nil && !u.Bot {
				ids = append(ids, u.ID)
			}
		}
	}
	return ids
}

// prefixGrants collects role tokens (member, admin) from args and user IDs from mentions.
func prefixGrants(rest []string, m *discordgo.MessageCreate) []string {
	seen := map[string]struct{}{}
	var grants []string
	add := func(g string) {
		if _, ok := seen[g]; ok {
			return
		}
		seen[g] = struct{}{}
		grants = append(grants, g)
	}
	for _, a := range rest {
		if strings.HasPrefix(a, "<@") {
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(a))
		if lower == appconfig.RoleMember || lower == appconfig.RoleAdmin {
			add(lower)
		}
	}
	for _, id := range mentionIDs(m) {
		add(id)
	}
	return grants
}

// slashGrants collects role tokens from the roles option and user IDs from user options.
func slashGrants(opts []*discordgo.ApplicationCommandInteractionDataOption) []string {
	seen := map[string]struct{}{}
	var grants []string
	add := func(g string) {
		if _, ok := seen[g]; ok {
			return
		}
		seen[g] = struct{}{}
		grants = append(grants, g)
	}
	for _, part := range strings.Split(slashRolesArg(opts), ",") {
		lower := strings.ToLower(strings.TrimSpace(part))
		if lower == appconfig.RoleMember || lower == appconfig.RoleAdmin {
			add(lower)
		}
	}
	for _, id := range slashUserIDs(opts) {
		add(id)
	}
	return grants
}

func slashRolesArg(opts []*discordgo.ApplicationCommandInteractionDataOption) string {
	for _, o := range opts {
		if o.Name == "roles" {
			return o.StringValue()
		}
	}
	return ""
}

// userOptions builds slash options: command path, optional roles, optional users.
func userOptions(n int) []*discordgo.ApplicationCommandOption {
	opts := make([]*discordgo.ApplicationCommandOption, 0, n+2)
	opts = append(opts, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "command",
		Description: "Command path, e.g. wallet, wallet.credit, wallet.all",
		Required:    true,
	})
	opts = append(opts, &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "roles",
		Description: "Role grants: member, admin (comma-separated)",
		Required:    false,
	})
	for idx := 1; idx <= n; idx++ {
		opts = append(opts, &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        fmt.Sprintf("user%d", idx),
			Description: fmt.Sprintf("User #%d", idx),
			Required:    false,
		})
	}
	return opts
}

// slashCommandArg reads the required "command" string option.
func slashCommandArg(opts []*discordgo.ApplicationCommandInteractionDataOption) string {
	for _, o := range opts {
		if o.Name == "command" {
			return o.StringValue()
		}
	}
	return ""
}

// resolveUserLevel returns the permission level for userID in a guild context.
func resolveUserLevel(s *discordgo.Session, guildID, userID string) permissions.Level {
	if guildID == "" {
		return permissions.LevelMember
	}
	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			return permissions.LevelMember
		}
	}
	member, err := s.GuildMember(guildID, userID)
	if err != nil || member == nil {
		return permissions.Resolve(guild.OwnerID, userID, false)
	}
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			continue
		}
		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return permissions.Resolve(guild.OwnerID, userID, true)
		}
	}
	return permissions.Resolve(guild.OwnerID, userID, false)
}

func grantList(grants []string) string {
	if len(grants) == 0 {
		return "(none)"
	}
	parts := make([]string, len(grants))
	for i, g := range grants {
		switch g {
		case appconfig.RoleMember, appconfig.RoleAdmin:
			parts[i] = "**" + g + "**"
		default:
			parts[i] = "<@" + g + ">"
		}
	}
	return strings.Join(parts, ", ")
}

func noPing() *discordgo.MessageAllowedMentions {
	return &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, title, desc string, color int) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:          []*discordgo.MessageEmbed{{Title: title, Description: desc, Color: color}},
			AllowedMentions: noPing(),
		},
	})
}

func send(s *discordgo.Session, channelID, title, desc string, color int) error {
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:          []*discordgo.MessageEmbed{{Title: title, Description: desc, Color: color}},
		AllowedMentions: noPing(),
	})
	return err
}

const (
	colorGreen = 0x00ff00
	colorRed   = 0xff0000
	colorBlue  = 0x0099ff
)
