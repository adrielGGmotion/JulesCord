package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// VoiceGen returns the definition for the /voicegen command.
func VoiceGen(database *db.DB) *Command {
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
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			// Check permissions
			if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
				SendError(s, i, "You need Administrator permissions to use this command.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			if subcommand.Name == "setup" {
				var baseChannelID string
				var maxChannels int

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "base_channel":
						baseChannelID = opt.ChannelValue(s).ID
					case "max_channels":
						maxChannels = int(opt.IntValue())
					}
				}

				if maxChannels < 1 {
					SendError(s, i, "Maximum channels must be at least 1.")
					return
				}

				err := database.SetVoiceGeneratorConfig(context.Background(), i.GuildID, baseChannelID, maxChannels)
				if err != nil {
					SendError(s, i, "Failed to configure voice generator: "+err.Error())
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "🔊 Voice Generator Configured",
					Description: fmt.Sprintf("Users joining <#%s> will now have a new voice channel generated. Maximum generated channels: %d.", baseChannelID, maxChannels),
					Color:       0x00FF00,
				}
				SendEmbed(s, i, embed)
			}
		},
	}
}
