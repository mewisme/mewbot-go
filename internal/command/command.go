package command

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// SubCommandInfo describes a subcommand for help and perm-path validation.
type SubCommandInfo struct {
	Name        string
	Description string
	Aliases     []string
	// Children allows nested subcommand groups (up to the perm-tree depth).
	Children []SubCommandInfo
}

// Command is a single bot command, usable as both slash and prefix.
type Command interface {
	// Name is the canonical command name (lowercase).
	Name() string
	// Description is a short, human-readable summary.
	Description() string

	// Slash returns the discordgo application command definition.
	Slash() *discordgo.ApplicationCommand
	// RunSlash handles a slash interaction for this command.
	RunSlash(s *discordgo.Session, i *discordgo.InteractionCreate) error

	// Prefix is the text-command trigger; empty means it equals Name.
	Prefix() string
	// RunPrefix handles a prefix invocation with the remaining args.
	RunPrefix(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error

	// Aliases are alternative prefix triggers.
	Aliases() []string
	// Cooldown is the per-user delay between uses.
	Cooldown() time.Duration
	// Version is the command's semantic version.
	Version() string
	// Subcommands lists declared subcommands (for help + perm paths).
	Subcommands() []SubCommandInfo
}

// Info is a flattened, read-only view of a command for help rendering.
type Info struct {
	Name        string
	Description string
	Prefix      string
	Aliases     []string
	CooldownSec int
	Version     string
	Subcommands []SubCommandInfo
}

// Base provides default implementations for optional Command methods.
// Embed it in a command struct to avoid boilerplate.
type Base struct{}

func (Base) Prefix() string                { return "" }
func (Base) Aliases() []string             { return nil }
func (Base) Cooldown() time.Duration       { return 3 * time.Second }
func (Base) Version() string               { return "1.0.0" }
func (Base) Subcommands() []SubCommandInfo { return nil }

// ResolveSubcommand returns the canonical subcommand name for a name or alias.
func ResolveSubcommand(subs []SubCommandInfo, nameOrAlias string) (string, bool) {
	for _, s := range subs {
		if equalFold(s.Name, nameOrAlias) {
			return s.Name, true
		}
		for _, a := range s.Aliases {
			if equalFold(a, nameOrAlias) {
				return s.Name, true
			}
		}
	}
	return "", false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
