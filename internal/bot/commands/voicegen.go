package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// IsGeneratedChannelOwnerChecker is an interface for checking if a user owns a generated voice channel.
type IsGeneratedChannelOwnerChecker interface {
	IsGeneratedChannelOwner(channelID, userID string) bool
}

// VoiceGen returns the definition for the /voicegen command.
func VoiceGen(database *db.DB, ownerChecker IsGeneratedChannelOwnerChecker) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:         "voicegen",
			DMPermission: new(bool),
			Description:  "Manage voice generator configuration",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Configure the voice generator (Admin only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:         "base_channel",
							Description:  "The base voice channel users join to generate a new channel",
							Type:         discordgo.ApplicationCommandOptionChannel,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildVoice},
							Required:     true,
						},
						{
							Name:        "max_channels",
							Description: "The maximum number of generated channels to allow",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    new(float64),
						},
						{
							Name:        "allow_custom_names",
							Description: "Allow users to rename their generated voice channels",
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Required:    true,
						},
						{
							Name:        "default_name_template",
							Description: "Template for generated channel names. Use {user} for username. Default: {user}'s Channel",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "name",
					Description: "Rename your generated voice channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The new name for your voice channel",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			if subcommand.Name == "setup" {
				// Check permissions
				if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
					SendError(s, i, "You need Administrator permissions to use this command.")
					return
				}

				var baseChannelID string
				var maxChannels int
				var allowCustomNames bool
				var defaultNameTemplate *string

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "base_channel":
						baseChannelID = opt.ChannelValue(s).ID
					case "max_channels":
						maxChannels = int(opt.IntValue())
					case "allow_custom_names":
						allowCustomNames = opt.BoolValue()
					case "default_name_template":
						val := opt.StringValue()
						defaultNameTemplate = &val
					}
				}

				if maxChannels < 1 {
					SendError(s, i, "Maximum channels must be at least 1.")
					return
				}

				err := database.SetVoiceGeneratorConfig(context.Background(), i.GuildID, baseChannelID, maxChannels, allowCustomNames, defaultNameTemplate)
				if err != nil {
					SendError(s, i, "Failed to configure voice generator: "+err.Error())
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "🔊 Voice Generator Configured",
					Description: fmt.Sprintf("Users joining <#%s> will now have a new voice channel generated. Maximum generated channels: %d.\nCustom Names: %v", baseChannelID, maxChannels, allowCustomNames),
					Color:       0x00FF00,
				}
				SendEmbed(s, i, embed)
			} else if subcommand.Name == "name" {
				var newName string
				for _, opt := range subcommand.Options {
					if opt.Name == "name" {
						newName = opt.StringValue()
					}
				}

				voiceState, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
				if err != nil || voiceState.ChannelID == "" {
					SendError(s, i, "You are not in a voice channel.")
					return
				}

				if ownerChecker == nil || !ownerChecker.IsGeneratedChannelOwner(voiceState.ChannelID, i.Member.User.ID) {
					SendError(s, i, "You do not own this voice channel.")
					return
				}

				config, err := database.GetVoiceGeneratorConfig(context.Background(), i.GuildID)
				if err != nil || config == nil || !config.AllowCustomNames {
					SendError(s, i, "Custom names are not enabled for this server.")
					return
				}

				channelEdit := &discordgo.ChannelEdit{
					Name: newName,
				}
				_, err = s.ChannelEdit(voiceState.ChannelID, channelEdit)
				if err != nil {
					SendError(s, i, "Failed to rename channel.")
					return
				}

				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Channel renamed to " + newName,
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				if err != nil {
					fmt.Println("Error responding to interaction:", err)
				}
			}
		},
	}
}
