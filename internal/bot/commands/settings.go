package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Settings creates the /settings command.
func Settings(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "settings",
			Description: "Manage your personal user settings",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "dnd",
					Description: "Toggle Do Not Disturb mode to block DMs from the bot",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "enabled",
							Description: "Enable or disable DND mode",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "dm-notifications",
					Description: "Toggle direct message notifications",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "enabled",
							Description: "Enable or disable DM notifications",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Acknowledge interaction
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				return
			}

			userID := ""
			if i.Member != nil {
				userID = i.Member.User.ID
			} else if i.User != nil {
				userID = i.User.ID
			} else {
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subCommand := options[0]
			if len(subCommand.Options) == 0 {
				return
			}

			enabled := subCommand.Options[0].BoolValue()

			var dndMode *bool
			var dmNotifications *bool

			switch subCommand.Name {
			case "dnd":
				dndMode = &enabled
			case "dm-notifications":
				dmNotifications = &enabled
			}

			ctx := context.Background()
			err = database.SetUserConfig(ctx, userID, dndMode, dmNotifications)
			if err != nil {
				slog.Error("Failed to set user config", "error", err, "user_id", userID)
				msg := "❌ Failed to update settings."
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &msg,
				})
				return
			}

			msg := fmt.Sprintf("✅ Setting `%s` has been updated to `%v`.", subCommand.Name, enabled)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
		},
	}
}
