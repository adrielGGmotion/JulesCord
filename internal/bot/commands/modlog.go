package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// ModLog returns the command definition and handler for `/modlog`.
func ModLog(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "modlog",
			Description: "Set the moderation log channel for this server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "The channel to send moderation logs to",
					Required:    true,
				},
			},
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionAdministrator),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				respondWithError(s, i, "Database connection is not available.")
				return
			}

			if i.GuildID == "" {
				respondWithError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			channelOpt := options[0].ChannelValue(s)

			// Update the config in DB
			err := database.SetModLogChannel(context.Background(), i.GuildID, channelOpt.ID)
			if err != nil {
				log.Printf("Failed to set mod log channel for guild %s: %v", i.GuildID, err)
				respondWithError(s, i, "Failed to update moderation log channel configuration.")
				return
			}

			// Respond
			embed := &discordgo.MessageEmbed{
				Title:       "Configuration Updated",
				Description: fmt.Sprintf("Moderation logs will now be sent to <#%s>.", channelOpt.ID),
				Color:       0x00FF00, // Green
			}

			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
			if err != nil {
				log.Printf("Failed to respond to /modlog: %v", err)
			}
		},
	}
}

// respondWithError sends a red embed error message.
func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	embed := &discordgo.MessageEmbed{
		Title:       "Error",
		Description: message,
		Color:       0xFF0000, // Red
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
