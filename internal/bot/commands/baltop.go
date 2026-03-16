package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Baltop returns the /baltop command definition and handler.
func Baltop(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "baltop",
			Description: "Displays the top 10 users by coins in the server.",
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
						Content: "Database is not connected. Cannot fetch baltop data.",
					},
				})
				return
			}

			ctx := context.Background()

			// Fetch top users by coins
			topUsers, err := database.GetTopUsersByCoins(ctx, i.GuildID)
			if err != nil {
				slog.Error("Failed to fetch top users by coins", "error", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to fetch baltop data.",
					},
				})
				return
			}

			if len(topUsers) == 0 {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No users have any coins yet in this server.",
					},
				})
				return
			}

			// Build baltop string
			var sb strings.Builder
			for index, user := range topUsers {
				sb.WriteString(fmt.Sprintf("**%d.** <@%s> — %d coins\n", index+1, user.UserID, user.Coins))
			}

			// Send embed
			embed := &discordgo.MessageEmbed{
				Title:       "Server Coin Leaderboard",
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
