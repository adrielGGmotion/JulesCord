package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// Verification returns the command definition for the verification system.
func Verification(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "verify",
			Description:              "Verification system commands",
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionAdministrator),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Setup the verification system in this channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to assign upon successful verification",
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

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				SendError(s, i, "Invalid subcommand.")
				return
			}

			subcommand := options[0]

			switch subcommand.Name {
			case "setup":
				handleVerifySetup(s, i, database, subcommand.Options)
			default:
				SendError(s, i, "Unknown subcommand.")
			}
		},
	}
}

func handleVerifySetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if database == nil {
		SendError(s, i, "Database connection not available.")
		return
	}

	var roleID string
	for _, opt := range options {
		switch opt.Name {
		case "role":
			roleID = opt.RoleValue(s, i.GuildID).ID
		}
	}

	if roleID == "" {
		SendError(s, i, "Role is required.")
		return
	}

	err := database.SetVerificationConfig(context.Background(), i.GuildID, roleID)
	if err != nil {
		slog.Error("Failed to set verification config", "error", err, "guild_id", i.GuildID)
		SendError(s, i, "Failed to save verification configuration.")
		return
	}

	// Send the verification panel to the channel
	embed := &discordgo.MessageEmbed{
		Title:       "Server Verification",
		Description: "Please click the button below to verify your account and gain access to the rest of the server.",
		Color:       0x00FF00, // Green
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Verify",
					Style:    discordgo.SuccessButton,
					CustomID: "verify_button",
				},
			},
		},
	}

	_, err = s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})

	if err != nil {
		slog.Error("Failed to send verification panel", "error", err, "channel_id", i.ChannelID)
		SendError(s, i, "Verification configuration saved, but failed to send the verification panel.")
		return
	}

	SendEmbed(s, i, &discordgo.MessageEmbed{
		Title:       "Verification System",
		Description: fmt.Sprintf("Verification system setup successful. Users will receive the <@&%s> role upon verification.", roleID),
		Color:       0x00FF00,
	})
}
