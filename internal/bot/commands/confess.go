package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

func Confess(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:         "confess",
			Description:  "Anonymously post a confession to the configured channel",
			DMPermission: new(bool),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "The confession you want to make anonymously",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			// Defer response immediately to avoid timeout since we make DB/API calls
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				fmt.Println("Error deferring interaction in confess:", err)
				return
			}

			// Get the configured confession channel
			channelID, err := database.GetConfessionChannel(context.Background(), i.GuildID)
			if err != nil {
				editInteractionResponse(s, i, "❌ An error occurred while checking the confession channel config.")
				return
			}

			if channelID == "" {
				editInteractionResponse(s, i, "❌ This server has not set up a confession channel yet. Ask an admin to use `/confession setup`.")
				return
			}

			message := i.ApplicationCommandData().Options[0].StringValue()

			embed := &discordgo.MessageEmbed{
				Title:       "🤫 Anonymous Confession",
				Description: message,
				Color:       0x2F3136, // Dark Discord gray
			}

			_, err = s.ChannelMessageSendEmbed(channelID, embed)
			if err != nil {
				editInteractionResponse(s, i, "❌ Failed to post confession. Make sure I have permissions to send messages and embeds in the configured channel.")
				return
			}

			editInteractionResponse(s, i, "✅ Your confession has been posted anonymously!")
		},
	}
}

// editInteractionResponse is a helper to edit a deferred response
func editInteractionResponse(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		fmt.Println("Error editing interaction response:", err)
	}
}
