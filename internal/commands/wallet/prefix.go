package wallet

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mewisme/mewbot-go/internal/command"
)

// humanMentions returns non-bot mentioned user IDs from a message.
func humanMentions(m *discordgo.MessageCreate) []string {
	var ids []string
	for _, u := range m.Mentions {
		if u != nil && !u.Bot {
			ids = append(ids, u.ID)
		}
	}
	return ids
}

// nonMentionArgs drops tokens that are user mentions, leaving positional args.
func nonMentionArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if strings.HasPrefix(a, "<@") && strings.HasSuffix(a, ">") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// RunPrefix handles the wallet prefix command.
func (w *Wallet) RunPrefix(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if m.GuildID == "" {
		return sendError(s, m.ChannelID, "This command can only be used in a server.")
	}

	isAdmin, ok := callerAdmin(s, m.GuildID, m.Author.ID, m.Member)
	if !ok {
		return sendError(s, m.ChannelID, "Could not load server information.")
	}

	subArg := "check"
	if len(args) > 0 {
		subArg = strings.ToLower(args[0])
		args = args[1:]
	}
	if canonical, found := command.ResolveSubcommand(w.Subcommands(), subArg); found {
		subArg = canonical
	}

	mentions := humanMentions(m)
	rest := nonMentionArgs(args)

	switch subArg {
	case "check":
		return w.prefixCheck(s, m, mentions, isAdmin)
	case "credit", "debit":
		return w.prefixCreditDebit(s, m, subArg, rest, mentions, isAdmin)
	case "init":
		return w.prefixInitReset(s, m, "init", rest, mentions, isAdmin)
	case "reset":
		return w.prefixInitReset(s, m, "reset", rest, mentions, isAdmin)
	case "unit":
		return w.prefixUnit(s, m, rest, isAdmin)
	default:
		return w.prefixCheck(s, m, mentions, isAdmin)
	}
}

func (w *Wallet) prefixCheck(s *discordgo.Session, m *discordgo.MessageCreate, mentions []string, isAdmin bool) error {
	targets := []string{m.Author.ID}
	if len(mentions) > 0 {
		if !isAdmin {
			return sendError(s, m.ChannelID, "You need **bot owner** or **server admin** to view others' wallets.")
		}
		targets = mentions
	}

	walletLock.Lock()
	defer walletLock.Unlock()
	data := loadWallet()

	var notInited []string
	for _, id := range targets {
		if !data.hasUser(id) {
			notInited = append(notInited, userMention(id))
		}
	}
	if len(notInited) > 0 {
		return sendError(s, m.ChannelID, fmt.Sprintf("The following user(s) have not been initialized: **%s**. Use **wallet init** first.", strings.Join(notInited, ", ")))
	}

	var lines []string
	for _, id := range targets {
		bal, _ := data.balanceIfExists(id)
		lines = append(lines, fmt.Sprintf("**%s**: %s", userMention(id), formatBalance(bal, data.Unit)))
	}
	return sendEmbed(s, m.ChannelID, "Wallet", strings.Join(lines, "\n"), colorGreen)
}

func (w *Wallet) prefixCreditDebit(s *discordgo.Session, m *discordgo.MessageCreate, sub string, rest, mentions []string, isAdmin bool) error {
	if !isAdmin {
		return sendError(s, m.ChannelID, "You need **bot owner** or **server admin** to add/remove balance.")
	}
	if len(rest) == 0 {
		return sendError(s, m.ChannelID, "Invalid amount (positive number required).")
	}
	amount, ok := parsePositive(rest[0])
	if !ok {
		return sendError(s, m.ChannelID, "Invalid amount (positive number required).")
	}
	targets := []string{m.Author.ID}
	if len(mentions) > 0 {
		targets = mentions
	}

	walletLock.Lock()
	defer walletLock.Unlock()
	data := loadWallet()

	var notInited []string
	for _, id := range targets {
		if !data.hasUser(id) {
			notInited = append(notInited, userMention(id))
		}
	}
	if len(notInited) > 0 {
		return sendError(s, m.ChannelID, fmt.Sprintf("The following user(s) have not been initialized: **%s**. Use **wallet init** first.", strings.Join(notInited, ", ")))
	}

	now := nowISO()
	if sub == "credit" {
		for _, id := range targets {
			data.addBalance(id, amount, now)
		}
		if err := saveWallet(data); err != nil {
			return err
		}
		names := make([]string, len(targets))
		for idx, id := range targets {
			names[idx] = userMention(id)
		}
		return sendEmbed(s, m.ChannelID, "Wallet", fmt.Sprintf("Added **%s** to: %s", formatBalance(uint64(amount), data.Unit), strings.Join(names, ", ")), colorGreen)
	}

	var okNames, failed []string
	for _, id := range targets {
		if _, err := data.subtractBalance(id, amount, now); err != nil {
			failed = append(failed, userMention(id))
		} else {
			okNames = append(okNames, userMention(id))
		}
	}
	if err := saveWallet(data); err != nil {
		return err
	}
	amtStr := formatBalance(uint64(amount), data.Unit)
	if len(failed) == 0 {
		return sendEmbed(s, m.ChannelID, "Wallet", fmt.Sprintf("Removed **%s** from: %s", amtStr, strings.Join(okNames, ", ")), colorGreen)
	}
	return sendEmbed(s, m.ChannelID, "Wallet (partial)", fmt.Sprintf("Removed from: %s. Insufficient balance: %s", strings.Join(okNames, ", "), strings.Join(failed, ", ")), colorAmber)
}

func (w *Wallet) prefixInitReset(s *discordgo.Session, m *discordgo.MessageCreate, mode string, rest, mentions []string, isAdmin bool) error {
	if !isAdmin {
		return sendError(s, m.ChannelID, fmt.Sprintf("You need **bot owner** or **server admin** to %s wallets.", mode))
	}
	var targets []string
	if len(mentions) > 0 {
		targets = mentions
	} else {
		targets = guildHumanMembers(s, m.GuildID)
	}
	if len(targets) == 0 {
		return sendError(s, m.ChannelID, fmt.Sprintf("No users to %s (could not load server members from cache or API).", mode))
	}
	var amount int64
	for _, tok := range rest {
		if n, ok := parseNonNeg(tok); ok {
			amount = n
			break
		}
	}

	walletLock.Lock()
	defer walletLock.Unlock()
	data := loadWallet()
	now := nowISO()

	if mode == "init" {
		created := 0
		for _, id := range targets {
			if data.initUserIfNew(id, amount, now) {
				created++
			}
		}
		if err := saveWallet(data); err != nil {
			return err
		}
		skipped := len(targets) - created
		balStr := formatBalance(clampNonNeg(amount), data.Unit)
		desc := fmt.Sprintf("Initialized **%d** user(s) with balance **%s**.", created, balStr)
		if skipped > 0 {
			desc = fmt.Sprintf("Initialized **%d** new user(s) with balance **%s**. Skipped **%d** already in wallet.", created, balStr, skipped)
		}
		return sendEmbed(s, m.ChannelID, "Wallet Init", desc, colorGreen)
	}

	for _, id := range targets {
		data.initUser(id, amount, now)
	}
	if err := saveWallet(data); err != nil {
		return err
	}
	balStr := formatBalance(clampNonNeg(amount), data.Unit)
	return sendEmbed(s, m.ChannelID, "Wallet Reset", fmt.Sprintf("Reset **%d** user(s) to balance **%s**.", len(targets), balStr), colorGreen)
}

func (w *Wallet) prefixUnit(s *discordgo.Session, m *discordgo.MessageCreate, rest []string, isAdmin bool) error {
	if !isAdmin {
		return sendError(s, m.ChannelID, "You need **bot owner** or **server admin** to set the wallet unit.")
	}
	unitVal := ""
	if len(rest) > 0 {
		unitVal = strings.TrimSpace(rest[0])
	}

	walletLock.Lock()
	defer walletLock.Unlock()
	data := loadWallet()
	data.Unit = unitVal
	if err := saveWallet(data); err != nil {
		return err
	}
	desc := "Wallet unit cleared (balance will show as number only)."
	if unitVal != "" {
		desc = fmt.Sprintf("Wallet unit set to **%s**. Balances will display like: 1,000 %s", unitVal, unitVal)
	}
	return sendEmbed(s, m.ChannelID, "Wallet Unit", desc, colorGreen)
}

// parseNonNeg parses a non-negative integer token.
func parseNonNeg(s string) (int64, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}
