package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// AdvancedLog returns the /advancedlog command.
func AdvancedLog(database *db.DB) *Command {
	if database == nil {
		return nil
	}

	defaultMemberPermissions := int64(discordgo.PermissionManageGuild)

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "advancedlog",
			Description:              "Manage advanced event logging for this server.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set up advanced logging.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send advanced logs to.",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "events",
							Description: "Comma-separated events: channel_create, channel_delete, role_create, role_delete, all",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Type != discordgo.InteractionApplicationCommand {
				return
			}

			if i.GuildID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0].Name
			if subcommand == "setup" {
				var channelID string
				var events string

				for _, opt := range options[0].Options {
					if opt.Name == "channel" {
						channelID = opt.ChannelValue(nil).ID
					} else if opt.Name == "events" {
						events = opt.StringValue()
					}
				}

				// Basic validation
				events = strings.ToLower(strings.TrimSpace(events))

				err := database.SetAdvancedLogConfig(context.Background(), i.GuildID, events, channelID)
				if err != nil {
					slog.Error("Failed to set advanced log config", "guild_id", i.GuildID, "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to configure advanced logging.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Advanced logging configured! Events: `%s` will be sent to <#%s>.", events, channelID),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
		},
	}
}
