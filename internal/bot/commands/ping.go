package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Ping creates the /ping command.
func Ping() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "ping",
			Description: "Replies with Pong and bot latency!",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Acknowledge the interaction immediately since we might need time
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})
			if err != nil {
				return
			}

			// Calculate latency based on session heartbeat
			latency := s.HeartbeatLatency()
			latencyMs := latency.Milliseconds()

			// Build the response string
			var content string
			if latencyMs > 0 {
				content = fmt.Sprintf("Pong! 🏓\nLatency: `%dms`", latencyMs)
			} else {
				// If heartbeat hasn't triggered yet, just say Pong!
				content = "Pong! 🏓\nLatency: `Calculating...`"
			}

			// Update the deferred response
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
		},
	}
}
