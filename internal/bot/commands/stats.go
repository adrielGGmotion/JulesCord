package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Global reference for uptime tracking
var startTime = time.Now()

// Stats creates the /stats command.
func Stats(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "stats",
			Description: "Displays guild count, user count, uptime, and commands run",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			uptime := time.Since(startTime).Round(time.Second)

			guildCount := 0
			userCount := 0
			commandCount := 0

			if database != nil {
				// Fetch stats from database
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				var dbErr error
				var gc, uc, cc int64
				gc, uc, cc, dbErr = database.GetStats(ctx)
				if dbErr == nil {
					guildCount = int(gc)
					userCount = int(uc)
					commandCount = int(cc)
				}
			} else {
				// Fallback to session cache if DB is not available
				guildCount = len(s.State.Guilds)
			}

			embed := &discordgo.MessageEmbed{
				Title: "📊 JulesCord Stats",
				Color: 0x42f554, // Green
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Uptime",
						Value:  fmt.Sprintf("`%s`", uptime.String()),
						Inline: true,
					},
					{
						Name:   "Servers",
						Value:  fmt.Sprintf("`%d`", guildCount),
						Inline: true,
					},
					{
						Name:   "Users Tracked",
						Value:  fmt.Sprintf("`%d`", userCount),
						Inline: true,
					},
					{
						Name:   "Commands Executed",
						Value:  fmt.Sprintf("`%d`", commandCount),
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Autonomous Data Collection",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
		},
	}
}
