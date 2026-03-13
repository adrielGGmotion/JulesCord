package commands

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// SendModLog sends an embed to the configured moderation log channel, if one is set.
func SendModLog(s *discordgo.Session, database *db.DB, guildID string, embed *discordgo.MessageEmbed) {
	if database == nil {
		return
	}

	channelID, err := database.GetGuildLogChannel(context.Background(), guildID)
	if err != nil {
		log.Printf("Error getting log channel for guild %s: %v", guildID, err)
		return
	}

	if channelID == "" {
		// No log channel configured
		return
	}

	_, err = s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Failed to send mod log to channel %s in guild %s: %v", channelID, guildID, err)
	}
}
