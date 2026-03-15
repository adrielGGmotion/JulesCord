package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// AutoThread creates the /autothread command.
func AutoThread(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "autothread",
			Description:              "Manage auto-threads for channels",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageChannels); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Configure a channel for auto-threads",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to enable auto-threads for",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "template",
							Description: "Template for thread names (e.g., 'Thread for {user}' or '{title}')",
							Required:    false,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove auto-threads from a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to disable auto-threads for",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all auto-thread channels",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name
			guildID := i.GuildID

			switch subcommand {
			case "add":
				channelOpt := options[0].Options[0].ChannelValue(s)
				if channelOpt.Type != discordgo.ChannelTypeGuildText && channelOpt.Type != discordgo.ChannelTypeGuildNews {
					SendError(s, i, "Please select a valid text channel.")
					return
				}

				template := "Thread"
				if len(options[0].Options) > 1 {
					template = options[0].Options[1].StringValue()
				}

				err := database.AddAutoThreadChannel(context.Background(), guildID, channelOpt.ID, template)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to add auto-thread channel: %v", err))
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Enabled auto-threads for <#%s> with template `%s`.", channelOpt.ID, template),
					},
				})

			case "remove":
				channelOpt := options[0].Options[0].ChannelValue(s)
				err := database.RemoveAutoThreadChannel(context.Background(), guildID, channelOpt.ID)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to remove auto-thread channel: %v", err))
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Disabled auto-threads for <#%s>.", channelOpt.ID),
					},
				})

			case "list":
				configs, err := database.GetAutoThreadChannels(context.Background(), guildID)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to fetch auto-thread channels: %v", err))
					return
				}

				if len(configs) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "No auto-thread channels configured.",
						},
					})
					return
				}

				var lines []string
				for _, config := range configs {
					lines = append(lines, fmt.Sprintf("• <#%s> - Template: `%s`", config.ChannelID, config.ThreadNameTemplate))
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Auto-Thread Channels",
					Description: strings.Join(lines, "\n"),
					Color:       0x00FF00,
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
