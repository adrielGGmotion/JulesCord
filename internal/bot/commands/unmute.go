package commands

import (
	"context"
	"fmt"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Unmute returns the command definition for the /unmute command
func Unmute(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "unmute",
			Description: "Remove a timeout from a user.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to unmute",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for the unmute",
					Required:    false,
				},
			},
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionModerateMembers); return &p }(),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			// Defer response
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			options := i.ApplicationCommandData().Options
			targetUser := options[0].UserValue(s)

			reason := "No reason provided."
			if len(options) > 1 {
				reason = options[1].StringValue()
			}

			// Remove timeout in Discord
			err = s.GuildMemberTimeout(i.GuildID, targetUser.ID, nil, discordgo.WithAuditLogReason(reason))
			if err != nil {
				SendError(s, i, fmt.Sprintf("Failed to unmute user: %v", err))
				return
			}

			// Save to database
			if database != nil {
				err = database.RemoveMute(context.Background(), i.GuildID, targetUser.ID)
				if err != nil {
					SendError(s, i, "Failed to remove mute from database.")
					return
				}

				// Log to mod channel
				_ = database.LogModAction(context.Background(), i.GuildID, i.Member.User.ID, targetUser.ID, "Unmute", reason)

				// Send to mod log channel if configured
				logChannelID, err := database.GetGuildLogChannel(context.Background(), i.GuildID)
				if err == nil && logChannelID != "" {
					embed := &discordgo.MessageEmbed{
						Title: "🔊 User Unmuted",
						Color: 0x00FF00, // Green
						Fields: []*discordgo.MessageEmbedField{
							{Name: "User", Value: fmt.Sprintf("<@%s> (%s)", targetUser.ID, targetUser.ID), Inline: true},
							{Name: "Moderator", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
							{Name: "Reason", Value: reason, Inline: false},
						},
						Timestamp: time.Now().Format(time.RFC3339),
					}
					_, _ = s.ChannelMessageSendEmbed(logChannelID, embed)
				}
			}

			embed := &discordgo.MessageEmbed{
				Title:       "🔊 User Unmuted",
				Description: fmt.Sprintf("Successfully unmuted <@%s>.", targetUser.ID),
				Color:       0x00FF00, // Green
			}
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
