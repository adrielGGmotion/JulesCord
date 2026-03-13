package commands

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// LogModerationAction sends an embed to the guild's configured mod log channel.
func LogModerationAction(s *discordgo.Session, database *db.DB, guildID string, action string, user *discordgo.User, moderator *discordgo.User, reason string) {
	if database == nil {
		return
	}

	config, err := database.GetGuildConfig(context.Background(), guildID)
	if err != nil || config.ModLogChannelID == nil {
		return // No config or mod log channel set
	}

	color := 0x000000
	switch action {
	case "Warn":
		color = 0xFFA500 // Orange
	case "Kick":
		color = 0xFF4500 // Red-Orange
	case "Ban":
		color = 0xFF0000 // Red
	case "Purge":
		color = 0x808080 // Gray
	}

	description := fmt.Sprintf("**Moderator:** <@%s>\n**Reason:** %s", moderator.ID, reason)
	if user != nil {
		description = fmt.Sprintf("**User:** <@%s> (%s)\n", user.ID, user.ID) + description
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Moderation Action: %s", action),
		Description: description,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if user != nil {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: user.AvatarURL(""),
		}
	}

	_, err = s.ChannelMessageSendEmbed(*config.ModLogChannelID, embed)
	if err != nil {
		log.Printf("Failed to send moderation log to channel %s: %v", *config.ModLogChannelID, err)
	}
}
