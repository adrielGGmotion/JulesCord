package commands

import (
	"context"
	"fmt"
	"log/slog"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// ServerLog creates the serverlog command
func ServerLog(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "serverlog",
			Description:              "Configure the server log channel for tracking deleted/edited messages",
			DefaultMemberPermissions: func(i int64) *int64 { return &i }(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the channel for server logs",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send server logs to",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if database == nil {
				SendError(s, i, "Database is not configured.")
				return
			}

			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			subcommand := i.ApplicationCommandData().Options[0]
			switch subcommand.Name {
			case "setup":
				handleServerLogSetup(s, i, database, subcommand.Options)
			}
		},
	}
}

func handleServerLogSetup(s *discordgo.Session, i *discordgo.InteractionCreate, database *db.DB, options []*discordgo.ApplicationCommandInteractionDataOption) {
	channelID := options[0].ChannelValue(s).ID

	err := database.SetServerLogChannel(context.Background(), i.GuildID, channelID)
	if err != nil {
		slog.Error("Failed to set server log channel", "error", err)
		SendError(s, i, "Failed to update server log configuration in database.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Server Logs Configured",
		Description: fmt.Sprintf("✅ Server log messages (edited and deleted) will now be sent to <#%s>.", channelID),
		Color:       0x00FF00,
	}

	SendEmbed(s, i, embed)
}
