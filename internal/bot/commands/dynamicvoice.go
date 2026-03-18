package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// DynamicVoice returns the definition for the /dynamicvoice command.
func DynamicVoice(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:         "dynamicvoice",
			DMPermission: new(bool),
			Description:  "Manage dynamic voice channel configuration",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Configure dynamic voice channels (Admin only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:         "category",
							Description:  "The category to create dynamic channels in",
							Type:         discordgo.ApplicationCommandOptionChannel,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildCategory},
							Required:     true,
						},
						{
							Name:         "trigger_channel",
							Description:  "The voice channel users join to trigger creation",
							Type:         discordgo.ApplicationCommandOptionChannel,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildVoice},
							Required:     true,
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
				var categoryID, triggerChannelID string

				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "category":
						categoryID = opt.ChannelValue(s).ID
					case "trigger_channel":
						triggerChannelID = opt.ChannelValue(s).ID
					}
				}

				err := database.SetDynamicVoiceConfig(context.Background(), i.GuildID, categoryID, triggerChannelID)
				if err != nil {
					SendError(s, i, "Failed to configure dynamic voice channels: "+err.Error())
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "🎤 Dynamic Voice Configured",
					Description: fmt.Sprintf("Users joining <#%s> will now have a dynamic voice channel created for them in the selected category.", triggerChannelID),
					Color:       0x00FF00,
				}
				SendEmbed(s, i, embed)
			}
		},
	}
}
