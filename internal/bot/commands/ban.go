package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Ban returns the /ban command definition and handler.
func Ban(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionBanMembers)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "ban",
			Description:              "Bans a user.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to ban",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "The reason for the ban",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "evidence",
					Description: "Evidence attachment (image/log)",
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
			var targetUser *discordgo.User
			var reason string
			var evidenceURL *string
			for _, option := range i.ApplicationCommandData().Options {
				switch option.Name {
				case "user":
					targetUser = option.UserValue(s)
				case "reason":
					reason = option.StringValue()
				case "evidence":
					attID, ok := option.Value.(string)
					if ok {
						if i.ApplicationCommandData().Resolved != nil && i.ApplicationCommandData().Resolved.Attachments != nil {
							att := i.ApplicationCommandData().Resolved.Attachments[attID]
							if att != nil {
								url := att.URL
								evidenceURL = &url
							}
						}
					}
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

			// Execute the ban
			err := s.GuildBanCreateWithReason(i.GuildID, targetUser.ID, fmt.Sprintf("Banned by %s: %s", moderator.Username, reason), 0)
			if err != nil {
				slog.Error("Error banning user %s from guild %s", "arg1", targetUser.ID, "arg2", i.GuildID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to ban the user. Ensure the bot has higher permissions than the target user.",
					},
				})
				return
			}

			// Upsert user if they don't exist
			if database != nil {
				err = database.UpsertUser(context.Background(), targetUser.ID, targetUser.Username, targetUser.GlobalName, targetUser.AvatarURL(""))
				if err != nil {
					slog.Error("Failed to upsert user %s for ban", "arg1", targetUser.ID, "error", err)
				}

				// Log Moderation Action
				err = database.LogModActionComplex(context.Background(), i.GuildID, targetUser.ID, moderator.ID, "ban", reason, nil, evidenceURL)
				if err != nil {
					slog.Error("Error logging mod action 'ban' for user %s", "arg1", targetUser.ID, "error", err)
				}
			}

			// Respond with Embed
			embed := &discordgo.MessageEmbed{
				Title:       "User Banned",
				Description: fmt.Sprintf("<@%s> has been banned.", targetUser.ID),
				Color:       0xFF0000, // Red
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
			if evidenceURL != nil {
				embed.Image = &discordgo.MessageEmbedImage{
					URL: *evidenceURL,
				}
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
