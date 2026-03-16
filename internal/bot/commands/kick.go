package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Kick returns the /kick command definition and handler.
func Kick(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionKickMembers)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "kick",
			Description:              "Kicks a user.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to kick",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "The reason for the kick",
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

			// Execute the kick
			err := s.GuildMemberDeleteWithReason(i.GuildID, targetUser.ID, fmt.Sprintf("Kicked by %s: %s", moderator.Username, reason))
			if err != nil {
				slog.Error("Error kicking user %s from guild %s", "arg1", targetUser.ID, "arg2", i.GuildID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to kick the user. Ensure the bot has higher permissions than the target user.",
					},
				})
				return
			}

			// Upsert user if they don't exist
			if database != nil {
				err = database.UpsertUser(context.Background(), targetUser.ID, targetUser.Username, targetUser.GlobalName, targetUser.AvatarURL(""))
				if err != nil {
					slog.Error("Failed to upsert user %s for kick", "arg1", targetUser.ID, "error", err)
				}

				// Log Moderation Action
				err = database.LogModActionComplex(context.Background(), i.GuildID, targetUser.ID, moderator.ID, "kick", reason, nil, evidenceURL)
				if err != nil {
					slog.Error("Error logging mod action 'kick' for user %s", "arg1", targetUser.ID, "error", err)
				}
			}

			// Respond with Embed
			embed := &discordgo.MessageEmbed{
				Title:       "User Kicked",
				Description: fmt.Sprintf("<@%s> has been kicked.", targetUser.ID),
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
