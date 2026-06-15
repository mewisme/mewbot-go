package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds runtime settings loaded from the environment.
type Config struct {
	DiscordToken string
	Prefix       string
	AdminUserID  string
}

// Load reads configuration from .env (if present) and the environment.
func Load() (*Config, error) {
	_ = godotenv.Load()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN not found in environment")
	}

	prefix := os.Getenv("COMMAND_PREFIX")
	if prefix == "" {
		prefix = "m/"
	}

	adminID := os.Getenv("ADMIN_USER_ID")
	if adminID != "" {
		if _, err := strconv.ParseUint(adminID, 10, 64); err != nil {
			return nil, fmt.Errorf("ADMIN_USER_ID must be a numeric user ID: %w", err)
		}
	}

	return &Config{
		DiscordToken: token,
		Prefix:       prefix,
		AdminUserID:  adminID,
	}, nil
}

// IsAdmin reports whether the given user ID is the configured bot owner.
func (c *Config) IsAdmin(userID string) bool {
	return c.AdminUserID != "" && c.AdminUserID == userID
}
