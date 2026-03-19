package commands

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"julescord/internal/db"
)

// ThreadWatch creates the /threadwatch command.
func ThreadWatch(database *db.DB) *Command {
	if database == nil {
		return nil
	}

	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "threadwatch",
			Description: "Automatically join new threads created in a specific channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Start watching a channel for new threads",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to watch",
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
								discordgo.ChannelTypeGuildForum,
							},
						},
					},
				},
				{
					Name:        "remove",
					Description: "Stop watching a channel for new threads",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to stop watching",
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
			channelID := options[0].Options[0].Value.(string)
			ctx := context.Background()

			if subcommand == "add" {
				err := database.AddThreadWatcher(ctx, i.GuildID, channelID, i.Member.User.ID)
				if err != nil {
					SendError(s, i, "Failed to add thread watcher: "+err.Error())
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("👀 You are now watching <#%s>. You will automatically be added to any new threads created there.", channelID),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})

			} else if subcommand == "remove" {
				err := database.RemoveThreadWatcher(ctx, i.GuildID, channelID, i.Member.User.ID)
				if err != nil {
					SendError(s, i, "Failed to remove thread watcher: "+err.Error())
					return
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("🛑 You are no longer watching <#%s>.", channelID),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
		},
	}
}
