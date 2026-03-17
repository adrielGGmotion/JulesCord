package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// ClearWarnings returns the /clearwarnings command definition and handler.
func ClearWarnings(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionKickMembers | discordgo.PermissionManageMessages)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "clearwarnings",
			Description:              "Clears all warnings for a user.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to clear warnings for",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "The reason for clearing warnings",
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

			if database == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not connected. Cannot clear warnings.",
					},
				})
				return
			}

			// Get options
			var targetUser *discordgo.User
			var reason string
			for _, option := range i.ApplicationCommandData().Options {
				switch option.Name {
				case "user":
					targetUser = option.UserValue(s)
				case "reason":
					reason = option.StringValue()
				}
			}

			if targetUser == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Could not find the specified user.",
					},
				})
				return
			}

			moderator := i.Member.User

			// Clear warnings
			err := database.ClearWarnings(context.Background(), i.GuildID, targetUser.ID)
			if err != nil {
				slog.Error("Error clearing warnings for user", "user_id", "arg1", targetUser.ID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while trying to clear warnings.",
					},
				})
				return
			}

			if reason == "" {
				reason = "No reason provided."
			}

			// Log Moderation Action
			err = database.LogModActionComplex(context.Background(), i.GuildID, targetUser.ID, moderator.ID, "clearwarnings", reason, nil, nil)
			if err != nil {
				slog.Error("Error logging mod action 'clearwarnings'", "user_id", "arg1", targetUser.ID, "error", err)
			}

			// Respond with Embed
			embed := &discordgo.MessageEmbed{
				Title:       "Warnings Cleared",
				Description: fmt.Sprintf("All warnings for <@%s> have been cleared.", targetUser.ID),
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
