package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// JoinLeaveLog creates the /joinleavelog command structure.
func JoinLeaveLog(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "joinleavelog",
			Description:              "Manage the join/leave logging system",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageGuild); return &p }(),
			DMPermission:             func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "setup",
					Description: "Configure the channel for join/leave logs",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send join/leave logs to",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
								discordgo.ChannelTypeGuildNews,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "log_joins",
							Description: "Enable logging of member joins (default: true)",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "log_leaves",
							Description: "Enable logging of member leaves (default: true)",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Disable the join/leave logging system",
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is not connected. Cannot manage settings.")
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
				logJoins := true
				logLeaves := true

				for _, option := range subCommand.Options {
					switch option.Name {
					case "channel":
						targetChannel = option.ChannelValue(s)
					case "log_joins":
						logJoins = option.BoolValue()
					case "log_leaves":
						logLeaves = option.BoolValue()
					}
				}

				if targetChannel == nil {
					SendError(s, i, "Could not find the specified channel.")
					return
				}

				err := database.SetJoinLeaveLog(context.Background(), i.GuildID, targetChannel.ID, logJoins, logLeaves)
				if err != nil {
					slog.Error("Failed to set join/leave log channel", "guild_id", i.GuildID, "error", err)
					SendError(s, i, "An error occurred while saving the configuration.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Join/Leave Logging Setup Complete",
					Description: fmt.Sprintf("Join and leave logs will now be sent to <#%s>.\n**Log Joins:** %v\n**Log Leaves:** %v", targetChannel.ID, logJoins, logLeaves),
					Color:       0x00FF00, // Green
				}

				SendEmbed(s, i, embed)

			case "remove":
				err := database.RemoveJoinLeaveLog(context.Background(), i.GuildID)
				if err != nil {
					slog.Error("Failed to remove join/leave log configuration", "guild_id", i.GuildID, "error", err)
					SendError(s, i, "An error occurred while removing the configuration.")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Join/Leave Logging Disabled",
					Description: "Join and leave logs will no longer be sent.",
					Color:       0xFF0000, // Red
				}

				SendEmbed(s, i, embed)
			}
		},
	}
}
