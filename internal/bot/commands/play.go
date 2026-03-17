package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Play returns the play slash command.
func Play(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:         "play",
			Description:  "Play a song from YouTube/Spotify or a query",
			DMPermission: new(bool),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "query",
					Description: "Song name or URL",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
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

			var query string
			for _, opt := range i.ApplicationCommandData().Options {
				if opt.Name == "query" {
					query = opt.StringValue()
					break
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			q, err := database.PlayMusic(ctx, i.GuildID, i.Member.User.ID, query)
			if err != nil {
				SendError(s, i, "Failed to enqueue music")
				return
			}

			embed := &discordgo.MessageEmbed{
				Title:       "🎵 Added to Queue",
				Description: fmt.Sprintf("**Query:** %s\n**Position:** #%d", q.Query, q.Position),
				Color:       0x1DB954, // Spotify Green
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Requested by %s", i.Member.User.String()),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
