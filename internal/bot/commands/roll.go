package commands

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Roll creates the /roll command.
func Roll() *Command {
	sidesMin := float64(2)
	countMin := float64(1)
	countMax := float64(10)

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "roll",
			Description: "Roll virtual dice.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "sides",
					Description: "Number of sides on the dice (default: 6)",
					Required:    false,
					MinValue:    &sidesMin,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "count",
					Description: "Number of dice to roll (default: 1, max: 10)",
					Required:    false,
					MinValue:    &countMin,
					MaxValue:    countMax,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			sides := 6
			count := 1

			// Parse options
			options := i.ApplicationCommandData().Options
			for _, opt := range options {
				switch opt.Name {
				case "sides":
					sides = int(opt.IntValue())
				case "count":
					count = int(opt.IntValue())
				}
			}

			var rolls []int
			total := 0
			for j := 0; j < count; j++ {
				roll := rand.Intn(sides) + 1
				rolls = append(rolls, roll)
				total += roll
			}

			// Format response
			var rollsStr []string
			for _, r := range rolls {
				rollsStr = append(rollsStr, fmt.Sprintf("`%d`", r))
			}
			rollsFormatted := strings.Join(rollsStr, ", ")

			embed := &discordgo.MessageEmbed{
				Title:       "🎲 Dice Roll",
				Description: fmt.Sprintf("**Rolled %dd%d:**\n%s\n\n**Total:** %d", count, sides, rollsFormatted, total),
				Color:       0x00FF00, // Green color
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Rolled by %s", i.Member.User.Username),
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
				SendError(s, i, "Failed to send roll response.")
			}
		},
	}
}
