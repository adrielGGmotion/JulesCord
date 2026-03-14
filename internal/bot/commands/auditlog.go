package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// AuditLog returns the definition for the /auditlog command.
func AuditLog(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "auditlog",
			DMPermission: new(bool),
			Description: "Configure the server's audit log system",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the channel for audit log announcements (Admin only)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The channel to send audit logs",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database connection not available")
				return
			}

			// Check permissions
			if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
				SendError(s, i, "You need Administrator permissions to use this command.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]

			if subcommand.Name == "setup" {
				var channelID string
				for _, opt := range subcommand.Options {
					if opt.Name == "channel" {
						channelID = opt.ChannelValue(s).ID
					}
				}

				err := database.SetAuditLogChannel(context.Background(), i.GuildID, channelID)
				if err != nil {
					SendError(s, i, "Failed to configure audit log channel: "+err.Error())
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "📋 Audit Log Configured",
					Description: fmt.Sprintf("Audit logs will now be sent to <#%s>.", channelID),
					Color:       0x00FF00, // Green
				}
				SendEmbed(s, i, embed)
			}
		},
	}
}
