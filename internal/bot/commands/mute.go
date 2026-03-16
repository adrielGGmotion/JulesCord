package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Mute returns the command definition for the /mute command
func Mute(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "mute",
			Description: "Timeout a user for a specific duration.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to mute",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "duration",
					Description: "Duration of the mute (e.g., 10m, 1h, 1d)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for the mute",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "evidence",
					Description: "Evidence attachment (image/log)",
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
			var targetUser *discordgo.User
			var durationStr string
			reason := "No reason provided."
			var evidenceURL *string

			for _, option := range options {
				switch option.Name {
				case "user":
					targetUser = option.UserValue(s)
				case "duration":
					durationStr = option.StringValue()
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

			// Parse duration
			var duration time.Duration
			if strings.HasSuffix(durationStr, "d") {
				daysStr := strings.TrimSuffix(durationStr, "d")
				days, err := strconv.Atoi(daysStr)
				if err != nil {
					SendError(s, i, "Invalid duration format. Use e.g. 10m, 1h, 1d.")
					return
				}
				duration = time.Duration(days) * 24 * time.Hour
			} else {
				parsedDuration, err := time.ParseDuration(durationStr)
				if err != nil {
					SendError(s, i, "Invalid duration format. Use e.g. 10m, 1h.")
					return
				}
				duration = parsedDuration
			}

			expiresAt := time.Now().Add(duration)

			// Execute timeout in Discord
			err = s.GuildMemberTimeout(i.GuildID, targetUser.ID, &expiresAt, discordgo.WithAuditLogReason(reason))
			if err != nil {
				SendError(s, i, fmt.Sprintf("Failed to mute user: %v", err))
				return
			}

			// Save to database
			if database != nil {
				err = database.AddMute(context.Background(), i.GuildID, targetUser.ID, i.Member.User.ID, reason, expiresAt)
				if err != nil {
					SendError(s, i, "Failed to save mute to database.")
					return
				}

				// Log to mod channel
				_ = database.LogModActionComplex(context.Background(), i.GuildID, i.Member.User.ID, targetUser.ID, "Mute", reason, &durationStr, evidenceURL)

				// Send to mod log channel if configured
				logChannelID, err := database.GetGuildLogChannel(context.Background(), i.GuildID)
				if err == nil && logChannelID != "" {
					embed := &discordgo.MessageEmbed{
						Title: "🔇 User Muted",
						Color: 0xFFA500, // Orange
						Fields: []*discordgo.MessageEmbedField{
							{Name: "User", Value: fmt.Sprintf("<@%s> (%s)", targetUser.ID, targetUser.ID), Inline: true},
							{Name: "Moderator", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
							{Name: "Duration", Value: durationStr, Inline: true},
							{Name: "Expires", Value: fmt.Sprintf("<t:%d:R>", expiresAt.Unix()), Inline: true},
							{Name: "Reason", Value: reason, Inline: false},
						},
						Timestamp: time.Now().Format(time.RFC3339),
					}
					if evidenceURL != nil {
						embed.Image = &discordgo.MessageEmbedImage{
							URL: *evidenceURL,
						}
					}
					_, _ = s.ChannelMessageSendEmbed(logChannelID, embed)
				}
			}

			embed := &discordgo.MessageEmbed{
				Title:       "🔇 User Muted",
				Description: fmt.Sprintf("Successfully muted <@%s> for %s.", targetUser.ID, durationStr),
				Color:       0xFFA500, // Orange
			}
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
