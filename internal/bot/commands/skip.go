package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Skip returns the skip slash command.
func Skip(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "skip",
			Description: "Skip the currently playing track",
			DMPermission: new(bool),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			q, err := database.SkipMusic(ctx, i.GuildID)
			if err != nil {
				SendError(s, i, "Failed to skip track")
				return
			}

			if q == nil {
				SendError(s, i, "The queue is currently empty.")
				return
			}

			embed := &discordgo.MessageEmbed{
				Title:       "⏭️ Track Skipped",
				Description: fmt.Sprintf("Skipped: **%s**", q.Query),
				Color:       0x1DB954, // Spotify Green
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Skipped by %s", i.Member.User.String()),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			// Try to peek next
			nextList, nextErr := database.GetQueue(ctx, i.GuildID)
			if nextErr == nil && len(nextList) > 0 {
				next := nextList[0]
				embed.Description += fmt.Sprintf("\n\n**Next up:** %s", next.Query)
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
