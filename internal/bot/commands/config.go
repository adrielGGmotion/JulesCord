package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// Config returns the /config command definition and handler.
func Config(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionAdministrator)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "config",
			Description:              "Configure server settings.",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set-log-channel",
					Description: "Sets the channel where moderation actions will be logged.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send moderation logs to",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set-welcome-channel",
					Description: "Sets the channel where welcome messages will be sent.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send welcome messages to",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set-mod-role",
					Description: "Sets the moderator role.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to set as moderator",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "set-auto-role",
					Description: "Sets the role to automatically assign to new members.",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to auto-assign",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "view",
					Description: "View the current server configuration.",
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "This command can only be used in a server.",
					},
				})
				return
			}

			if database == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database is not connected. Cannot configure settings.",
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]

			switch subCommand.Name {
			case "set-log-channel":
				var targetChannel *discordgo.Channel
				for _, option := range subCommand.Options {
					if option.Name == "channel" {
						targetChannel = option.ChannelValue(s)
					}
				}

				if targetChannel == nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Could not find the specified channel.",
						},
					})
					return
				}

				err := database.SetGuildLogChannel(context.Background(), i.GuildID, targetChannel.ID)
				if err != nil {
					log.Printf("Failed to set log channel for guild %s: %v", i.GuildID, err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while saving the configuration.",
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Configuration Updated",
					Description: fmt.Sprintf("Moderation log channel has been set to <#%s>.", targetChannel.ID),
					Color:       0x00FF00, // Green
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})

			case "set-welcome-channel":
				var targetChannel *discordgo.Channel
				for _, option := range subCommand.Options {
					if option.Name == "channel" {
						targetChannel = option.ChannelValue(s)
					}
				}

				if targetChannel == nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Could not find the specified channel.",
						},
					})
					return
				}

				err := database.SetGuildWelcomeChannel(context.Background(), i.GuildID, targetChannel.ID)
				if err != nil {
					log.Printf("Failed to set welcome channel for guild %s: %v", i.GuildID, err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while saving the configuration.",
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Configuration Updated",
					Description: fmt.Sprintf("Welcome channel has been set to <#%s>.", targetChannel.ID),
					Color:       0x00FF00, // Green
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			case "set-mod-role":
				var targetRole *discordgo.Role
				for _, option := range subCommand.Options {
					if option.Name == "role" {
						targetRole = option.RoleValue(s, i.GuildID)
					}
				}

				if targetRole == nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Could not find the specified role.",
						},
					})
					return
				}

				err := database.SetGuildModRole(context.Background(), i.GuildID, targetRole.ID)
				if err != nil {
					log.Printf("Failed to set mod role for guild %s: %v", i.GuildID, err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while saving the configuration.",
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Configuration Updated",
					Description: fmt.Sprintf("Moderator role has been set to <@&%s>.", targetRole.ID),
					Color:       0x00FF00, // Green
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			case "set-auto-role":
				var targetRole *discordgo.Role
				for _, option := range subCommand.Options {
					if option.Name == "role" {
						targetRole = option.RoleValue(s, i.GuildID)
					}
				}

				if targetRole == nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Could not find the specified role.",
						},
					})
					return
				}

				err := database.SetGuildAutoRole(context.Background(), i.GuildID, targetRole.ID)
				if err != nil {
					log.Printf("Failed to set auto role for guild %s: %v", i.GuildID, err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while saving the configuration.",
						},
					})
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Configuration Updated",
					Description: fmt.Sprintf("Auto-role has been set to <@&%s>.", targetRole.ID),
					Color:       0x00FF00, // Green
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			case "view":
				config, err := database.GetGuildConfig(context.Background(), i.GuildID)
				if err != nil {
					log.Printf("Failed to get config for guild %s: %v", i.GuildID, err)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while retrieving the configuration.",
						},
					})
					return
				}

				desc := ""
				if config.LogChannelID != nil {
					desc += fmt.Sprintf("**Log Channel:** <#%s>\n", *config.LogChannelID)
				} else {
					desc += "**Log Channel:** Not set\n"
				}

				if config.WelcomeChannelID != nil {
					desc += fmt.Sprintf("**Welcome Channel:** <#%s>\n", *config.WelcomeChannelID)
				} else {
					desc += "**Welcome Channel:** Not set\n"
				}

				if config.ModRoleID != nil {
					desc += fmt.Sprintf("**Mod Role:** <@&%s>\n", *config.ModRoleID)
				} else {
					desc += "**Mod Role:** Not set\n"
				}

				if config.AutoRoleID != nil {
					desc += fmt.Sprintf("**Auto Role:** <@&%s>\n", *config.AutoRoleID)
				} else {
					desc += "**Auto Role:** Not set\n"
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Server Configuration",
					Description: desc,
					Color:       0x3498DB, // Blue
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
