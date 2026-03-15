package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// CustomCommand creates the /customcommand slash command.
func CustomCommand(database *db.DB) *Command {
	defaultPerms := int64(discordgo.PermissionManageServer)

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "customcommand",
			Description:              "Manage custom prefix commands",
			DefaultMemberPermissions: &defaultPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add or update a custom command",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The command trigger (e.g. 'hello')",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "response",
							Description: "The response for the command",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a custom command",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "The command name to remove",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all custom commands",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Database connection is not available.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subCmd := options[0]

			switch subCmd.Name {
			case "add":
				var name, response string
				for _, opt := range subCmd.Options {
					switch opt.Name {
					case "name":
						name = strings.ToLower(strings.TrimSpace(opt.StringValue()))
						name = strings.TrimPrefix(name, "!") // Remove prefix if provided by user
					case "response":
						response = opt.StringValue()
					}
				}

				if strings.ContainsAny(name, " \t\n\r") {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Command name cannot contain spaces.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				err := database.AddCustomCommand(context.Background(), i.GuildID, name, response)
				if err != nil {
					slog.Error("Failed to add custom command", "error", err, "guild_id", i.GuildID)
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to add custom command. Please try again later.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Custom command `!%s` has been saved successfully.", name),
					},
				})

			case "remove":
				name := strings.ToLower(strings.TrimSpace(subCmd.Options[0].StringValue()))
				name = strings.TrimPrefix(name, "!")

				err := database.RemoveCustomCommand(context.Background(), i.GuildID, name)
				if err != nil {
					slog.Error("Failed to remove custom command", "error", err, "guild_id", i.GuildID)
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to remove custom command. Please try again later.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("🗑️ Custom command `!%s` has been removed.", name),
					},
				})

			case "list":
				cmds, err := database.ListCustomCommands(context.Background(), i.GuildID)
				if err != nil {
					slog.Error("Failed to list custom commands", "error", err, "guild_id", i.GuildID)
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to list custom commands. Please try again later.",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}

				if len(cmds) == 0 {
					_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "There are no custom commands in this server yet.",
						},
					})
					return
				}

				var desc strings.Builder
				for _, c := range cmds {
					desc.WriteString(fmt.Sprintf("• `!%s` - %s\n", c.Name, c.Response))
				}

				embed := &discordgo.MessageEmbed{
					Title:       "📋 Custom Commands",
					Description: desc.String(),
					Color:       0x3498db, // Blue
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			}
		},
	}
}
