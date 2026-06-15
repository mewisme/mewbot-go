# Discord Bot (Go)

A Discord bot built with Go using the [discordgo](https://github.com/bwmarrin/discordgo) library. It is a straightforward (non-plugin) port of the Rust/Serenity bot, with a clean package layout under `internal/`. The bot supports slash commands and prefix commands, per-command cooldowns, a permission hierarchy (bot owner, server admin, member), a help system, and a runtime per-command/per-subcommand permission config.

**Author:** Mew

## Features

- **Simple package structure** - Core packages under `internal/`; each command is a package under `internal/commands/`.
- **Unified command model** - The same commands work as slash and prefix.
- **Automatic slash registration** - Slash commands are bulk-overwritten with Discord on ready (clearing stale globals).
- **Per-user, per-command cooldown** - Bot owner bypasses via `ADMIN_USER_ID`.
- **Permission hierarchy** - **Bot owner** (all servers) > **server admin** (per server) > member.
- **Runtime permission config** - All command access is controlled via `data/config.json`. Grant user IDs and/or role tokens (`member`, `admin`) per command path with `/addperm`, `/rmperm`, `/checkperm`.
- **Help** - `/help` lists commands; `/help <command>` shows command + subcommands; `/help <command> <subcommand>` shows subcommand help.
- **Wallet command** - check, credit, debit, init, reset, unit (JSON-backed); user mentions are suppressed (no ping).

## Prerequisites

- Go 1.26+
- Discord bot token
- For permission checks and wallet init-all: **Server Members Intent** enabled in the Discord Developer Portal
- For prefix commands: **Message Content Intent** enabled

## Install

Install the latest released binary directly with Go (requires Go 1.26+ and `$GOPATH/bin` on your `PATH`):

```bash
go install github.com/mewisme/mewbot-go/cmd/mewbot@latest
```

This installs a `mewbot` binary into `$GOPATH/bin` (or `$GOBIN`). Run it from a directory containing your `.env` file:

```bash
mewbot
```

To install a specific version, replace `@latest` with a tag, e.g. `@v1.0.0`.

## Setup

1. Create a `.env` file in the project root (see `.env.example`):

```env
DISCORD_TOKEN=your_bot_token_here
COMMAND_PREFIX=m/
ADMIN_USER_ID=
```

2. Get your Discord bot token from the [Discord Developer Portal](https://discord.com/developers/applications) (Bot section).

3. Build and run:

```bash
go build ./...
go run ./cmd/mewbot
```

## Configuration

| Variable         | Description                                                                                          |
|------------------|------------------------------------------------------------------------------------------------------|
| `DISCORD_TOKEN`  | Discord bot token (required)                                                                         |
| `COMMAND_PREFIX` | Prefix for text commands (default: `m/`)                                                             |
| `ADMIN_USER_ID`  | User ID of the **bot owner** (optional, recommended). Bypasses cooldowns and can use all commands.   |

### Permission levels

- **Bot owner** - User in `ADMIN_USER_ID`. Highest level; all commands in any server.
- **Server admin** - Guild owner or users with **Administrator** in that server.
- **Member** - Everyone else.

### Permission config

All commands are gated by the perm tree in `data/config.json`. The **bot owner** (`ADMIN_USER_ID`) always passes. Everyone else needs a matching grant on the command path.

**Permission levels** (used when resolving role tokens):

- **Bot owner** - User in `ADMIN_USER_ID`. All commands in any server.
- **Server admin** - Guild owner or users with **Administrator** in that server.
- **Member** - Everyone else in the guild.

**Grant types** (stored in each node's `users` or `all` bucket):

- User ID - that specific user passes.
- `member` - any guild member passes.
- `admin` - guild admin or owner passes.

Entries in a bucket use **OR** logic: if a bucket has user IDs and `admin`, a user passes when they match **any** entry (listed ID, or `member`, or `admin` as applicable).

Paths use dotted notation up to 4 levels deep, e.g. `wallet`, `wallet.credit`.

Each node has two buckets:

- `users` - grants at that **exact** path only.
- `all` - grants at that path **and** all subcommands (inherits down). Target it by appending `.all`, e.g. `wallet.all`.

A user is allowed at a path if any entry matches in that node's `users` bucket, or in the `all` bucket of that node or any ancestor.

**Managing grants** (prefix examples; slash supports `roles` + user options):

- `addperm wallet.all member` - any member can use wallet and all subcommands.
- `addperm wallet.credit admin @user` - admins or the mentioned user can run `wallet credit`.
- `rmperm wallet.credit admin` - remove the admin grant.
- `checkperm wallet.credit` - list grants; `checkperm wallet.credit @user` - test a user.

Until grants are configured, only the bot owner can run commands. Use the owner account to seed access, e.g. `addperm help.all member`.

Example `data/config.json`:

```json
{
  "perm": {
    "wallet": {
      "all": ["member"],
      "users": [],
      "children": {
        "credit": { "all": ["admin", "234567890123456789"], "users": [], "children": {} }
      }
    }
  }
}
```

Here any member can use wallet; only admins or user `234...` can run `wallet credit` and its subcommands.

## Bot commands

**Slash** - Type `/` (e.g. `/help`, `/wallet`).
**Prefix** - Use prefix + command (e.g. `m/help`, `m/wallet`). Aliases (e.g. `m/w`, `m/bal` for wallet) work when defined.

### Help

- **`/help`** or **`m/help`** (aliases: `m/h`, `m/commands`) - List all commands with short descriptions.
- **`/help <command>`** - Help for one command: description, slash/prefix, aliases, cooldown, version, and subcommands.
- **`/help <command> <subcommand>`** - Help for a specific subcommand (e.g. `/help wallet check`).

### Wallet

- **check** (default) - View balance. Self or (with permission) others. **Bot owner** or **server admin** can view others.
- **credit** - Add balance. **Bot owner** or **server admin** only.
- **debit** - Remove balance. Same permission as credit.
- **init** - Initialize wallet(s), default balance 0. Optional user; no user = init all non-bot members in the server.
- **reset** - Set wallet(s) to a given balance. Same permission and user rules as init.
- **unit** - Set display unit for balances (e.g. "xu"). Same permission.

Data is stored in `data/wallet.json`.

### Permissions

- **addperm** / **rmperm** / **checkperm** - Manage the perm config (requires a grant on these commands; bot owner by default).

## Source code structure

```
discord-bot-go/
├── go.mod
├── go.sum
├── .env.example
├── README.md
├── cmd/
│   └── mewbot/
│       └── main.go              # entry point: load config, register commands, start session
└── internal/
    ├── config/config.go         # env loading (DISCORD_TOKEN, COMMAND_PREFIX, ADMIN_USER_ID)
    ├── appconfig/appconfig.go   # mutable data/config.json perm tree (Manager)
    ├── logger/logger.go         # colored Info/Warn/Error/Done/Debug logging
    ├── permissions/permissions.go  # Level, Resolve, Has
    ├── cooldown/cooldown.go     # per-(user,command) cooldown map; owner bypass
    ├── store/store.go           # generic data/ file load+save (path traversal guard)
    ├── command/command.go       # Command interface, SubCommandInfo, Info, Base
    ├── registry/registry.go     # name + prefix/alias maps; List() for help
    ├── bot/
    │   ├── bot.go               # session, intents, handler wiring
    │   ├── ready.go             # bulk-register slash commands on ready
    │   ├── message.go           # prefix dispatch: perm gate -> cooldown -> run
    │   ├── interaction.go       # slash dispatch: perm gate -> cooldown -> run
    │   └── perm.go              # shared permission resolution + allowlist gate
    └── commands/
        ├── help/help.go         # list all / command+subcommands / single subcommand
        ├── perm/                # addperm / rmperm / checkperm
        │   ├── perm.go          # shared path validation + helpers
        │   └── commands.go      # the three command implementations
        └── wallet/
            ├── wallet.go        # definition, options, helpers
            ├── slash.go         # slash handlers
            ├── prefix.go        # prefix handlers
            ├── respond.go       # response helpers
            └── store.go         # WalletData on top of internal/store (wallet.json)
```

## Adding new commands

1. Create a package under `internal/commands/<name>/`.
2. Implement the `command.Command` interface (embed `command.Base` for defaults). At minimum: `Name`, `Description`, `Slash`, `RunSlash`, `RunPrefix`. Override `Prefix`, `Aliases`, `Cooldown`, `Version`, `Subcommands` as needed.
3. Register it in `main.go` via `reg.Register(...)`.
4. Grant access via `addperm` (e.g. `addperm mycommand.all member`).
5. Rebuild and run.

## How it works

- **Registration** - `main` builds the `registry`, registers each command, and starts the session. On ready, all slash commands are bulk-overwritten with Discord.
- **Dispatch** - The message/interaction handlers resolve the command and its invocation path, check the perm config gate, check the cooldown, run, and set the cooldown on success.
- **Cooldowns** - Per user, per command; the bot owner is skipped. Entries older than an hour are dropped.

## Dependencies

- `github.com/bwmarrin/discordgo` - Discord API
- `github.com/joho/godotenv` - `.env` loading
- stdlib `encoding/json` - JSON storage

## License

MIT License. See [LICENSE](LICENSE).

Copyright (c) 2026 Mew
