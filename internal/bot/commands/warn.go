package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Warn returns the /warn command definition and handler.
func Warn(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionKickMembers | discordgo.PermissionManageMessages)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "warn",
			Description:              "Warns a user.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to warn",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "The reason for the warning",
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

			if database == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not connected. Cannot issue warnings.",
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

			// Upsert user if they don't exist yet (in case they haven't sent a message)
			err := database.UpsertUser(context.Background(), targetUser.ID, targetUser.Username, targetUser.GlobalName, targetUser.AvatarURL(""))
			if err != nil {
				slog.Error("Failed to upsert user %s for warning", "arg1", targetUser.ID, "error", err)
			}

			moderator := i.Member.User

			// Add Warning
			err = database.AddWarning(context.Background(), i.GuildID, targetUser.ID, moderator.ID, reason)
			if err != nil {
				slog.Error("Error adding warning for user %s", "arg1", targetUser.ID, "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "An error occurred while trying to warn the user.",
					},
				})
				return
			}

			// Log Moderation Action
			err = database.LogModActionComplex(context.Background(), i.GuildID, targetUser.ID, moderator.ID, "warn", reason, nil, evidenceURL)
			if err != nil {
				slog.Error("Error logging mod action 'warn' for user %s", "arg1", targetUser.ID, "error", err)
			}

			// Respond with Embed
			embed := &discordgo.MessageEmbed{
				Title:       "User Warned",
				Description: fmt.Sprintf("<@%s> has been warned.", targetUser.ID),
				Color:       0xFFA500, // Orange
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

			// Automated Punishments
			warnings, err := database.GetWarnings(context.Background(), i.GuildID, targetUser.ID)
			if err == nil {
				warningCount := len(warnings)
				rules, err := database.GetWarnAutomationRules(context.Background(), i.GuildID)
				if err == nil {
					for _, rule := range rules {
						if warningCount == rule.WarningThreshold {
							autoReason := fmt.Sprintf("Automated punishment: Reached %d warnings", rule.WarningThreshold)

							switch rule.Action {
							case "mute":
								if rule.Duration != nil {
									parsedDurationStr := *rule.Duration
									if strings.HasSuffix(parsedDurationStr, "d") {
										var days int
										fmt.Sscanf(parsedDurationStr, "%dd", &days)
										parsedDurationStr = fmt.Sprintf("%dh", days*24)
									}

									duration, err := time.ParseDuration(parsedDurationStr)
									if err == nil {
										expiresAt := time.Now().Add(duration)
										err = s.GuildMemberTimeout(i.GuildID, targetUser.ID, &expiresAt, discordgo.WithAuditLogReason(autoReason))
										if err == nil {
											_ = database.AddMute(context.Background(), i.GuildID, targetUser.ID, s.State.User.ID, autoReason, expiresAt)
											_ = database.LogModActionComplex(context.Background(), i.GuildID, targetUser.ID, s.State.User.ID, "mute", autoReason, rule.Duration, nil)

											muteEmbed := &discordgo.MessageEmbed{
												Title:       "User Muted (Automated)",
												Description: fmt.Sprintf("<@%s> has been muted automatically.", targetUser.ID),
												Color:       0xFFA500, // Orange
												Fields: []*discordgo.MessageEmbedField{
													{Name: "Reason", Value: autoReason, Inline: false},
													{Name: "Duration", Value: *rule.Duration, Inline: true},
												},
											}
											SendModLog(s, database, i.GuildID, muteEmbed)
										}
									}
								}
							case "kick":
								err = s.GuildMemberDeleteWithReason(i.GuildID, targetUser.ID, autoReason)
								if err == nil {
									_ = database.LogModActionComplex(context.Background(), i.GuildID, targetUser.ID, s.State.User.ID, "kick", autoReason, nil, nil)

									kickEmbed := &discordgo.MessageEmbed{
										Title:       "User Kicked (Automated)",
										Description: fmt.Sprintf("<@%s> has been kicked automatically.", targetUser.ID),
										Color:       0xFF0000, // Red
										Fields: []*discordgo.MessageEmbedField{
											{Name: "Reason", Value: autoReason, Inline: false},
										},
									}
									SendModLog(s, database, i.GuildID, kickEmbed)
								}
							case "ban":
								err = s.GuildBanCreateWithReason(i.GuildID, targetUser.ID, autoReason, 0)
								if err == nil {
									if rule.Duration != nil {
										parsedDurationStr := *rule.Duration
										if strings.HasSuffix(parsedDurationStr, "d") {
											var days int
											fmt.Sscanf(parsedDurationStr, "%dd", &days)
											parsedDurationStr = fmt.Sprintf("%dh", days*24)
										}

										duration, err := time.ParseDuration(parsedDurationStr)
										if err == nil {
											unbanAt := time.Now().Add(duration)
											_ = database.AddTempBan(targetUser.ID, i.GuildID, unbanAt)
										}
									}

									_ = database.LogModActionComplex(context.Background(), i.GuildID, targetUser.ID, s.State.User.ID, "ban", autoReason, rule.Duration, nil)

									banEmbed := &discordgo.MessageEmbed{
										Title:       "User Banned (Automated)",
										Description: fmt.Sprintf("<@%s> has been banned automatically.", targetUser.ID),
										Color:       0xFF0000, // Red
										Fields: []*discordgo.MessageEmbedField{
											{Name: "Reason", Value: autoReason, Inline: false},
										},
									}
									if rule.Duration != nil {
										banEmbed.Fields = append(banEmbed.Fields, &discordgo.MessageEmbedField{Name: "Duration", Value: *rule.Duration, Inline: true})
									}
									SendModLog(s, database, i.GuildID, banEmbed)
								}
							}
						}
					}
				}
			}
		},
	}
}
