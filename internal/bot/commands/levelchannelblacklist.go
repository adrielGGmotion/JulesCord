package commands

import (
	"context"
	"fmt"
	"strings"

	"julescord/internal/db"

	"github.com/bwmarrin/discordgo"
)

// LevelChannelBlacklist returns the /levelchannelblacklist command.
func LevelChannelBlacklist(database *db.DB) *Command {
	defaultMemberPermissions := int64(discordgo.PermissionManageGuild)
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "levelchannelblacklist",
			Description:              "Manage the leveling channel blacklist",
			DefaultMemberPermissions: &defaultMemberPermissions,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Add a channel to the leveling blacklist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to blacklist from earning XP",
							Required:    true,
						},
					},
				},
				{
					Name:        "remove",
					Description: "Remove a channel from the leveling blacklist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to remove from the blacklist",
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List all channels in the leveling blacklist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.GuildID == "" {
				SendError(s, i, "This command can only be used in a server.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				return
			}
			subcommand := options[0]

			switch subcommand.Name {
			case "add":
				channelID := subcommand.Options[0].Value.(string)
				err := database.AddLevelingChannelBlacklist(context.Background(), i.GuildID, channelID)
				if err != nil {
					SendError(s, i, "Failed to add channel to the leveling blacklist.")
					return
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Added <#%s> to the leveling blacklist. Users will no longer earn XP in this channel.", channelID),
					},
				})
			case "remove":
				channelID := subcommand.Options[0].Value.(string)
				err := database.RemoveLevelingChannelBlacklist(context.Background(), i.GuildID, channelID)
				if err != nil {
					SendError(s, i, "Failed to remove channel from the leveling blacklist.")
					return
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("✅ Removed <#%s> from the leveling blacklist. Users can now earn XP in this channel.", channelID),
					},
				})
			case "list":
				blacklists, err := database.GetLevelingChannelBlacklists(context.Background(), i.GuildID)
				if err != nil {
					SendError(s, i, "Failed to retrieve the leveling blacklist.")
					return
				}

				if len(blacklists) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "There are currently no blacklisted channels in this server.",
						},
					})
					return
				}

				var channels []string
				for _, c := range blacklists {
					channels = append(channels, fmt.Sprintf("- <#%s>", c))
				}

				embed := &discordgo.MessageEmbed{
					Title:       "Leveling Channel Blacklist",
					Color:       0x00FF00,
					Description: strings.Join(channels, "\n"),
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
