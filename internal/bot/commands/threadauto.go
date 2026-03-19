package commands

import (
	"context"
	"fmt"
	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// ThreadAuto returns the /threadauto command
func ThreadAuto(database *db.DB) *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "threadauto",
			Description:              "Configure thread automation settings",
			DefaultMemberPermissions: func(p int64) *int64 { return &p }(discordgo.PermissionManageChannels),
			DMPermission:             func(b bool) *bool { return &b }(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: "Setup auto-join for threads created in a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to monitor for new threads",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "auto_join",
							Description: "Automatically join threads created in this channel",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove thread automation config for a channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to remove config from",
							Required:    true,
						},
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			subcommand := options[0].Name
			ctx := context.Background()

			if subcommand == "setup" {
				channelOpt := options[0].Options[0].ChannelValue(s)
				autoJoinOpt := options[0].Options[1].BoolValue()

				err := database.SetThreadAutomation(ctx, i.GuildID, channelOpt.ID, autoJoinOpt)
				if err != nil {
					SendError(s, i, "Failed to save thread automation config")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Thread Automation Configured",
					Description: fmt.Sprintf("Auto-join for threads in <#%s> has been set to `%t`.", channelOpt.ID, autoJoinOpt),
					Color:       0x00FF00,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
					},
				})
			} else if subcommand == "remove" {
				channelOpt := options[0].Options[0].ChannelValue(s)

				err := database.RemoveThreadAutomation(ctx, i.GuildID, channelOpt.ID)
				if err != nil {
					SendError(s, i, "Failed to remove thread automation config")
					return
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Thread Automation Removed",
					Description: fmt.Sprintf("Removed thread automation for <#%s>.", channelOpt.ID),
					Color:       0x00FF00,
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
