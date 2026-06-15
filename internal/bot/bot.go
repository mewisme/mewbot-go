package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/appconfig"
	"github.com/mewisme/mewbot-go/internal/config"
	"github.com/mewisme/mewbot-go/internal/cooldown"
	"github.com/mewisme/mewbot-go/internal/registry"
)

// Bot bundles the session with the shared services handlers need.
type Bot struct {
	Session  *discordgo.Session
	Config   *config.Config
	Registry *registry.Registry
	Cooldown *cooldown.Manager
	Perms    *appconfig.Manager
}

// New builds a bot, wires the gateway intents, and registers event handlers.
func New(cfg *config.Config, reg *registry.Registry, perms *appconfig.Manager) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, err
	}

	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildMembers

	b := &Bot{
		Session:  session,
		Config:   cfg,
		Registry: reg,
		Cooldown: cooldown.New(cfg.IsAdmin),
		Perms:    perms,
	}

	session.AddHandler(b.onReady)
	session.AddHandler(b.onMessageCreate)
	session.AddHandler(b.onInteractionCreate)

	return b, nil
}

// Open connects to the gateway.
func (b *Bot) Open() error {
	return b.Session.Open()
}

// Close disconnects from the gateway.
func (b *Bot) Close() error {
	return b.Session.Close()
}
