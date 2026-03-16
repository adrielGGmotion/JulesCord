package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// ServerLeaderboard returns the serverleaderboard command definition.
func ServerLeaderboard(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "serverleaderboard",
			Description: "Displays the top users by custom server points",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				slog.Error("Failed to defer interaction", "error", err)
				return
			}

			ctx := context.Background()
			entries, err := database.GetServerLeaderboard(ctx, i.GuildID, 10)
			if err != nil {
				slog.Error("Failed to get server leaderboard", "error", err)
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: stringPtr("An error occurred while fetching the leaderboard."),
				})
				return
			}

			if len(entries) == 0 {
				embed := &discordgo.MessageEmbed{
					Title:       "Server Leaderboard",
					Description: "No one has any points yet!",
					Color:       0x00FF00,
				}
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{embed},
				})
				return
			}

			var sb strings.Builder
			for index, entry := range entries {
				rankStr := ""
				switch index {
				case 0:
					rankStr = "🥇"
				case 1:
					rankStr = "🥈"
				case 2:
					rankStr = "🥉"
				default:
					rankStr = fmt.Sprintf("**#%d**", index+1)
				}
				sb.WriteString(fmt.Sprintf("%s <@%s> - %d points\n", rankStr, entry.UserID, entry.Points))
			}

			embed := &discordgo.MessageEmbed{
				Title:       "🏆 Server Leaderboard",
				Description: sb.String(),
				Color:       0x00FF00,
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
