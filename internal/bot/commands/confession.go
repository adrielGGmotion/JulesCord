package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

func Confession(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "confession",
			Description:              "Configure the confession system",
			DMPermission:             new(bool),
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Set the channel for anonymous confessions",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to post confessions in",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
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

			subcommand := options[0].Name
			if subcommand == "setup" {
				channelID := options[0].Options[0].ChannelValue(s).ID

				err := database.SetConfessionChannel(context.Background(), i.GuildID, channelID)
				if err != nil {
					SendError(s, i, fmt.Sprintf("Failed to set confession channel: %v", err))
					return
				}

				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Confession channel set to <#%s>. Users can now use `/confess` to post anonymously.", channelID),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				if err != nil {
					fmt.Println("Error responding to confession setup:", err)
				}
			}
		},
	}
}
