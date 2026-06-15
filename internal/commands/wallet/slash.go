package wallet

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// subOptions returns the invoked subcommand name and its nested options.
func subOptions(data discordgo.ApplicationCommandInteractionData) (string, []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(data.Options) == 0 {
		return "check", nil
	}
	opt := data.Options[0]
	if opt.Type == discordgo.ApplicationCommandOptionSubCommand {
		return opt.Name, opt.Options
	}
	return opt.Name, nil
}

func findOpt(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, o := range opts {
		if o.Name == name {
			return o
		}
	}
	return nil
}

// RunSlash handles the wallet slash command.
func (w *Wallet) RunSlash(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if i.GuildID == "" {
		return respondEphemeral(s, i, "This command can only be used in a server.")
	}

	data := i.ApplicationCommandData()
	callerID := ""
	if i.Member != nil && i.Member.User != nil {
		callerID = i.Member.User.ID
	}

	isAdmin, ok := callerAdmin(s, i.GuildID, callerID, i.Member)
	if !ok {
		return respondEphemeral(s, i, "Could not load server information.")
	}

	sub, nested := subOptions(data)

	switch sub {
	case "check":
		return w.slashCheck(s, i, nested, callerID, isAdmin)
	case "credit", "debit":
		return w.slashCreditDebit(s, i, sub, nested, callerID, isAdmin)
	case "init":
		return w.slashInit(s, i, nested, isAdmin)
	case "reset":
		return w.slashReset(s, i, nested, isAdmin)
	case "unit":
		return w.slashUnit(s, i, nested, isAdmin)
	default:
		return w.slashCheck(s, i, nil, callerID, isAdmin)
	}
}

func (w *Wallet) slashCheck(s *discordgo.Session, i *discordgo.InteractionCreate, nested []*discordgo.ApplicationCommandInteractionDataOption, callerID string, isAdmin bool) error {
	targets := []string{callerID}
	if opt := findOpt(nested, "user"); opt != nil {
		targets = []string{opt.UserValue(nil).ID}
	}

	if len(targets) == 1 && targets[0] != callerID && !isAdmin {
		return respondEphemeral(s, i, "You need **bot owner** or **server admin** to view others' wallets.")
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
		return respondEphemeral(s, i, fmt.Sprintf("The following user(s) have not been initialized: **%s**. Use **wallet init** first.", strings.Join(notInited, ", ")))
	}

	var lines []string
	for _, id := range targets {
		bal, _ := data.balanceIfExists(id)
		lines = append(lines, fmt.Sprintf("**%s**: %s", userMention(id), formatBalance(bal, data.Unit)))
	}
	return respondEmbed(s, i, "Wallet", strings.Join(lines, "\n"), colorGreen)
}

func (w *Wallet) slashCreditDebit(s *discordgo.Session, i *discordgo.InteractionCreate, sub string, nested []*discordgo.ApplicationCommandInteractionDataOption, callerID string, isAdmin bool) error {
	if !isAdmin {
		return respondEphemeral(s, i, "You need **bot owner** or **server admin** to credit/debit balance.")
	}
	var amount int64
	if opt := findOpt(nested, "amount"); opt != nil {
		amount = opt.IntValue()
	}
	if amount <= 0 {
		return respondEphemeral(s, i, "Invalid amount (must be positive).")
	}
	targets := []string{callerID}
	if opt := findOpt(nested, "user"); opt != nil {
		targets = []string{opt.UserValue(nil).ID}
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
		return respondEphemeral(s, i, fmt.Sprintf("The following user(s) have not been initialized: **%s**. Use **wallet init** first.", strings.Join(notInited, ", ")))
	}

	now := nowISO()
	if sub == "credit" {
		for _, id := range targets {
			data.addBalance(id, amount, now)
		}
		if err := saveWallet(data); err != nil {
			return err
		}
		names := mentions(targets)
		return respondEmbed(s, i, "Wallet", fmt.Sprintf("Added **%s** to: %s", formatBalance(uint64(amount), data.Unit), strings.Join(names, ", ")), colorGreen)
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
		return respondEmbed(s, i, "Wallet", fmt.Sprintf("Removed **%s** from: %s", amtStr, strings.Join(okNames, ", ")), colorGreen)
	}
	return respondEmbed(s, i, "Wallet (partial)", fmt.Sprintf("Removed from: %s. Insufficient balance: %s", strings.Join(okNames, ", "), strings.Join(failed, ", ")), colorAmber)
}

func (w *Wallet) slashInit(s *discordgo.Session, i *discordgo.InteractionCreate, nested []*discordgo.ApplicationCommandInteractionDataOption, isAdmin bool) error {
	if !isAdmin {
		return respondEphemeral(s, i, "You need **bot owner** or **server admin** to init wallets.")
	}
	targets := w.slashTargetsOrAll(s, i, nested)
	if len(targets) == 0 {
		return respondEphemeral(s, i, "No users to init (could not load server members from cache or API).")
	}
	var amount int64
	if opt := findOpt(nested, "amount"); opt != nil {
		amount = opt.IntValue()
	}
	if amount < 0 {
		amount = 0
	}

	walletLock.Lock()
	defer walletLock.Unlock()
	data := loadWallet()
	now := nowISO()
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
	msg := fmt.Sprintf("Initialized **%d** user(s) with balance **%s**.", created, balStr)
	if skipped > 0 {
		msg = fmt.Sprintf("Initialized **%d** new user(s) with balance **%s**. Skipped **%d** already in wallet.", created, balStr, skipped)
	}
	return respondEmbed(s, i, "Wallet Init", msg, colorGreen)
}

func (w *Wallet) slashReset(s *discordgo.Session, i *discordgo.InteractionCreate, nested []*discordgo.ApplicationCommandInteractionDataOption, isAdmin bool) error {
	if !isAdmin {
		return respondEphemeral(s, i, "You need **bot owner** or **server admin** to reset wallets.")
	}
	targets := w.slashTargetsOrAll(s, i, nested)
	if len(targets) == 0 {
		return respondEphemeral(s, i, "No users to reset (could not load server members from cache or API).")
	}
	var amount int64
	if opt := findOpt(nested, "amount"); opt != nil {
		amount = opt.IntValue()
	}
	if amount < 0 {
		amount = 0
	}

	walletLock.Lock()
	defer walletLock.Unlock()
	data := loadWallet()
	now := nowISO()
	for _, id := range targets {
		data.initUser(id, amount, now)
	}
	if err := saveWallet(data); err != nil {
		return err
	}
	balStr := formatBalance(clampNonNeg(amount), data.Unit)
	return respondEmbed(s, i, "Wallet Reset", fmt.Sprintf("Reset **%d** user(s) to balance **%s**.", len(targets), balStr), colorGreen)
}

func (w *Wallet) slashUnit(s *discordgo.Session, i *discordgo.InteractionCreate, nested []*discordgo.ApplicationCommandInteractionDataOption, isAdmin bool) error {
	if !isAdmin {
		return respondEphemeral(s, i, "You need **bot owner** or **server admin** to set the wallet unit.")
	}
	unitVal := ""
	if opt := findOpt(nested, "unit"); opt != nil {
		unitVal = strings.TrimSpace(opt.StringValue())
	}

	walletLock.Lock()
	defer walletLock.Unlock()
	data := loadWallet()
	data.Unit = unitVal
	if err := saveWallet(data); err != nil {
		return err
	}
	msg := "Wallet unit cleared (balance will show as number only)."
	if unitVal != "" {
		msg = fmt.Sprintf("Wallet unit set to **%s**. Balances will display like: 1,000 %s", unitVal, unitVal)
	}
	return respondEmbed(s, i, "Wallet Unit", msg, colorGreen)
}

// slashTargetsOrAll returns the explicit user option or all human members.
func (w *Wallet) slashTargetsOrAll(s *discordgo.Session, i *discordgo.InteractionCreate, nested []*discordgo.ApplicationCommandInteractionDataOption) []string {
	if opt := findOpt(nested, "user"); opt != nil {
		return []string{opt.UserValue(nil).ID}
	}
	return guildHumanMembers(s, i.GuildID)
}

func mentions(ids []string) []string {
	out := make([]string, len(ids))
	for idx, id := range ids {
		out[idx] = userMention(id)
	}
	return out
}
