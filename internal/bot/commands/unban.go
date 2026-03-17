package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Unban returns the /unban command definition and handler.
func Unban(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionBanMembers)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "unban",
			Description:              "Unbans a user.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user_id",
					Description: "The ID of the user to unban",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "The reason for the unban",
					Required:    false,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			// Get options
			var targetUserID string
			var reason string
			for _, option := range i.ApplicationCommandData().Options {
				switch option.Name {
				case "user_id":
					targetUserID = option.StringValue()
				case "reason":
					reason = option.StringValue()
				}
			}

			if targetUserID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You must provide a valid user ID.",
					},
				})
				return
			}

			moderator := i.Member.User

			// Execute the unban
			err := s.GuildBanDelete(i.GuildID, targetUserID)
			if err != nil {
				slog.Error("Error unbanning user", "user_id", "arg1", targetUserID, "arg2", i.GuildID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to unban the user. Ensure the bot has higher permissions, and the user ID is correct and currently banned.",
					},
				})
				return
			}

			if reason == "" {
				reason = "No reason provided."
			}

			if database != nil {
				// Remove any temp ban if it exists
				err = database.RemoveTempBan(targetUserID, i.GuildID)
				if err != nil {
					slog.Error("Failed to remove temp ban for unbanned user", "error", err)
				}

				// Mark previous bans as resolved
				err = database.MarkAllUserModActionsResolved(context.Background(), i.GuildID, targetUserID, "ban")
				if err != nil {
					slog.Error("Failed to mark previous bans as resolved", "error", err)
				}

				// Log Moderation Action
				err = database.LogModActionComplex(context.Background(), i.GuildID, targetUserID, moderator.ID, "unban", reason, nil, nil)
				if err != nil {
					slog.Error("Error logging mod action 'unban'", "user_id", "arg1", targetUserID, "error", err)
				}
			}

			// Respond with Embed
			embed := &discordgo.MessageEmbed{
				Title:       "User Unbanned",
				Description: fmt.Sprintf("<@%s> has been unbanned.", targetUserID),
				Color:       0x00FF00, // Green
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Reason",
						Value:  reason,
						Inline: false,
					},
					{
						Name:   "Moderator",
						Value:  fmt.Sprintf("<@%s>", moderator.ID),
						Inline: true,
					},
				},
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})

			SendModLog(s, database, i.GuildID, embed)
		},
	}
}
