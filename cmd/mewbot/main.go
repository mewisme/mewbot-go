package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mewisme/mewbot-go/internal/appconfig"
	"github.com/mewisme/mewbot-go/internal/bot"
	"github.com/mewisme/mewbot-go/internal/commands/help"
	"github.com/mewisme/mewbot-go/internal/commands/perm"
	"github.com/mewisme/mewbot-go/internal/commands/wallet"
	"github.com/mewisme/mewbot-go/internal/config"
	"github.com/mewisme/mewbot-go/internal/logger"
	"github.com/mewisme/mewbot-go/internal/permissions"
	"github.com/mewisme/mewbot-go/internal/registry"
)

func main() {
	logger.Info("mewbot %s (commit %s, built %s)", buildVersion(), buildCommit(), buildDate())

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}
	logger.Done("Configuration loaded successfully")

	permissions.SetOwnerID(cfg.AdminUserID)

	perms, err := appconfig.Load()
	if err != nil {
		logger.Error("Failed to load perm config: %v", err)
		os.Exit(1)
	}
	logger.Done("Permission config loaded")

	reg := registry.New()
	reg.Register(wallet.New())
	reg.Register(perm.NewAdd(reg, perms))
	reg.Register(perm.NewRemove(reg, perms))
	reg.Register(perm.NewCheck(reg, perms))
	reg.Register(help.New(cfg, reg))
	for _, c := range reg.All() {
		logger.Done("Loaded command %s v%s", c.Name(), c.Version())
	}

	b, err := bot.New(cfg, reg, perms)
	if err != nil {
		logger.Error("Failed to create bot: %v", err)
		os.Exit(1)
	}

	if err := b.Open(); err != nil {
		logger.Error("Failed to open gateway connection: %v", err)
		os.Exit(1)
	}
	defer b.Close()

	logger.Info("Bot is running.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
	logger.Warn("Shutting down.")
}
