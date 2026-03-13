package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Config returns the /config command definition and handler.
func Config(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionAdministrator)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "config",
			Description:              "Configure server settings.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set-log-channel",
					Description: "Sets the channel where moderation actions will be logged.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send moderation logs to",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			if database == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not connected. Cannot configure settings.",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			switch subCommand.Name {
			case "set-log-channel":
				var targetChannel *discordgo.Channel
				for _, option := range subCommand.Options {
					if option.Name == "channel" {
						targetChannel = option.ChannelValue(s)
					}
				}

				if targetChannel == nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Could not find the specified channel.",
						},
					})
					return
				}

				err := database.SetGuildLogChannel(context.Background(), i.GuildID, targetChannel.ID)
				if err != nil {
					log.Printf("Failed to set log channel for guild %s: %v", i.GuildID, err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while saving the configuration.",
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Configuration Updated",
					Description: fmt.Sprintf("Moderation log channel has been set to <#%s>.", targetChannel.ID),
					Color:       0x00FF00, // Green
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
