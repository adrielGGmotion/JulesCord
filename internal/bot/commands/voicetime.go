package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"github.com/bwmarrin/discordgo"
)

func formatVoiceTime(totalSeconds int64) string {
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func NewVoiceTimeCommand(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:         "voicetime",
			Description:  "Check voice time stats or leaderboard",
			DMPermission: func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "stats",
					Description: "Check your or someone else's total voice time",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "User to check",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    false,
						},
					},
				},
				{
					Name:        "leaderboard",
					Description: "View top 10 users by voice time",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0]

			switch subcommand.Name {
			case "stats":
				targetUserID := i.Member.User.ID
				var targetUser *discordgo.User = i.Member.User

				if len(subcommand.Options) > 0 {
					targetUser = subcommand.Options[0].UserValue(s)
					targetUserID = targetUser.ID
				}

				totalSeconds, err := database.GetVoiceTime(context.Background(), i.GuildID, targetUserID)
				if err != nil {
					SendError(s, i, "Failed to retrieve voice time")
					return
				}

				timeStr := formatVoiceTime(totalSeconds)

				embed := &discordgo.MessageEmbed{
					Title:       fmt.Sprintf("🎙️ Voice Time - %s", targetUser.Username),
					Description: fmt.Sprintf("**Total Time:** %s", timeStr),
					Color:       0x00FF00,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: targetUser.AvatarURL(""),
					},
				}
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			case "leaderboard":
				stats, err := database.GetTopVoiceUsers(context.Background(), i.GuildID)
				if err != nil {
					SendError(s, i, "Failed to load voice time leaderboard")
					return
				}

				if len(stats) == 0 {
					SendError(s, i, "No voice time stats found for this server yet.")
					return
				}

				description := ""
				for index, stat := range stats {
					timeStr := formatVoiceTime(stat.TotalSeconds)
					description += fmt.Sprintf("**#%d** <@%s> - %s\n", index+1, stat.UserID, timeStr)
				}

				embed := &discordgo.MessageEmbed{
					Title:       "🏆 Voice Time Leaderboard",
					Description: description,
					Color:       0xFFD700,
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
