package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// AutoThread creates the /autothread command
func AutoThread(database *db.DB) *Command {
	adminPerms := int64(discordgo.PermissionManageChannels)

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "autothread",
			Description: "Configure auto-threads for channels",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Enable auto-threads for a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to enable auto-threads in",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
								discordgo.ChannelTypeGuildNews,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "template",
							Description: "Template for thread names (use {user} and {content})",
							Required:    true,
						},
					},
				},
				{
					Name:        "disable",
					Description: "Disable auto-threads for a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to disable auto-threads for",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
								discordgo.ChannelTypeGuildNews,
							},
						},
					},
				},
			},
			DefaultMemberPermissions: &adminPerms,
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member == nil {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			if database == nil {
				SendError(s, i, "Database connection not available.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]
			switch subcommand.Name {
			case "setup":
				handleAutoThreadSetup(s, i, database, subcommand.Options)
			case "disable":
				handleAutoThreadDisable(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleAutoThreadSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var channelID, template string
	for _, opt := range options {
		switch opt.Name {
		case "channel":
			channelID = opt.ChannelValue(nil).ID
		case "template":
			template = opt.StringValue()
		}
	}

	if strings.TrimSpace(template) == "" {
		SendError(s, i, "Template cannot be empty.")
		return
	}

	err := database.SetAutoThreadConfig(context.Background(), i.GuildID, channelID, template)
	if err != nil {
		SendError(s, i, fmt.Sprintf("Failed to save configuration: %v", err))
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title: "✅ Auto-Thread Setup",
		Description: fmt.Sprintf("Enabled auto-threads in <#%s> with template: `%s`", channelID, template),
		Color: 0x00FF00,
	})
}

func handleAutoThreadDisable(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var channelID string
	for _, opt := range options {
		if opt.Name == "channel" {
			channelID = opt.ChannelValue(nil).ID
		}
	}

	err := database.RemoveAutoThreadConfig(context.Background(), channelID)
	if err != nil {
		SendError(s, i, fmt.Sprintf("Failed to remove configuration: %v", err))
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title: "✅ Auto-Thread Disabled",
		Description: fmt.Sprintf("Disabled auto-threads for <#%s>.", channelID),
		Color: 0x00FF00,
	})
}
