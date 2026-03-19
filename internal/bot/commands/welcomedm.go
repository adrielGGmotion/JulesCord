package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// WelcomeDM returns a Command for configuring welcome DMs.
func WelcomeDM(database *db.DB) *Command {
	perm := int64(discordgo.PermissionManageServer)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "welcomedm",
			Description:              "Configure a welcome DM sent to new users when they join.",
			DefaultMemberPermissions: &perm,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set the welcome DM message (use {user} to mention the user).",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message",
							Description: "The welcome DM message.",
							Required:    true,
						},
					},
				},
				{
					Name:        "enable",
					Description: "Enable sending welcome DMs.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "disable",
					Description: "Disable sending welcome DMs.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0]

			switch subcommand.Name {
			case "set":
				message := subcommand.Options[0].StringValue()

				// Basic validation
				if strings.TrimSpace(message) == "" {
					SendError(s, i, "The welcome DM message cannot be empty.")
					return
				}

				err := database.SetWelcomeDM(context.Background(), i.GuildID, message)
				if err != nil {
					SendError(s, i, "Failed to set the welcome DM message.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Welcome DM message set and enabled successfully:\n\n%s", message),
					},
				})

			case "enable":
				err := database.ToggleWelcomeDM(context.Background(), i.GuildID, true)
				if err != nil {
					SendError(s, i, "Failed to enable the welcome DM.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "✅ Welcome DM has been enabled.",
					},
				})

			case "disable":
				err := database.ToggleWelcomeDM(context.Background(), i.GuildID, false)
				if err != nil {
					SendError(s, i, "Failed to disable the welcome DM.")
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "✅ Welcome DM has been disabled.",
					},
				})
			}
		},
	}
}
