package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Warnings returns the /warnings command definition and handler.
func Warnings(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "warnings",
			Description: "Lists all warnings for a user.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to check warnings for",
					Required:    true,
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
						Content: "Database is not connected. Cannot retrieve warnings.",
					},
				})
				return
			}

			// Get options
			var targetUser *discordgo.User
			for _, option := range i.ApplicationCommandData().Options {
				if option.Name == "user" {
					targetUser = option.UserValue(s)
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

			warnings, err := database.GetWarnings(context.Background(), i.GuildID, targetUser.ID)
			if err != nil {
				slog.Error("Error fetching warnings for user %s", "arg1", targetUser.ID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while fetching warnings.",
					},
				})
				return
			}

			if len(warnings) == 0 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "User Warnings",
								Description: fmt.Sprintf("<@%s> has no warnings.", targetUser.ID),
								Color:       0x00FF00, // Green
							},
						},
					},
				})
				return
			}

			// Build the embed with warnings
			embed := &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("Warnings for %s", targetUser.Username),
				Description: fmt.Sprintf("Total warnings: %d", len(warnings)),
				Color:       0xFFA500, // Orange
			}

			// Limit to maximum number of fields (25)
			maxFields := len(warnings)
			if maxFields > 25 {
				maxFields = 25
				embed.Footer = &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Showing 25 most recent out of %d warnings", len(warnings)),
				}
			}

			for idx, warning := range warnings[:maxFields] {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   fmt.Sprintf("Warning #%d", warning.ID),
					Value:  fmt.Sprintf("**Reason:** %s\n**Moderator:** <@%s>\n**Date:** %s", warning.Reason, warning.ModeratorID, warning.CreatedAt),
					Inline: false,
				})
				// To avoid huge messages or looping through everything, break early. (Already sliced up to maxFields)
				_ = idx
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
	}
}
