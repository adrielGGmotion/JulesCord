package commands

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// NewsPing creates the /newsping command.
func NewsPing(database *db.DB) *Command {
	if database == nil {
		return nil
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "newsping",
			Description:              "Configure automatic role pings for news channels",
			DefaultMemberPermissions: func() *int64 { p := int64(discordgo.PermissionAdministrator); return &p }(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Set a news ping rule",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The news/announcement channel",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildNews,
								discordgo.ChannelTypeGuildText,
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionRole,
							Name:        "role",
							Description: "The role to ping",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a news ping rule",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The news/announcement channel",
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
				return
			}

			subcommand := options[0].Name
			ctx := context.Background()

			if subcommand == "set" {
				channelID := ""
				roleID := ""
				for _, opt := range options[0].Options {
					if opt.Name == "channel" {
						channelID = opt.Value.(string)
					} else if opt.Name == "role" {
						roleID = opt.Value.(string)
					}
				}

				err := database.SetNewsPing(ctx, i.GuildID, channelID, roleID)
				if err != nil {
					SendError(s, i, "Failed to set news ping: "+err.Error())
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ News ping set! Whenever a message is posted in <#%s>, the <@&%s> role will be pinged.", channelID, roleID),
					},
				})

			} else if subcommand == "remove" {
				channelID := options[0].Options[0].Value.(string)

				err := database.RemoveNewsPing(ctx, i.GuildID, channelID)
				if err != nil {
					SendError(s, i, "Failed to remove news ping: "+err.Error())
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Removed news ping rule for <#%s>.", channelID),
					},
				})
			}
		},
	}
}
