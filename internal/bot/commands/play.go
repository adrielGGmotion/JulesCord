package commands

import (
	"github.com/bwmarrin/discordgo"
)

// Play returns the play slash command.
func Play() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "play",
			Description: "Play a song from YouTube/Spotify or a query",
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
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Coming soon 🎵",
				},
			})
		},
	}
}
