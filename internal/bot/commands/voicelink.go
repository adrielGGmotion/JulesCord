package commands

import (
	"context"
	"fmt"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// VoiceLink returns the command definition for /voicelink
func VoiceLink(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageChannels)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "voicelink",
			Description:              "Link a voice channel to a text channel.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			DMPermission:             func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Link a voice channel to a text channel.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "voice_channel",
							Description: "The voice channel to link.",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildVoice,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "text_channel",
							Description: "The text channel to link to the voice channel.",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a linked text channel from a voice channel.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "voice_channel",
							Description: "The voice channel to unlink.",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildVoice,
							},
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name

			switch subcommand {
			case "setup":
				subOptions := options[0].Options
				voiceChannel := subOptions[0].ChannelValue(s)
				textChannel := subOptions[1].ChannelValue(s)

				err := database.SetVoiceLink(context.Background(), i.GuildID, voiceChannel.ID, textChannel.ID)
				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to setup voice link in the database.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully linked voice channel <#%s> to text channel <#%s>.", voiceChannel.ID, textChannel.ID),
					},
				})

			case "remove":
				subOptions := options[0].Options
				voiceChannel := subOptions[0].ChannelValue(s)

				err := database.RemoveVoiceLink(context.Background(), i.GuildID, voiceChannel.ID)
				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to remove voice link from the database.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully unlinked text channel from voice channel <#%s>.", voiceChannel.ID),
					},
				})
			}
		},
	}
}
