package commands

import (
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"math/rand"

	"github.com/bwmarrin/discordgo"
)

// RPS creates the /rps command.
func RPS() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "rps",
			Description: "Play Rock, Paper, Scissors against the bot.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "choice",
					Description: "Your choice (Rock, Paper, or Scissors)",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Rock", Value: "rock"},
						{Name: "Paper", Value: "paper"},
						{Name: "Scissors", Value: "scissors"},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Get user choice
			options := i.ApplicationCommandData().Options
			userChoice := options[0].StringValue()

			// Bot choice
			choices := []string{"rock", "paper", "scissors"}

			botChoice := choices[rand.Intn(len(choices))]

			// Determine winner
			var result string
			var color int
			if userChoice == botChoice {
				result = "It's a tie!"
				color = 0xFFFF00 // Yellow
			} else if (userChoice == "rock" && botChoice == "scissors") ||
				(userChoice == "paper" && botChoice == "rock") ||
				(userChoice == "scissors" && botChoice == "paper") {
				result = "You win!"
				color = 0x00FF00 // Green
			} else {
				result = "Bot wins!"
				color = 0xFF0000 // Red
			}

			// Format emojis
			emojis := map[string]string{
				"rock":     "🪨",
				"paper":    "📄",
				"scissors": "✂️",
			}

			// Format embed
			embed := &discordgo.MessageEmbed{
				Title:       "Rock, Paper, Scissors",
				Description: fmt.Sprintf("**You chose:** %s %s\n**Bot chose:** %s %s\n\n**Result:** %s", cases.Title(language.English).String(userChoice), emojis[userChoice], cases.Title(language.English).String(botChoice), emojis[botChoice], result),
				Color:       color,
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Played by %s", i.Member.User.Username),
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
				SendError(s, i, "Failed to send rps response.")
			}
		},
	}
}
