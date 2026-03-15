package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// MediaChannel returns the command definition for the /mediachannel command.
func MediaChannel(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "mediachannel",
			Description:              "Manage media-only channels (Admin only)",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a media-only channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to mark as media-only",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a media-only channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to remove from media-only",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "list",
					Description: "List all media-only channels",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is not available.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "add":
				channelID := subcommand.Options[0].ChannelValue(s).ID
				err := database.AddMediaChannel(context.Background(), i.GuildID, channelID)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to add media channel: %v", err))
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Media Channel Added",
					Description: fmt.Sprintf("<#%s> is now a media-only channel. Messages without attachments or URLs will be deleted.", channelID),
					Color:       0x00FF00,
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			case "remove":
				channelID := subcommand.Options[0].ChannelValue(s).ID
				err := database.RemoveMediaChannel(context.Background(), i.GuildID, channelID)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to remove media channel: %v", err))
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Media Channel Removed",
					Description: fmt.Sprintf("<#%s> is no longer a media-only channel.", channelID),
					Color:       0x00FF00,
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			case "list":
				channels, err := database.ListMediaChannels(context.Background(), i.GuildID)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to retrieve media channels: %v", err))
					return
				}

				var descBuilder strings.Builder
				if len(channels) == 0 {
					descBuilder.WriteString("No media-only channels configured.")
				} else {
					for _, chID := range channels {
						descBuilder.WriteString(fmt.Sprintf("• <#%s>\n", chID))
					}
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Media-Only Channels",
					Description: descBuilder.String(),
					Color:       0x3498DB,
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
