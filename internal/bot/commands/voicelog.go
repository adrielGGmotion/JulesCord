package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// VoiceLog creates the /voicelog command structure.
func VoiceLog(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "voicelog",
			Description:              "Manage the voice logging system",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageGuild); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "setup",
					Description: "Configure the channel for voice logs",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send voice logs to",
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
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			if database == nil {
				SendError(s, i, "Database is not connected. Cannot configure settings.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			switch subCommand.Name {
			case "setup":
				var targetChannel *discordgo.Channel
				for _, option := range subCommand.Options {
					if option.Name == "channel" {
						targetChannel = option.ChannelValue(s)
					}
				}

				if targetChannel == nil {
					SendError(s, i, "Could not find the specified channel.")
					return
				}

				err := database.SetVoiceLogChannel(context.Background(), i.GuildID, targetChannel.ID)
				if err != nil {
					slog.Error("Failed to set voice log channel", "guild_id", i.GuildID, "error", err)
					SendError(s, i, "An error occurred while saving the configuration.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Voice Logging Setup Complete",
					Description: fmt.Sprintf("Voice logs will now be sent to <#%s>.", targetChannel.ID),
					Color:       0x00FF00, // Green
				}

				SendEmbed(s, i, embed)
			}
		},
	}
}
