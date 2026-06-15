package wallet

import "github.com/bwmarrin/discordgo"

// respondEphemeral sends an ephemeral plain-text interaction response.
func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, text string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:         text,
			Flags:           discordgo.MessageFlagsEphemeral,
			AllowedMentions: noPing(),
		},
	})
}

// respondEmbed sends a non-ephemeral embed interaction response.
func respondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, title, desc string, color int) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:          []*discordgo.MessageEmbed{{Title: title, Description: desc, Color: color}},
			AllowedMentions: noPing(),
		},
	})
}

// sendEmbed sends an embed message to a channel (prefix path).
func sendEmbed(s *discordgo.Session, channelID, title, desc string, color int) error {
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:          []*discordgo.MessageEmbed{{Title: title, Description: desc, Color: color}},
		AllowedMentions: noPing(),
	})
	return err
}

// sendError sends a red error embed to a channel (prefix path).
func sendError(s *discordgo.Session, channelID, text string) error {
	return sendEmbed(s, channelID, "Error", text, 0xff0000)
}
