package perm

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/appconfig"
	"github.com/mewisme/mewbot-go/internal/command"
	"github.com/mewisme/mewbot-go/internal/registry"
)

// userOptionCount is how many optional user slots the slash commands expose.
const userOptionCount = 10

// AddPerm grants users access to a command path via the perm allowlist.
type AddPerm struct {
	command.Base
	reg   *registry.Registry
	perms *appconfig.Manager
}

// NewAdd creates the addperm command.
func NewAdd(reg *registry.Registry, perms *appconfig.Manager) *AddPerm {
	return &AddPerm{reg: reg, perms: perms}
}

func (c *AddPerm) Name() string            { return "addperm" }
func (c *AddPerm) Description() string     { return "Grant users access to a command path" }
func (c *AddPerm) Aliases() []string       { return []string{"ap"} }
func (c *AddPerm) Prefix() string          { return "addperm" }
func (c *AddPerm) Cooldown() time.Duration { return 2 * time.Second }
func (c *AddPerm) Slash() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "addperm",
		Description: "Grant users access to a command path (e.g. wallet.credit)",
		Options:     userOptions(userOptionCount),
	}
}

func (c *AddPerm) RunSlash(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	opts := i.ApplicationCommandData().Options
	path, bucket, err := validatePath(c.reg, slashCommandArg(opts))
	if err != nil {
		return respond(s, i, "Invalid command", err.Error(), colorRed)
	}
	ids := slashGrants(opts)
	if len(ids) == 0 {
		return respond(s, i, "No grants", "Provide user mentions and/or roles: member, admin.", colorRed)
	}
	added, err := c.perms.Add(path, bucket, ids)
	if err != nil {
		return err
	}
	return respond(s, i, "Permission added",
		fmt.Sprintf("Path `%s`\nAdded: %s\nAlready present: %s",
			pathLabel(path, bucket), grantList(added), grantList(diff(ids, added))), colorGreen)
}

func (c *AddPerm) RunPrefix(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if len(args) == 0 {
		return send(s, m.ChannelID, "Usage", "`addperm <command[.sub][.all]> [member] [admin] @user ...`", colorRed)
	}
	path, bucket, err := validatePath(c.reg, args[0])
	if err != nil {
		return send(s, m.ChannelID, "Invalid command", err.Error(), colorRed)
	}
	ids := prefixGrants(args[1:], m)
	if len(ids) == 0 {
		return send(s, m.ChannelID, "No grants", "Provide user mentions and/or roles: member, admin.", colorRed)
	}
	added, err := c.perms.Add(path, bucket, ids)
	if err != nil {
		return err
	}
	return send(s, m.ChannelID, "Permission added",
		fmt.Sprintf("Path `%s`\nAdded: %s\nAlready present: %s",
			pathLabel(path, bucket), grantList(added), grantList(diff(ids, added))), colorGreen)
}

// RmPerm revokes users' allowlist access to a command path.
type RmPerm struct {
	command.Base
	reg   *registry.Registry
	perms *appconfig.Manager
}

// NewRemove creates the rmperm command.
func NewRemove(reg *registry.Registry, perms *appconfig.Manager) *RmPerm {
	return &RmPerm{reg: reg, perms: perms}
}

func (c *RmPerm) Name() string            { return "rmperm" }
func (c *RmPerm) Description() string     { return "Revoke users' access to a command path" }
func (c *RmPerm) Aliases() []string       { return []string{"rp"} }
func (c *RmPerm) Prefix() string          { return "rmperm" }
func (c *RmPerm) Cooldown() time.Duration { return 2 * time.Second }
func (c *RmPerm) Slash() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "rmperm",
		Description: "Revoke users' access to a command path (e.g. wallet.credit)",
		Options:     userOptions(userOptionCount),
	}
}

func (c *RmPerm) RunSlash(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	opts := i.ApplicationCommandData().Options
	path, bucket, err := validatePath(c.reg, slashCommandArg(opts))
	if err != nil {
		return respond(s, i, "Invalid command", err.Error(), colorRed)
	}
	ids := slashGrants(opts)
	if len(ids) == 0 {
		return respond(s, i, "No grants", "Provide user mentions and/or roles: member, admin.", colorRed)
	}
	removed, err := c.perms.Remove(path, bucket, ids)
	if err != nil {
		return err
	}
	return respond(s, i, "Permission removed",
		fmt.Sprintf("Path `%s`\nRemoved: %s\nNot present: %s",
			pathLabel(path, bucket), grantList(removed), grantList(diff(ids, removed))), colorGreen)
}

func (c *RmPerm) RunPrefix(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if len(args) == 0 {
		return send(s, m.ChannelID, "Usage", "`rmperm <command[.sub][.all]> [member] [admin] @user ...`", colorRed)
	}
	path, bucket, err := validatePath(c.reg, args[0])
	if err != nil {
		return send(s, m.ChannelID, "Invalid command", err.Error(), colorRed)
	}
	ids := prefixGrants(args[1:], m)
	if len(ids) == 0 {
		return send(s, m.ChannelID, "No grants", "Provide user mentions and/or roles: member, admin.", colorRed)
	}
	removed, err := c.perms.Remove(path, bucket, ids)
	if err != nil {
		return err
	}
	return send(s, m.ChannelID, "Permission removed",
		fmt.Sprintf("Path `%s`\nRemoved: %s\nNot present: %s",
			pathLabel(path, bucket), grantList(removed), grantList(diff(ids, removed))), colorGreen)
}

// CheckPerm reports the allowlist for a command path.
type CheckPerm struct {
	command.Base
	reg   *registry.Registry
	perms *appconfig.Manager
}

// NewCheck creates the checkperm command.
func NewCheck(reg *registry.Registry, perms *appconfig.Manager) *CheckPerm {
	return &CheckPerm{reg: reg, perms: perms}
}

func (c *CheckPerm) Name() string            { return "checkperm" }
func (c *CheckPerm) Description() string     { return "Show or test the allowlist for a command path" }
func (c *CheckPerm) Aliases() []string       { return []string{"cp"} }
func (c *CheckPerm) Prefix() string          { return "checkperm" }
func (c *CheckPerm) Cooldown() time.Duration { return 2 * time.Second }
func (c *CheckPerm) Slash() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "checkperm",
		Description: "Show or test the allowlist for a command path (e.g. wallet.credit)",
		Options:     userOptions(userOptionCount),
	}
}

func (c *CheckPerm) RunSlash(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	opts := i.ApplicationCommandData().Options
	path, bucket, err := validatePath(c.reg, slashCommandArg(opts))
	if err != nil {
		return respond(s, i, "Invalid command", err.Error(), colorRed)
	}
	title, desc, color := c.report(s, i.GuildID, path, bucket, slashUserIDs(opts))
	return respond(s, i, title, desc, color)
}

func (c *CheckPerm) RunPrefix(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if len(args) == 0 {
		return send(s, m.ChannelID, "Usage", "`checkperm <command[.sub][.all]> [@user ...]`", colorRed)
	}
	path, bucket, err := validatePath(c.reg, args[0])
	if err != nil {
		return send(s, m.ChannelID, "Invalid command", err.Error(), colorRed)
	}
	title, desc, color := c.report(s, m.GuildID, path, bucket, mentionIDs(m))
	return send(s, m.ChannelID, title, desc, color)
}

// report builds the checkperm output.
func (c *CheckPerm) report(s *discordgo.Session, guildID string, path []string, bucket appconfig.Bucket, checkIDs []string) (string, string, int) {
	label := pathLabel(path, bucket)
	if len(checkIDs) > 0 {
		var lines []string
		for _, id := range checkIDs {
			level := resolveUserLevel(s, guildID, id)
			status := "not allowed"
			if c.perms.BucketAllows(path, bucket, id, level) {
				status = "allowed"
			} else if c.perms.Allowed(path, id, level) {
				status = "allowed (inherited)"
			}
			lines = append(lines, fmt.Sprintf("<@%s>: %s", id, status))
		}
		return fmt.Sprintf("Permission check: %s", label), strings.Join(lines, "\n"), colorBlue
	}
	grants := c.perms.List(path, bucket)
	if len(grants) == 0 {
		return fmt.Sprintf("Permission check: %s", label), "No grants in this bucket.", colorBlue
	}
	return fmt.Sprintf("Permission check: %s", label), "Grants:\n" + grantList(grants), colorBlue
}

// diff returns the elements of all not present in subset.
func diff(all, subset []string) []string {
	in := map[string]struct{}{}
	for _, s := range subset {
		in[s] = struct{}{}
	}
	var out []string
	for _, a := range all {
		if _, ok := in[a]; !ok {
			out = append(out, a)
		}
	}
	return out
}
