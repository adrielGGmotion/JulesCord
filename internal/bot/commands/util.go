package commands

import (
	"context"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// SendModLog sends an embed to the configured moderation log channel, if one is set.
func SendModLog(s *discordgo.Session, database *db.DB, guildID string, embed *discordgo.MessageEmbed) {
	if database == nil {
		return
	}

	channelID, err := database.GetGuildLogChannel(context.Background(), guildID)
	if err != nil {
		slog.Error("Error getting log channel for guild %s", "arg1", guildID, "error", err)
		return
	}

	if channelID == "" {
		// No log channel configured
		return
	}

	_, err = s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		slog.Error("Failed to send mod log to channel %s in guild %s", "arg1", channelID, "arg2", guildID, "error", err)
	}
}

// SendError sends an error message reply to an interaction.
func SendError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// SendErrorEdit sends an error message reply to a deferred interaction.
func SendErrorEdit(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	content := "❌ " + message
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
}

// SendEmbed sends an embed reply to an interaction.
func SendEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
