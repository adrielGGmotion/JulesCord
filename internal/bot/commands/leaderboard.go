package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Leaderboard returns the /leaderboard command definition and handler.
func Leaderboard(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "leaderboard",
			Description: "Displays the top 10 users by XP in the server.",
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
						Content: "Database is not connected. Cannot fetch leaderboard data.",
					},
				})
				return
			}

			ctx := context.Background()

			// Fetch top users
			topUsers, err := database.GetTopUsersByXP(ctx, i.GuildID)
			if err != nil {
				slog.Error("Failed to fetch top users", "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to fetch leaderboard data.",
					},
				})
				return
			}

			if len(topUsers) == 0 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No users have earned XP yet in this server.",
					},
				})
				return
			}

			// Build leaderboard string
			var sb strings.Builder
			for index, user := range topUsers {
				sb.WriteString(fmt.Sprintf("**%d.** <@%s> — Level %d (%d XP)\n", index+1, user.UserID, user.Level, user.XP))
			}

			// Send embed
			embed := &discordgo.MessageEmbed{
				Title:       "Server XP Leaderboard",
				Description: sb.String(),
				Color:       0xf1c40f, // Yellow/Gold
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
