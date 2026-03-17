package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Stop returns the stop slash command.
func Stop(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:         "stop",
			Description:  "Stop music and clear the queue",
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

			err = database.StopMusic(ctx, i.GuildID)
			if err != nil {
				SendError(s, i, "Failed to clear the queue")
				return
			}

			embed := &discordgo.MessageEmbed{
				Title:       "⏹️ Music Stopped",
				Description: "Playback stopped and queue cleared.",
				Color:       0xED4245, // Red
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Stopped by %s", i.Member.User.String()),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
