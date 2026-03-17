package commands

import (
	"fmt"
	"math/rand"

	"github.com/bwmarrin/discordgo"
)

var eightBallResponses = []string{
	// Affirmative
	"It is certain.",
	"It is decidedly so.",
	"Without a doubt.",
	"Yes - definitely.",
	"You may rely on it.",
	"As I see it, yes.",
	"Most likely.",
	"Outlook good.",
	"Yes.",
	"Signs point to yes.",
	// Non-committal
	"Reply hazy, try again.",
	"Ask again later.",
	"Better not tell you now.",
	"Cannot predict now.",
	"Concentrate and ask again.",
	// Negative
	"Don't count on it.",
	"My reply is no.",
	"My sources say no.",
	"Outlook not so good.",
	"Very doubtful.",
}

// EightBall creates the /8ball command.
func EightBall() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "8ball",
			Description: "Ask the Magic 8-Ball a yes/no question.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "question",
					Description: "The question you want to ask",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Get the question
			options := i.ApplicationCommandData().Options
			question := options[0].StringValue()

			// Select a random response

			response := eightBallResponses[rand.Intn(len(eightBallResponses))]

			// Create the embed
			embed := &discordgo.MessageEmbed{
				Title:       "🎱 Magic 8-Ball",
				Description: fmt.Sprintf("**Question:** %s\n**Answer:** %s", question, response),
				Color:       0x000000, // Black color
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Asked by %s", i.Member.User.Username),
				},
			}

			// Respond
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
			if err != nil {
				SendError(s, i, "Failed to send 8-ball response.")
			}
		},
	}
}
