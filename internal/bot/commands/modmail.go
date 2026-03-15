package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

func Modmail(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "modmail",
			Description:              "Configure the modmail system",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionManageServer); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the category or channel where new modmail threads will be created",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The text channel or category to use for modmail",
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

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}

			subcommand := options[0]
			if subcommand.Name == "setup" {
				channelID := subcommand.Options[0].ChannelValue(s).ID

				err := database.SetModmailChannel(context.Background(), i.GuildID, channelID)
				if err != nil {
					slog.Error("Failed to set modmail channel", "error", err)
					SendError(s, i, "Failed to configure modmail system.")
					return
				}

				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Modmail configured to use <#%s> for incoming messages.", channelID),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
		},
	}
}
