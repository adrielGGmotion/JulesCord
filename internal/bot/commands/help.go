package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Help creates the /help command.
func Help(registry *Registry) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "help",
			Description: "Lists all available commands",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var sb strings.Builder
			for _, cmd := range registry.Commands {
				sb.WriteString(fmt.Sprintf("**`/%s`** - %s\n", cmd.Definition.Name, cmd.Definition.Description))
			}

			embed := &discordgo.MessageEmbed{
				Title:       "JulesCord Help Menu 📚",
				Description: "Here is a list of all commands I support:\n\n" + sb.String(),
				Color:       0xfcba03, // Yellow
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Total Commands: %d", len(registry.Commands)),
				},
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
		},
	}
}
