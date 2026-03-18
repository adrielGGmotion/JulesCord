package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// AutoPublish returns the /autopublish command definition and handler.
func AutoPublish(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "autopublish",
			Description: "Configure channels for automatic message crossposting",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a channel to auto-publish configuration",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to auto-publish in (must be an announcement channel)",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildNews,
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a channel from auto-publish configuration",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to remove",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildNews,
							},
						},
					},
				},
			},
			DefaultMemberPermissions: func() *int64 {
				perm := int64(discordgo.PermissionManageChannels)
				return &perm
			}(),
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

			if database == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not connected. Cannot configure auto-publish.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0].Name
			subOptions := options[0].Options

			var targetChannelID string
			for _, opt := range subOptions {
				if opt.Name == "channel" {
					targetChannelID = opt.Value.(string)
				}
			}

			if targetChannelID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Please provide a valid channel.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			switch subCommand {
			case "add":
				err := database.AddAutoPublishChannel(context.Background(), i.GuildID, targetChannelID)
				if err != nil {
					slog.Error("Error adding auto-publish channel", "guild_id", i.GuildID, "channel_id", targetChannelID, "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to add channel to auto-publish configuration.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully configured <#%s> for auto-publishing messages.", targetChannelID),
					},
				})

			case "remove":
				err := database.RemoveAutoPublishChannel(context.Background(), i.GuildID, targetChannelID)
				if err != nil {
					slog.Error("Error removing auto-publish channel", "guild_id", i.GuildID, "channel_id", targetChannelID, "error", err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to remove channel from auto-publish configuration.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Successfully removed <#%s> from auto-publish configuration.", targetChannelID),
					},
				})
			}
		},
	}
}
