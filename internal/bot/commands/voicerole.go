package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// VoiceRole returns the /voicerole command definition and handler.
func VoiceRole(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "voicerole",
			Description:              "Manage roles assigned when users join specific voice channels.",
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageRoles),
			DMPermission:             func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set a role to be assigned when users join a voice channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The voice channel",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildVoice,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to assign",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove the voice role configuration for a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The voice channel",
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
			if database == nil {
				SendError(s, i, "Database is not connected.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]
			ctx := context.Background()

			switch subcommand.Name {
			case "set":
				var channelID, roleID string
				for _, opt := range subcommand.Options {
					switch opt.Name {
					case "channel":
						channelID = opt.ChannelValue(nil).ID
					case "role":
						roleID = opt.RoleValue(nil, "").ID
					}
				}

				if channelID == "" || roleID == "" {
					SendError(s, i, "Missing required arguments.")
					return
				}

				err := database.SetVoiceRole(ctx, i.GuildID, channelID, roleID)
				if err != nil {
					slog.Error("Failed to set voice role", "guild", i.GuildID, "channel", channelID, "error", err)
					SendError(s, i, "An error occurred while saving the voice role configuration.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Users joining <#%s> will now be assigned the <@&%s> role.", channelID, roleID),
					},
				})

			case "remove":
				var channelID string
				for _, opt := range subcommand.Options {
					if opt.Name == "channel" {
						channelID = opt.ChannelValue(nil).ID
					}
				}

				if channelID == "" {
					SendError(s, i, "Missing channel argument.")
					return
				}

				err := database.RemoveVoiceRole(ctx, i.GuildID, channelID)
				if err != nil {
					slog.Error("Failed to remove voice role", "guild", i.GuildID, "channel", channelID, "error", err)
					SendError(s, i, "An error occurred while removing the voice role configuration.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Removed voice role configuration for <#%s>.", channelID),
					},
				})
			}
		},
	}
}
