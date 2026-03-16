package commands

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func AutoThread(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "autothread",
			Description:              "Configure auto-threads for a channel",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Setup auto-threads in a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The channel to setup auto-threads in",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
						{
							Name:        "template",
							Description: "The thread name template (use {user} for author name)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove auto-threads from a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The channel to remove auto-threads from",
							Type:        discordgo.ApplicationCommandOptionChannel,
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
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})

			options := i.ApplicationCommandData().Options
			subcommand := options[0]

			switch subcommand.Name {
			case "setup":
				channelID := subcommand.Options[0].ChannelValue(s).ID
				template := subcommand.Options[1].StringValue()

				err := database.SetAutoThread(context.Background(), i.GuildID, channelID, template)
				if err != nil {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Embeds: &[]*discordgo.MessageEmbed{
							{
								Title:       "Error",
								Description: "Failed to setup auto-thread.",
								Color:       0xff0000,
							},
						},
					})
					return
				}

				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{
						{
							Title:       "Auto-Thread Setup",
							Description: fmt.Sprintf("Successfully setup auto-threads in <#%s> with template: `%s`", channelID, template),
							Color:       0x00ff00,
						},
					},
				})

			case "remove":
				channelID := subcommand.Options[0].ChannelValue(s).ID

				err := database.RemoveAutoThread(context.Background(), i.GuildID, channelID)
				if err != nil {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Embeds: &[]*discordgo.MessageEmbed{
							{
								Title:       "Error",
								Description: "Failed to remove auto-thread.",
								Color:       0xff0000,
							},
						},
					})
					return
				}

				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{
						{
							Title:       "Auto-Thread Removed",
							Description: fmt.Sprintf("Successfully removed auto-threads from <#%s>.", channelID),
							Color:       0x00ff00,
						},
					},
				})
			}
		},
	}
}
